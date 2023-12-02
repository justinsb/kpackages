package kmod

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

func UName() (string, error) {
	u := syscall.Utsname{}
	if err := syscall.Uname(&u); err != nil {
		return "", fmt.Errorf("error from uname: %w", err)
	}

	var bytes []byte
	for i := 0; ; i++ {
		if u.Release[i] == 0 {
			break
		}
		bytes = append(bytes, byte(u.Release[i]))
	}
	kernelVersion := string(bytes)
	log.Printf("kernel version is %v", kernelVersion)
	return kernelVersion, nil
}

func LoadModuleByPath(modulePath string, params string, flags int) error {
	log.Printf("loading module %q", modulePath)
	f, err := os.Open(modulePath)
	if err != nil {
		return fmt.Errorf("error opening file %q: %w", modulePath, err)
	}
	defer f.Close()

	if err := unix.FinitModule(int(f.Fd()), params, flags); err != nil {
		if errno, ok := err.(syscall.Errno); ok {
			switch errno {
			case syscall.EEXIST:
				log.Printf("ignoring EEXIST from finit(%q)", modulePath)
				return nil
			}
		}
		return fmt.Errorf("error from finit(%q, %q, %v): %w", modulePath, params, flags, err)
	}
	return nil
}

type ModulesDep struct {
	baseDir string
	modules map[string]*moduleInfo
}

type moduleInfo struct {
	path         string
	dependencies []string
}

func LoadModulesDep() (*ModulesDep, error) {
	kernelVersion, err := UName()
	if err != nil {
		return nil, err
	}

	modules := make(map[string]*moduleInfo)

	baseDir := filepath.Join("/usr/lib", "modules", kernelVersion)
	p := filepath.Join(baseDir, "modules.dep")
	f, err := os.Open(p)
	if err != nil {
		return nil, fmt.Errorf("error opening %q: %w", p, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		tokens := strings.Fields(line)
		if !strings.HasSuffix(tokens[0], ":") {
			log.Printf("unexpected line (no colon) in %q: %q", p, line)
			continue
		}

		modulePath := strings.TrimSuffix(tokens[0], ":")
		moduleName := filepath.Base(modulePath)
		moduleName = strings.TrimSuffix(moduleName, ".ko")

		// TODO: Modules map - to _; is this hard-coded?
		moduleName = strings.ReplaceAll(moduleName, "-", "_")

		info := &moduleInfo{
			path: modulePath,
		}
		for _, dep := range tokens[1:] {
			name := filepath.Base(dep)
			name = strings.TrimSuffix(name, ".ko")
			name = strings.ReplaceAll(name, "-", "_")

			info.dependencies = append(info.dependencies, name)
		}
		modules[moduleName] = info
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading %q: %w", p, err)
	}

	return &ModulesDep{baseDir: baseDir, modules: modules}, nil
}

func buildLoadedModules() (map[string]bool, error) {
	p := filepath.Join("/proc", "modules")
	f, err := os.Open(p)
	if err != nil {
		return nil, fmt.Errorf("error opening %q: %w", p, err)
	}
	defer f.Close()

	modules := make(map[string]bool)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		tokens := strings.Fields(line)
		moduleName := tokens[0]
		log.Printf("module %q already loaded", moduleName)
		modules[moduleName] = true
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading %q: %w", p, err)
	}

	return modules, nil
}

type moduleLoadPlan struct {
	deps *ModulesDep

	done map[string]bool

	queue map[string]*moduleInfo
}

func (p *moduleLoadPlan) addToPlan(moduleName string) error {
	if p.done[moduleName] {
		// Already loaded
		return nil
	}

	modInfo, found := p.deps.modules[moduleName]
	if !found {
		log.Printf("module %q not known", moduleName)
		return fmt.Errorf("module %q not known", moduleName)
	}

	if _, found := p.queue[moduleName]; found {
		// Already in queue
		return nil
	}

	for _, dep := range modInfo.dependencies {
		// We can't load a dependency, so we can't load this one
		if err := p.addToPlan(dep); err != nil {
			return err
		}
	}

	p.queue[moduleName] = modInfo

	return nil
}

type ModuleLoadFilterFunc func(moduleName string) bool

func LoadModules(moduleNames []string, deps *ModulesDep, shouldLoad ModuleLoadFilterFunc) error {
	plan := &moduleLoadPlan{
		deps:  deps,
		done:  make(map[string]bool),
		queue: make(map[string]*moduleInfo),
	}

	loadedModules, err := buildLoadedModules()
	if err != nil {
		return err
	}
	for module := range loadedModules {
		plan.done[module] = true
	}

	for _, moduleName := range moduleNames {
		if !shouldLoad(moduleName) {
			log.Printf("skipping module %q", moduleName)
			continue
		}
		if err := plan.addToPlan(moduleName); err != nil {
			log.Printf("will not be able to load module %q: %v", moduleName, err)
		}
	}

	for {
		progress := false
		allDone := true

		for moduleName, info := range plan.queue {
			if plan.done[moduleName] {
				continue
			}

			if !shouldLoad(moduleName) {
				log.Printf("skipping module %q", moduleName)
				continue
			}

			allDone = false

			dependenciesLoaded := true
			for _, dep := range info.dependencies {
				if !plan.done[dep] {
					log.Printf("cannot yet load %q, waiting on %q", moduleName, dep)
					dependenciesLoaded = false
				}
			}

			if !dependenciesLoaded {
				continue
			}

			fullPath := filepath.Join(deps.baseDir, info.path)
			if err := LoadModuleByPath(fullPath, "", 0); err != nil {
				log.Printf("error loading %q", fullPath)
				log.Printf("  error is %v", err)
				//return fmt.Errorf("error loading %q: %w", fullPath, err)
				continue
			}
			log.Printf("loaded module %q", info.path)
			plan.done[moduleName] = true
			progress = true
		}

		if allDone {
			return nil
		}

		if !progress {
			return fmt.Errorf("not making progress loading modules")
		}
	}
}
