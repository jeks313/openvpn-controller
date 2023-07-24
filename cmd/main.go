package main

import (
	"context"
	"embed"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/mux"
	"github.com/jeks313/go-mongo-slow-queries/pkg/options"
	"github.com/jeks313/go-mongo-slow-queries/pkg/server"
	flags "github.com/jessevdk/go-flags"
)

//go:embed templates
var templateFS embed.FS

// OpenVPNOpts is all the mongo specific connection options
type OpenVPNOpts struct {
}

var opts struct {
	Port        int                        `long:"port" env:"PORT" default:"9172" description:"port number to listen on"`
	Application options.ApplicationOptions `group:"Default Application Options"`
	OpenVPN     OpenVPNOpts                `group:"OpenvVPN Options"`
}

var loggingLevel = new(slog.LevelVar)

func main() {
	history := NewLogHistory(100)
	mw := io.MultiWriter(os.Stderr, history)
	log := slog.New(slog.NewTextHandler(mw, &slog.HandlerOptions{Level: loggingLevel}))
	slog.SetDefault(log)

	_, err := flags.ParseArgs(&opts, os.Args[1:])
	if err != nil {
		log.Error("failed to parse command line arguments", "error", err)
		os.Exit(1)
	}

	if opts.Application.Debug {
		loggingLevel.Set(slog.LevelDebug)
	}

	if opts.Application.Version {
		options.LogVersion()
		os.Exit(0)
	}

	// router
	r := mux.NewRouter()
	// r.Use(handlers.CompressHandler)

	// setup logging
	server.Log(r)

	// default end points
	server.Profiling(r, "/debug/pprof")

	// metrics
	server.Metrics(r, "/metrics")

	listen := fmt.Sprintf(":%d", opts.Port)

	srv := &http.Server{
		Handler:      r,
		Addr:         listen,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second}

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
			srv.Shutdown(ctx)
		case <-ctx.Done():
		}
	}()

	var checks []Checker

	sandboxPing := NewPing("vcr1sandbox1.absolute.com")
	sandboxDNS := NewDNS("vcr1sandbox1.absolute.com")

	checks = append(checks, sandboxPing)
	checks = append(checks, sandboxDNS)

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
	r.HandleFunc("/", GetIndex(display, checks))
	r.HandleFunc("/connect", PostConnect(vpn))
	r.HandleFunc("/log", GetLog(history))
	r.HandleFunc("/updatews", GetUpdateWs(display, history))
	r.HandleFunc("/logstream", GetLogStream(history))
	r.HandleFunc("/vpn", GetVPN(display, vpn))

	log.Info("started server ...", "port", opts.Port)

	if err = srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error("failed to start http server", "error", err)
		os.Exit(1)
	}

	log.Info("stopped")
}
