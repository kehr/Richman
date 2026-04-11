package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/richman/backend/internal/config"
	"github.com/richman/backend/internal/repo"
)

// Seed loads idempotent reference data from backend/db/seed/*.sql.
//
// Seed files must be self-idempotent (ON CONFLICT DO NOTHING, CREATE IF NOT
// EXISTS, etc.) because this command is safe to run repeatedly and does not
// track history the way schema migrations do. Files are executed in
// lexicographic order inside their own transactions using pgx's simple query
// protocol, mirroring migration/runner.go's execFile so multi-statement
// scripts and DO $$ ... $$ blocks work correctly.
func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
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

	seedDir := filepath.Join("db", "seed")
	files, err := collectSeedFiles(seedDir)
	if err != nil {
		return fmt.Errorf("scan seed dir: %w", err)
	}
	if len(files) == 0 {
		fmt.Printf("no seed files found in %s\n", seedDir)
		return nil
	}

	for _, path := range files {
		if err := execSeedFile(ctx, pool, path); err != nil {
			return fmt.Errorf("apply seed %s: %w", filepath.Base(path), err)
		}
		fmt.Printf("seed applied: %s\n", filepath.Base(path))
	}
	fmt.Printf("seeding complete (%d file(s))\n", len(files))
	return nil
}

func collectSeedFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		files = append(files, filepath.Join(dir, entry.Name()))
	}
	sort.Strings(files)
	return files, nil
}

func execSeedFile(ctx context.Context, pool *pgxpool.Pool, path string) error {
	contents, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	body := strings.TrimSpace(string(contents))
	if body == "" {
		return nil
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	mrr := tx.Conn().PgConn().Exec(ctx, body)
	if _, execErr := mrr.ReadAll(); execErr != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("exec failed: %w (rollback failed: %v)", execErr, rbErr)
		}
		return execErr
	}
	return tx.Commit(ctx)
}
