package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Leopold1975/banners_control/internal/banners/app"
	"github.com/Leopold1975/banners_control/internal/pkg/config"
)

func main() {
	var configPath string

	flag.StringVar(&configPath, "config", "", "path to configuration file")
	flag.Parse()

	cfg, err := config.New(configPath)
	if err != nil {
		log.Fatal(err)
	}

	interruptSignals := []os.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP}

	ctx, cancel := signal.NotifyContext(context.Background(), interruptSignals...)
	defer cancel()

	a, err := app.New(ctx, cfg)
	if err != nil {
		log.Println(err)
		return
	}

	a.Run(ctx)
}
