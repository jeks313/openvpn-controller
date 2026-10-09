package main

import (
	"crypto/tls"
	"errors"
	"log/slog"
	"os"

	"github.com/foomo/simplecert"
	"github.com/foomo/tlsconfig"
)

// selfManagedTLS obtains and renews its own Let's Encrypt certificate via a
// Cloudflare DNS-01 challenge. Only used when running directly on a host; behind
// a TLS-terminating proxy (k3s/Traefik) run with --plain-http instead.
//
// needs CLOUDFLARE_EMAIL and CLOUDFLARE_API_KEY set in the environment
func selfManagedTLS() (*tls.Config, error) {
	cfg := simplecert.Default
	cfg.Domains = []string{"spiff.hyde.ca"}
	cfg.CacheDir = "/etc/letsencrypt/live"
	cfg.SSLEmail = "chris@hyde.ca"
	cfg.DNSProvider = "cloudflare"
	cfg.TLSAddress = ""
	cfg.HTTPAddress = ""

	if os.Getenv("CLOUDFLARE_EMAIL") == "" {
		slog.Error("please set CLOUDFLARE_EMAIL environment variable")
		return nil, errors.New("CLOUDFLARE_EMAIL not set")
	}

	if os.Getenv("CLOUDFLARE_API_KEY") == "" {
		slog.Error("please set CLOUDFLARE_API_KEY environment variable")
		return nil, errors.New("CLOUDFLARE_API_KEY not set")
	}

	certReloader, err := simplecert.Init(cfg, nil)
	if err != nil {
		slog.Error("failed to initialize cert reloader", "error", err)
		return nil, err
	}

	tlsConf := tlsconfig.NewServerTLSConfig(tlsconfig.TLSModeServerStrict)
	tlsConf.GetCertificate = certReloader.GetCertificateFunc()
	return tlsConf, nil
}
