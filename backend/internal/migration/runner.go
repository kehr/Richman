package migration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Runner executes SQL migrations located on disk.
type Runner struct {
	pool    *pgxpool.Pool
	rootDir string
}

// NewRunner creates a migration runner rooted at the given directory.
func NewRunner(pool *pgxpool.Pool, rootDir string) *Runner {
	return &Runner{pool: pool, rootDir: rootDir}
}

// Up applies all pending migrations.
func (r *Runner) Up(ctx context.Context) error {
	if err := r.ensureTable(ctx); err != nil {
		return err
	}
	available, err := r.loadMigrations()
	if err != nil {
		return err
	}
	applied, err := r.appliedVersions(ctx)
	if err != nil {
		return err
	}
	for _, m := range available {
		if applied[m.Version] {
			continue
		}
		if err := r.execFile(ctx, m.UpPath); err != nil {
			return fmt.Errorf("apply migration %s: %w", m.Name, err)
		}
		if err := r.recordApplied(ctx, m.Version, m.Name); err != nil {
			return err
		}
	}
	return nil
}

// Down rolls back the most recently applied migration.
func (r *Runner) Down(ctx context.Context) error {
	if err := r.ensureTable(ctx); err != nil {
		return err
	}
	available, err := r.loadMigrations()
	if err != nil {
		return err
	}
	order, err := r.appliedHistory(ctx)
	if err != nil {
		return err
	}
	if len(order) == 0 {
		return nil
	}
	last := order[len(order)-1]
	m, ok := availableByVersion(available)[last.Version]
	if !ok {
		return fmt.Errorf("migration %d not found on disk", last.Version)
	}
	if m.DownPath == "" {
		return fmt.Errorf("migration %s has no down script", m.Name)
	}
	if err := r.execFile(ctx, m.DownPath); err != nil {
		return fmt.Errorf("rollback migration %s: %w", m.Name, err)
	}
	return r.removeApplied(ctx, last.Version)
}

type migrationFile struct {
	Version  int
	Name     string
	UpPath   string
	DownPath string
}

func (r *Runner) loadMigrations() ([]migrationFile, error) {
	entries, err := os.ReadDir(r.rootDir)
	if err != nil {
		return nil, err
	}
	files := map[int]*migrationFile{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		var suffix string
		switch {
		case strings.HasSuffix(name, ".up.sql"):
			suffix = ".up.sql"
		case strings.HasSuffix(name, ".down.sql"):
			suffix = ".down.sql"
		default:
			continue
		}
		version, slug, err := parseMigrationName(name, suffix)
		if err != nil {
			return nil, err
		}
		rec, ok := files[version]
		if !ok {
			rec = &migrationFile{Version: version, Name: slug}
			files[version] = rec
		}
		path := filepath.Join(r.rootDir, name)
		if suffix == ".up.sql" {
			rec.UpPath = path
		} else {
			rec.DownPath = path
		}
	}
	list := make([]migrationFile, 0, len(files))
	for _, rec := range files {
		if rec.UpPath == "" {
			return nil, fmt.Errorf("migration %d missing up script", rec.Version)
		}
		list = append(list, *rec)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Version < list[j].Version
	})
	return list, nil
}

func parseMigrationName(name, suffix string) (version int, slug string, err error) {
	trimmed := strings.TrimSuffix(name, suffix)
	parts := strings.SplitN(trimmed, "_", 2)
	if len(parts) != 2 {
		return 0, "", fmt.Errorf("invalid migration filename: %s", name)
	}
	version, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, "", fmt.Errorf("invalid migration version in %s", name)
	}
	return version, parts[1], nil
}

func (r *Runner) ensureTable(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version BIGINT PRIMARY KEY,
		name TEXT NOT NULL,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`)
	return err
}

func (r *Runner) appliedVersions(ctx context.Context) (map[int]bool, error) {
	rows, err := r.pool.Query(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	versions := map[int]bool{}
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		versions[v] = true
	}
	return versions, rows.Err()
}

type migrationRecord struct {
	Version int
	Name    string
	Applied time.Time
}

func (r *Runner) appliedHistory(ctx context.Context) ([]migrationRecord, error) {
	rows, err := r.pool.Query(ctx, `SELECT version, name, applied_at FROM schema_migrations ORDER BY version ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []migrationRecord
	for rows.Next() {
		var rec migrationRecord
		if err := rows.Scan(&rec.Version, &rec.Name, &rec.Applied); err != nil {
			return nil, err
		}
		items = append(items, rec)
	}
	return items, rows.Err()
}

func (r *Runner) recordApplied(ctx context.Context, version int, name string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO schema_migrations (version, name, applied_at) VALUES ($1, $2, NOW())`,
		version, name,
	)
	return err
}

func (r *Runner) removeApplied(ctx context.Context, version int) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM schema_migrations WHERE version = $1`, version)
	return err
}

func (r *Runner) execFile(ctx context.Context, path string) error {
	contents, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	stmts := splitStatements(string(contents))
	if len(stmts) == 0 {
		return nil
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	for _, stmt := range stmts {
		if _, err := tx.Exec(ctx, stmt); err != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				return fmt.Errorf("exec failed: %w (rollback failed: %v)", err, rbErr)
			}
			return err
		}
	}
	return tx.Commit(ctx)
}

func splitStatements(body string) []string {
	parts := strings.Split(body, ";")
	stmts := make([]string, 0, len(parts))
	for _, part := range parts {
		stmt := strings.TrimSpace(part)
		if stmt == "" {
			continue
		}
		stmts = append(stmts, stmt)
	}
	return stmts
}

func availableByVersion(list []migrationFile) map[int]migrationFile {
	m := make(map[int]migrationFile, len(list))
	for _, item := range list {
		m[item.Version] = item
	}
	return m
}
