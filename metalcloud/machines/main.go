package main

import (
	"context"
	"crypto/x509"
	"crypto/x509/pkix"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/justinsb/packages/kinspire/pkg/certs"
)

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	machineName := ""
	clientName := "client1"
	flag.StringVar(&machineName, "name", machineName, "name of machine")
	flag.Parse()

	if machineName == "" {
		return fmt.Errorf("must specify --name with name of machine")
	}

	outDir := machineName
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("making directory %q: %w", outDir, err)
	}

	{
		caTemplate := x509.Certificate{
			Subject: pkix.Name{
				CommonName: "server-ca",
			},
			NotBefore: time.Now().Add(-10 * time.Minute),
			NotAfter:  time.Now().AddDate(10, 0, 0),

			// KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
			// ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth | x509.ExtKeyUsageServerAuth},
			IsCA:                  true,
			BasicConstraintsValid: true,
		}

		caCert, caKey, err := certs.CreateCertificate(caTemplate, nil, nil)
		if err != nil {
			return err
		}

		caCertBytes, err := certs.PEMEncodeCertificate(caCert)
		if err != nil {
			return err
		}

		caKeyBytes, err := certs.EncodePrivateKey(caKey)
		if err != nil {
			return err
		}

		serverTemplate := x509.Certificate{
			Subject: pkix.Name{
				CommonName: machineName,
			},
			DNSNames:  []string{machineName},
			NotBefore: time.Now().Add(-10 * time.Minute),
			NotAfter:  time.Now().AddDate(10, 0, 0),

			KeyUsage:              x509.KeyUsageDigitalSignature,
			ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			IsCA:                  false,
			BasicConstraintsValid: true,
		}

		serverCert, serverKey, err := certs.CreateCertificate(serverTemplate, caCert, caKey)
		if err != nil {
			return err
		}

		// There might be a better way to do this, but this is pretty robust
		serverCertBytes, err := certs.PEMEncodeCertificate(serverCert)
		if err != nil {
			return err
		}
		serverKeyBytes, err := certs.EncodePrivateKey(serverKey)
		if err != nil {
			return err
		}

		if err := os.WriteFile(filepath.Join(outDir, "ca.crt"), caCertBytes, 0644); err != nil {
			return fmt.Errorf("error writing ca.crt: %w", err)
		}
		if err := os.WriteFile(filepath.Join(outDir, "ca.key"), caKeyBytes, 0600); err != nil {
			return fmt.Errorf("error writing ca.key: %w", err)
		}

		if err := os.WriteFile(filepath.Join(outDir, "server.crt"), serverCertBytes, 0644); err != nil {
			return fmt.Errorf("error writing server.crt: %w", err)
		}
		if err := os.WriteFile(filepath.Join(outDir, "server.key"), serverKeyBytes, 0600); err != nil {
			return fmt.Errorf("error writing server.key: %w", err)
		}
	}

	{
		clientCATemplate := x509.Certificate{
			Subject: pkix.Name{
				CommonName: "client-ca",
			},
			NotBefore: time.Now().Add(-10 * time.Minute),
			NotAfter:  time.Now().AddDate(10, 0, 0),

			// KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
			// ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth | x509.ExtKeyUsageServerAuth},
			IsCA:                  true,
			BasicConstraintsValid: true,
		}

		caCert, caKey, err := certs.CreateCertificate(clientCATemplate, nil, nil)
		if err != nil {
			return err
		}

		caCertBytes, err := certs.PEMEncodeCertificate(caCert)
		if err != nil {
			return err
		}

		caKeyBytes, err := certs.EncodePrivateKey(caKey)
		if err != nil {
			return err
		}

		clientTemplate := x509.Certificate{
			Subject: pkix.Name{
				CommonName: clientName,
			},
			NotBefore: time.Now().Add(-10 * time.Minute),
			NotAfter:  time.Now().AddDate(10, 0, 0),

			KeyUsage:              x509.KeyUsageDigitalSignature,
			ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			IsCA:                  false,
			BasicConstraintsValid: true,
		}

		clientCert, clientKey, err := certs.CreateCertificate(clientTemplate, caCert, caKey)
		if err != nil {
			return err
		}

		// There might be a better way to do this, but this is pretty robust
		clientCertBytes, err := certs.PEMEncodeCertificate(clientCert)
		if err != nil {
			return err
		}
		clientKeyBytes, err := certs.EncodePrivateKey(clientKey)
		if err != nil {
			return err
		}

		if err := os.WriteFile(filepath.Join(outDir, "client-ca.crt"), caCertBytes, 0644); err != nil {
			return fmt.Errorf("error writing client-ca.crt: %w", err)
		}
		if err := os.WriteFile(filepath.Join(outDir, "client-ca.key"), caKeyBytes, 0600); err != nil {
			return fmt.Errorf("error writing client-ca.key: %w", err)
		}

		if err := os.WriteFile(filepath.Join(outDir, "client.crt"), clientCertBytes, 0644); err != nil {
			return fmt.Errorf("error writing client.crt: %w", err)
		}
		if err := os.WriteFile(filepath.Join(outDir, "client.key"), clientKeyBytes, 0600); err != nil {
			return fmt.Errorf("error writing client.key: %w", err)
		}
	}

	return nil
}
