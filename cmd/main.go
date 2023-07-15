package main

import (
	"context"
	"embed"
	"fmt"
	stdlog "log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/jeks313/go-mongo-slow-queries/pkg/options"
	"github.com/jeks313/go-mongo-slow-queries/pkg/server"
	flags "github.com/jessevdk/go-flags"
	"github.com/rs/zerolog"
)

//go:embed templates
var templateFS embed.FS

// MongoOpts is all the mongo specific connection options
type OpenVPNOpts struct {
}

var opts struct {
	Port        int                        `long:"port" env:"PORT" default:"9172" description:"port number to listen on"`
	Application options.ApplicationOptions `group:"Default Application Options"`
	OpenVPN     OpenVPNOpts                `group:"OpenvVPN Options"`
}

func main() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log := zerolog.New(os.Stdout).With().Timestamp().Logger()
	stdlog.SetFlags(0)
	stdlog.SetOutput(log)

	_, err := flags.ParseArgs(&opts, os.Args[1:])
	if err != nil {
		log.Error().Err(err).Msg("failed to parse command line arguments")
		os.Exit(1)
	}

	if opts.Application.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	if opts.Application.Version {
		options.LogVersion()
		os.Exit(0)
	}

	// router
	r := mux.NewRouter()
	r.Use(handlers.CompressHandler)

	// setup logging
	server.Log(r, log)

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
			log.Info().Msg("interrupt, shutting down ...")
			srv.Shutdown(ctx)
		case <-ctx.Done():
		}
	}()

	r.HandleFunc("/status", GetVPNStatus)
	r.HandleFunc("/", GetIndex)
	//r.HandleFunc("/running", mongoslow.RunningQueryTableHandler(slow))
	//r.HandleFunc("/history.json", mongoslow.HistoryQueryHandler(slow))
	//r.HandleFunc("/history", mongoslow.HistoryQueryTableHandler(slow))

	log.Info().Int("port", opts.Port).Msg("started server ...")

	if err = srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error().Err(err).Msg("failed to start http server")
		os.Exit(1)
	}

	log.Info().Msg("stopped")
}
