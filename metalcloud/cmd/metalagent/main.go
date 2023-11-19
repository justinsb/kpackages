package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	agentv1 "github.com/justinsb/metalcloud/agent/v1"
	"github.com/justinsb/metalcloud/pkg/kmod"
	"github.com/justinsb/metalcloud/pkg/server"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
)

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	if err := bootstrapMachine(ctx); err != nil {
		log.Printf("error from bootstrap: %v", err)
	}

	go func() {
		for {
			err := poll(ctx)
			if err != nil {
				log.Printf("poll gave error %v", err)
			}
			time.Sleep(1 * time.Minute)
		}
	}()

	listenOn := "0.0.0.0:8080"
	listener, err := net.Listen("tcp", listenOn)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", listenOn, err)
	}

	grpcServer := grpc.NewServer()
	agentv1.RegisterAgentServiceServer(grpcServer, server.NewAgentService())
	log.Println("Listening on", listenOn)
	if err := grpcServer.Serve(listener); err != nil {
		return fmt.Errorf("failed to serve gRPC server: %w", err)
	}

	return nil
}

func mountDev() error {
	dir := "/dev"

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to MkdirAll(%q): %w", dir, err)
	}

	if err := unix.Mount("udev", dir, "devtmpfs", 0, ""); err != nil {
		return fmt.Errorf("failed to mount %s: %w", dir, err)
	}
	return nil
}

func mountSys() error {
	dir := "/sys"

	if err := os.MkdirAll(dir, 0555); err != nil {
		return fmt.Errorf("failed to MkdirAll(%q): %w", dir, err)
	}

	if err := unix.Mount("sysfs", dir, "sysfs", 0, ""); err != nil {
		return fmt.Errorf("failed to mount %s: %w", dir, err)
	}
	return nil
}

func mountProc() error {
	dir := "/proc"

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to MkdirAll(%q): %w", dir, err)
	}

	if err := unix.Mount("proc", dir, "proc", 0, ""); err != nil {
		return fmt.Errorf("failed to mount /proc: %w", err)
	}
	return nil
}

func poll(ctx context.Context) error {
	cmdline, err := os.ReadFile("/proc/cmdline")
	if err != nil {
		return fmt.Errorf("error reading /proc/cmdline: %w", err)
	}
	log.Printf("/proc/cmdline is %q", string(cmdline))

	if err := os.MkdirAll("etc", 0755); err != nil {
		return fmt.Errorf("failed to MkdirAll(%q): %w", "/etc", err)
	}

	kernelVersion, err := kmod.UName()
	if err != nil {
		return err
	}

	{
		devicesDir := "/sys/devices"

		var modaliases []string

		if err := filepath.WalkDir(devicesDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			name := filepath.Base(path)
			if name == "modalias" {
				b, err := os.ReadFile(path)
				if err != nil {
					return fmt.Errorf("error reading %q: %w", path, err)
				}
				modaliases = append(modaliases, string(b))
				log.Printf("modalias %v is %v", path, string(b))
			}
			return nil
		}); err != nil {
			return fmt.Errorf("error walking %q: %w", devicesDir, err)
		}

		log.Printf("modaliases is %v", modaliases)

		loadModules := make(map[string]bool)
		{

			// TODO: load modules.aliases once

			aliases := filepath.Join("/lib", "modules", kernelVersion, "modules.alias")
			f, err := os.Open(aliases)
			if err != nil {
				return fmt.Errorf("error opening %q: %w", aliases, err)
			}
			defer f.Close()

			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "#") {
					continue
				}

				tokens := strings.Fields(line)
				if len(tokens) != 3 {
					log.Printf("unexpected line in %q: %q", aliases, line)
					continue
				}

				for _, modalias := range modaliases {
					if matched, _ := filepath.Match(tokens[1], modalias); matched {
						module := tokens[2]
						log.Printf("found module %q for %q", module, modalias)
						loadModules[module] = true
					}
				}
			}
			if err := scanner.Err(); err != nil {
				return fmt.Errorf("error reading %q: %w", aliases, err)
			}
		}

		var moduleNames []string
		for module := range loadModules {
			moduleNames = append(moduleNames, module)
		}
		dep, err := kmod.LoadModulesDep()
		if err != nil {
			return err
		}
		log.Printf("loading modules %v", moduleNames)
		if err := kmod.LoadModules(moduleNames, dep); err != nil {
			log.Printf("failed to load modules: %v", err)
		}
	}

	if _, err := os.Stat("/bin/ip"); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("error getting stat on /bin/ip: %w", err)
		}
		if err := runCommand(ctx, "/bin/busybox", "--install", "-s", "/bin"); err != nil {
			return err
		}
	}

	if err := runIP(ctx, "link"); err != nil {
		return err
	}

	if err := runIP(ctx, "addr"); err != nil {
		return err
	}

	interfaces, err := net.Interfaces()
	if err != nil {
		return fmt.Errorf("error getting interfaces: %w", err)
	}
	for _, iface := range interfaces {
		if (iface.Flags & net.FlagUp) == 0 {
			log.Printf("setting iface %q to up", iface.Name)
			if err := runIP(ctx, "link", "set", "dev", iface.Name, "up"); err != nil {
				return err
			}
		} else {
			log.Printf("iface %q is up", iface.Name)
		}
	}

	if err := runDHCP(ctx); err != nil {
		return err
	}

	return nil
}

func runCommand(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error from %s: %w", args, err)
	}
	log.Printf("%s succeeded", args)
	return nil
}

func runIP(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "/bin/ip", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error from ip %s: %w", args, err)
	}
	log.Printf("ip %s succeeded", args)
	return nil
}

func runDHCP(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "/bin/udhcpc", "--script=/usr/share/udhcpc/simple.script")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error from udhcpc: %w", err)
	}
	return nil
}

func bootstrapMachine(ctx context.Context) error {
	// kernelVersion, err := kmod.UName()
	// if err != nil {
	// 	return err
	// }

	// modules := []string{
	// 	// USB keyboards
	// 	"/lib/modules/" + kernelVersion + "/kernel/drivers/usb/common/usb-common.ko",
	// 	"/lib/modules/" + kernelVersion + "/kernel/drivers/usb/core/usbcore.ko",
	// 	"/lib/modules/" + kernelVersion + "/kernel/drivers/hid/hid.ko",
	// 	"/lib/modules/" + kernelVersion + "/kernel/drivers/hid/hid-generic.ko",
	// 	"/lib/modules/" + kernelVersion + "/kernel/drivers/hid/usbhid/usbhid.ko",
	// 	"/lib/modules/" + kernelVersion + "/kernel/drivers/input/evdev.ko",
	// 	"/lib/modules/" + kernelVersion + "/kernel/drivers/usb/host/xhci-hcd.ko",
	// 	"/lib/modules/" + kernelVersion + "/kernel/drivers/usb/host/xhci-pci.ko",

	// 	// 8169
	// 	"/lib/modules/" + kernelVersion + "/kernel/drivers/net/phy/libphy.ko",
	// 	"/lib/modules/" + kernelVersion + "/kernel/drivers/net/phy/realtek.ko",
	// 	"/lib/modules/" + kernelVersion + "/kernel/drivers/net/phy/mdio_devres.ko",
	// 	"/lib/modules/" + kernelVersion + "/kernel/drivers/net/ethernet/realtek/r8169.ko",

	// 	// 8139
	// 	"/lib/modules/" + kernelVersion + "/kernel/drivers/net/mii.ko",
	// 	"/lib/modules/" + kernelVersion + "/kernel/drivers/net/ethernet/realtek/8139cp.ko",
	// }
	// for _, module := range modules {
	// 	if err := kmod.LoadModuleByPath(module, "", 0); err != nil {
	// 		return err
	// 	}
	// 	log.Printf("loaded module %q\n", module)
	// }

	if err := mountDev(); err != nil {
		return err
	}

	if err := mountSys(); err != nil {
		return err
	}

	if err := mountProc(); err != nil {
		return err
	}

	return nil
}
