package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"

	agentv1 "github.com/justinsb/metalcloud/agent/v1"
	"github.com/justinsb/metalcloud/pkg/kmod"
	"github.com/justinsb/metalcloud/pkg/server"
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

func bootstrapMachine(ctx context.Context) error {
	modules := []string{
		"/lib/modules/5.10.0-18-amd64/kernel/drivers/usb/common/usb-common.ko",
		"/lib/modules/5.10.0-18-amd64/kernel/drivers/usb/core/usbcore.ko",
		"/lib/modules/5.10.0-18-amd64/kernel/drivers/hid/hid.ko",
		"/lib/modules/5.10.0-18-amd64/kernel/drivers/hid/hid-generic.ko",
		"/lib/modules/5.10.0-18-amd64/kernel/drivers/hid/usbhid/usbhid.ko",
		"/lib/modules/5.10.0-18-amd64/kernel/drivers/input/evdev.ko",
		"/lib/modules/5.10.0-18-amd64/kernel/drivers/usb/host/xhci-hcd.ko",
		"/lib/modules/5.10.0-18-amd64/kernel/drivers/usb/host/xhci-pci.ko",
	}
	for _, module := range modules {
		if err := kmod.LoadModule(module, "", 0); err != nil {
			return err
		}
		log.Printf("loaded module %q\n", module)
	}

	return nil
}
