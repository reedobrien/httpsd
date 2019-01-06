package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/reedobrien/acc"
	"github.com/reedobrien/httpsd/logging"
	zerolog "github.com/rs/zerolog"
	"golang.org/x/crypto/acme/autocert"
)

const (
	app       = "httpsd"
	drainTime = 10 * time.Second
)

var (
	// BuildDate is a build flag destination.
	BuildDate string
	// BuildVersion is a build flag destination.
	BuildVersion string
	// GitBranch is a build flag destination.
	GitBranch string
	// GitHash is a build flag destination.
	GitHash string
)

func main() {
	var (
		addrTLS = flag.String("addrTLS", ":443", "The address on which to listen for HTTPS.")

		bucket  = flag.String("bucket", "certbucket", "The bucket to cache certificates in.")
		region  = flag.String("region", "us-east-1", "The region where the bucket lives.")
		rootDir = flag.String("rootDir", "/var/www/", "The directory to use as the server root.")
		verbose = flag.Bool("verbose", false, "If logging should be verbose.")
		version = flag.Bool("version", false, "Display version and build information.")

		logger zerolog.Logger
	)

	flag.Parse()

	if *version {
		fmt.Printf(`
build date: %s
branch    : %s
hash      : %s
version   : %s
built with: %s
`[1:], BuildDate, GitBranch, GitHash, BuildVersion, runtime.Version())
		os.Exit(0)
	}

	// The channel where we wait for an exit signal.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	logger = logging.NewLogger(app, *verbose, nil)

	// Add the app version to the logger
	if BuildVersion != "" {
		logger = logger.With().
			Str("app_version", BuildVersion+"-"+GitHash).
			Logger()
	}

	// Setup the s3 service/client.
	sess := session.Must(session.NewSession(&aws.Config{Region: region}))
	svc := s3.New(sess)

	// Setup the https and http server.
	m := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(whitelist...),
		Cache:      acc.MustS3(svc, *bucket, ""),
	}

	fs := http.FileServer(http.Dir(*rootDir))
	fsLogger := logger.With().Str(
		"handler", "httpsd").Logger()
	loggingHandler := logging.NewAccessLogger(fs, fsLogger)

	// Create the servers.
	h443 := &http.Server{
		Addr:           *addrTLS,
		Handler:        loggingHandler,
		TLSConfig:      &tls.Config{GetCertificate: m.GetCertificate},
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	// Run the servers in goroutines.
	go func() {
		if err := h443.ListenAndServeTLS("", ""); err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("failed to start https server")
		}
	}()

	go func() {
		h := m.HTTPHandler(nil)
		h80 := &http.Server{
			Addr:           "0.0.0.0:80",
			Handler:        h,
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
			MaxHeaderBytes: 1 << 20,
		}
		if err := h80.ListenAndServe(); err != nil {
			logger.Fatal().Err(err).Msg("failed to start http server")

		}
	}()

	// Block on the stop channel. This will receive when a SIGINT or SIGTERM
	// is sent.
	<-stop

	// Create a context that will timeout.
	ctx, cancel := context.WithTimeout(context.Background(), drainTime)
	defer cancel()

	// Call shutdown on the servers with that context. This will close the
	// server and wait for current connections to finish -- for the duration
	// of the timeout.
	if err := h443.Shutdown(ctx); err != nil {
		logger.Error().Err(err).Msg("caught on h443.Shutdown")
	}
}
