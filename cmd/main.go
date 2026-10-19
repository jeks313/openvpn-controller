package main

import (
	"context"
	"embed"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"runtime/debug"
	"strconv"
	"time"
)

//go:embed templates
var templateFS embed.FS

var opts struct {
	Port    int
	Address string
	Debug   bool
	Version bool
}

// envOr returns the environment variable key, or def when it is unset.
func envOr(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}

var loggingLevel = new(slog.LevelVar)

func main() {
	history := NewLogHistory(100)
	mw := io.MultiWriter(os.Stderr, history)
	log := slog.New(slog.NewTextHandler(mw, &slog.HandlerOptions{Level: loggingLevel}))
	slog.SetDefault(log)

	port, err := strconv.Atoi(envOr("PORT", "9172"))
	if err != nil {
		log.Error("invalid PORT", "error", err)
		os.Exit(1)
	}
	debugEnv, err := strconv.ParseBool(envOr("DEBUG", "false"))
	if err != nil {
		log.Error("invalid DEBUG", "error", err)
		os.Exit(1)
	}
	flag.IntVar(&opts.Port, "port", port, "port number to listen on (env PORT)")
	flag.StringVar(&opts.Address, "address", envOr("ADDRESS", ""), "address to bind to, default all (env ADDRESS)")
	flag.BoolVar(&opts.Debug, "debug", debugEnv, "enable debug logging level (env DEBUG)")
	flag.BoolVar(&opts.Version, "version", false, "print the build's version control info and exit")
	flag.Parse()

	if opts.Debug {
		loggingLevel.Set(slog.LevelDebug)
	}

	if opts.Version {
		if bi, ok := debug.ReadBuildInfo(); ok {
			for _, s := range bi.Settings {
				fmt.Printf("%s=%s\n", s.Key, s.Value)
			}
		}
		os.Exit(0)
	}

	r := http.NewServeMux()

	r.HandleFunc("/debug/pprof/", pprof.Index)
	r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	r.HandleFunc("/debug/pprof/profile", pprof.Profile)
	r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	r.HandleFunc("/debug/pprof/trace", pprof.Trace)

	r.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("ok"))
	})

	listen := fmt.Sprintf("%s:%d", opts.Address, opts.Port)

	srv := &http.Server{
		Handler:      logRequests(r),
		Addr:         listen,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	defer func() {
		signal.Stop(c)
		cancel()
	}()

	go func() {
		select {
		case <-c:
			cancel()
			log.Info("interrupt, shutting down ...")
			if err := srv.Shutdown(ctx); err != nil {
				log.Error("shutdown", "error", err)
			}
		case <-ctx.Done():
		}
	}()

	var checks []Checker

	sandboxPing := NewPing("10.13.132.150")
	sandboxDNS := NewDNS("aws1grafana1.absolute.com")

	checks = append(checks, sandboxPing, sandboxDNS)

	vpn := NewOpenVPN("")
	display, err := NewDisplay("templates/openvpn.html")

	if err != nil {
		slog.Error("failed to initialize display", "error", err)
		os.Exit(1)
	}

	go func() {
		var nw *NullWriter
		for {
			for _, check := range checks {
				err := check.Check()
				if err != nil {
					slog.Error("failed to run check", "error", err)
				}
			}
			display.VPNStatus(nw, checks)
			display.VPN(nw, vpn)
			time.Sleep(5 * time.Second)
		}
	}()

	r.HandleFunc("/status", GetVPNStatus(display, checks))
	r.HandleFunc("/{$}", GetIndex(display, checks))
	r.HandleFunc("/connect", PostConnect(vpn))
	r.HandleFunc("/log", GetLog(history))
	r.HandleFunc("/updates", GetUpdates(display, history))
	r.HandleFunc("/logstream", GetLogStream(history))
	r.HandleFunc("/vpn", GetVPN(display, vpn))

	log.Info("started server ...", "address", listen)

	err = srv.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("failed to start http server", "error", err)
		os.Exit(1)
	}

	log.Info("stopped")
}
