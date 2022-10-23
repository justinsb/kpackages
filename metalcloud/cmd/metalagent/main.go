package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
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

	if err := os.MkdirAll("proc", 0755); err != nil {
		return fmt.Errorf("failed to MkdirAll(%q): %w", "/proc", err)
	}

	if err := unix.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		return fmt.Errorf("failed to mount /proc: %w", err)
	}

	if err := runCommand(ctx, "/bin/busybox", "--install", "-s", "/bin"); err != nil {
		return err
	}

	if err := runIP(ctx, "link"); err != nil {
		return err
	}

	if err := runIP(ctx, "addr"); err != nil {
		return err
	}

	if err := runIP(ctx, "link", "set", "dev", "eth0", "up"); err != nil {
		return err
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
	modules := []string{
		// USB keyboards
		"/lib/modules/5.10.0-18-amd64/kernel/drivers/usb/common/usb-common.ko",
		"/lib/modules/5.10.0-18-amd64/kernel/drivers/usb/core/usbcore.ko",
		"/lib/modules/5.10.0-18-amd64/kernel/drivers/hid/hid.ko",
		"/lib/modules/5.10.0-18-amd64/kernel/drivers/hid/hid-generic.ko",
		"/lib/modules/5.10.0-18-amd64/kernel/drivers/hid/usbhid/usbhid.ko",
		"/lib/modules/5.10.0-18-amd64/kernel/drivers/input/evdev.ko",
		"/lib/modules/5.10.0-18-amd64/kernel/drivers/usb/host/xhci-hcd.ko",
		"/lib/modules/5.10.0-18-amd64/kernel/drivers/usb/host/xhci-pci.ko",

		// 8169
		"/lib/modules/5.10.0-18-amd64/kernel/drivers/net/phy/libphy.ko",
		"/lib/modules/5.10.0-18-amd64/kernel/drivers/net/phy/realtek.ko",
		"/lib/modules/5.10.0-18-amd64/kernel/drivers/net/phy/mdio_devres.ko",
		"/lib/modules/5.10.0-18-amd64/kernel/drivers/net/ethernet/realtek/r8169.ko",

		// 8139
		"/lib/modules/5.10.0-18-amd64/kernel/drivers/net/mii.ko",
		"/lib/modules/5.10.0-18-amd64/kernel/drivers/net/ethernet/realtek/8139cp.ko",
	}
	for _, module := range modules {
		if err := kmod.LoadModule(module, "", 0); err != nil {
			return err
		}
		log.Printf("loaded module %q\n", module)
	}

	if err := mountDev(); err != nil {
		return err
	}

	if err := mountProc(); err != nil {
		return err
	}

	return nil
}
