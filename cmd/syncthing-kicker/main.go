package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/rcarmo/syncthing-kicker/internal/app"
	"github.com/rcarmo/syncthing-kicker/internal/syncthing"
)

func main() {
	_ = godotenv.Load() // best-effort; do not override env

	check := flag.Bool("check", false, "Check Syncthing folder status and exit")
	flag.Parse()

	logger := log.New(os.Stdout, "", log.LstdFlags)

	settings, err := app.LoadSettingsFromEnv()
	if err != nil {
		logger.Printf("Failed to load settings: %v", err)
		os.Exit(1)
	}

	client, err := syncthing.NewClient(settings.APIURL, settings.APIKey, syncthing.ClientOptions{
		VerifyTLS:      settings.VerifyTLS,
		RequestTimeout: seconds(settings.RequestTimeout),
	})
	if err != nil {
		logger.Printf("Failed to initialize client: %v", err)
		os.Exit(1)
	}

	svc := &app.Service{Settings: settings, Client: client, Logger: logger}

	if *check {
		if err := svc.CheckOnce(context.Background()); err != nil {
			logger.Printf("Check failed: %v", err)
			os.Exit(1)
		}
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := svc.Run(ctx); err != nil {
		if err == context.Canceled {
			return
		}
		logger.Printf("Service stopped: %v", err)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func seconds(v float64) time.Duration {
	if v <= 0 {
		return 0
	}
	return time.Duration(v * float64(time.Second))
}
