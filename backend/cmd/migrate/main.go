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
	var direction string
	flag.StringVar(&direction, "dir", "up", "migration direction: up or down")
	flag.Parse()

	if arg := flag.Arg(0); arg != "" {
		direction = arg
	}
	if direction != "up" && direction != "down" {
		log.Fatalf("invalid direction %q (expected up or down)", direction)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	ctx := context.Background()
	pool, err := repo.NewDBPool(ctx, cfg)
	if err != nil {
		log.Fatalf("connect database: %v", err)
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
		log.Fatalf("run migrations: %v", err)
	}

	fmt.Printf("migrations %s complete\n", direction)
}
