package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"path/filepath"

	"github.com/richman/backend/internal/config"
	"github.com/richman/backend/internal/migration"
	"github.com/richman/backend/internal/repo"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	var direction string
	flag.StringVar(&direction, "dir", "up", "migration direction: up or down")
	flag.Parse()

	if arg := flag.Arg(0); arg != "" {
		direction = arg
	}
	if direction != "up" && direction != "down" {
		return fmt.Errorf("invalid direction %q (expected up or down)", direction)
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	ctx := context.Background()
	pool, err := repo.NewDBPool(ctx, cfg)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer pool.Close()

	runner := migration.NewRunner(pool, filepath.Join("db", "migration"))
	switch direction {
	case "up":
		err = runner.Up(ctx)
	case "down":
		err = runner.Down(ctx)
	}
	if err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	fmt.Printf("migrations %s complete\n", direction)
	return nil
}
