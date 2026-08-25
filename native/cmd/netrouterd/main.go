// NetRouter daemon is the privileged management boundary in the bootable OS.
// It supports a root-only Unix socket and optional mTLS for desktop clients.
package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/ahmedha43/netrouter-manager/native/internal/config"
	"github.com/ahmedha43/netrouter-manager/native/internal/network"
	"github.com/ahmedha43/netrouter-manager/native/internal/runner"
	"github.com/ahmedha43/netrouter-manager/native/internal/service"
)

func main() {
	var unixSocket, listen, certificate, key, clientCA, configPath string
	flag.StringVar(&unixSocket, "unix-socket", "/run/netrouterd.sock", "root-only Unix management socket")
	flag.StringVar(&listen, "listen", "", "optional mTLS TCP listener, e.g. 0.0.0.0:8443")
	flag.StringVar(&certificate, "tls-cert", "", "PEM server certificate for mTLS")
	flag.StringVar(&key, "tls-key", "", "PEM server private key for mTLS")
	flag.StringVar(&clientCA, "tls-client-ca", "", "PEM CA used to verify manager client certificates")
	flag.StringVar(&configPath, "config", "/etc/netrouter/config.json", "path to persistent config.json")
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// 1. Initialize persistent configuration store
	cfgStore := config.NewStore(configPath)
	cfg, err := cfgStore.Load()
	if err != nil {
		log.Printf("warning: loading config failed, using defaults: %v", err)
		cfg = config.Default()
	}

	netManager := network.NewManager(runner.OSExecutor{})

	// 2. Start Neighbor Discovery Beacon (UDP: 8444)
	go network.StartDiscoveryServer(ctx, cfg.System.Identity, "v0.1.3")

	server := service.NewWithConfig(netManager, cfgStore)
	unixListener, err := server.ListenUnix(unixSocket)
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(unixSocket)
	go serveOrLog(ctx, server, unixListener, "unix")

	if listen != "" {
		tlsCfg, err := loadMTLS(certificate, key, clientCA)
		if err != nil {
			log.Fatal(err)
		}
		tcpListener, err := tls.Listen("tcp", listen, tlsCfg)
		if err != nil {
			log.Fatal(fmt.Errorf("listen with mTLS: %w", err))
		}
		go serveOrLog(ctx, server, tcpListener, "mtls")
	}

	log.Printf("netrouterd started: Unix socket %s, Identity: %s", unixSocket, cfg.System.Identity)
	<-ctx.Done()
	log.Print("netrouterd stopping")
}

func serveOrLog(ctx context.Context, server *service.Server, listener net.Listener, name string) {
	if err := server.Serve(ctx, listener); err != nil {
		log.Printf("%s listener stopped: %v", name, err)
	}
}

func loadMTLS(certificatePath, keyPath, clientCAPath string) (*tls.Config, error) {
	if certificatePath == "" || keyPath == "" || clientCAPath == "" {
		return nil, fmt.Errorf("mTLS listener requires --tls-cert, --tls-key, and --tls-client-ca")
	}
	certificate, err := tls.LoadX509KeyPair(certificatePath, keyPath)
	if err != nil {
		return nil, fmt.Errorf("load server certificate: %w", err)
	}
	caPEM, err := os.ReadFile(filepath.Clean(clientCAPath))
	if err != nil {
		return nil, fmt.Errorf("read client CA: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("parse client CA")
	}
	return &tls.Config{
		MinVersion:   tls.VersionTLS13,
		Certificates: []tls.Certificate{certificate},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    pool,
	}, nil
}
