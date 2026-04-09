package migration

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrSchemaDrift is returned by VerifyCurrent when the database is missing
// migrations that exist on disk. Callers in main() should surface this as a
// fatal startup error with clear remediation instructions, because continuing
// to boot a server against a stale schema causes confusing "500 internal
// server error" responses whose root cause is not visible in the access log.
var ErrSchemaDrift = errors.New("schema drift detected")

// DriftDetails describes the specific versions that are present on disk but
// missing from the database. Attach this to the returned error with
// fmt.Errorf("...: %w", ErrSchemaDrift) to keep the sentinel checkable while
// still surfacing diagnostic context.
type DriftDetails struct {
	// Missing is the sorted list of migration versions present on disk but
	// not yet recorded in schema_migrations.
	Missing []int
}

func (d DriftDetails) String() string {
	return fmt.Sprintf("pending versions: %v (run 'make migrate-up' to apply)", d.Missing)
}

// VerifyCurrent compares migration files on disk with rows in the
// schema_migrations table and returns a non-nil error if the database is
// missing any migration that has been committed to disk.
//
// Error classification:
//   - rootDir unreadable or contains malformed files: returned as-is from
//     loadMigrations (file system / parse error)
//   - schema_migrations table does not exist: returned as a wrapped error
//     indicating no migrations have ever been applied; remediation is the
//     same ('run make migrate-up')
//   - schema_migrations query fails for any other reason: returned wrapped
//     with the DB error so operators can distinguish connectivity issues
//   - database is behind disk (drift): returned as fmt.Errorf wrapping
//     ErrSchemaDrift with DriftDetails in the message
//   - database is ahead of disk (rollback / foreign deployment): VerifyCurrent
//     intentionally tolerates this state by returning nil, because the
//     currently-deployed code only needs its own migrations applied and does
//     not care about versions it has no SQL for
func VerifyCurrent(ctx context.Context, pool *pgxpool.Pool, rootDir string) error {
	runner := NewRunner(pool, rootDir)
	available, err := runner.loadMigrations()
	if err != nil {
		return fmt.Errorf("load migration files: %w", err)
	}
	applied, err := queryAppliedVersions(ctx, pool)
	if err != nil {
		return err
	}
	missing := diffMissingVersions(available, applied)
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("%w: %s", ErrSchemaDrift, DriftDetails{Missing: missing}.String())
}

// diffMissingVersions returns the sorted list of on-disk migration versions
// that are not present in the applied set. Extracted as a pure helper so it
// can be unit tested without a live database pool.
func diffMissingVersions(available []migrationFile, applied map[int]struct{}) []int {
	missing := make([]int, 0)
	for _, m := range available {
		if _, ok := applied[m.Version]; !ok {
			missing = append(missing, m.Version)
		}
	}
	sort.Ints(missing)
	return missing
}

// queryAppliedVersions reads schema_migrations directly. This intentionally
// does not call runner.ensureTable: a verify step must never mutate the
// database. Instead we treat a missing table as drift with a dedicated
// message so the remediation ('run make migrate-up') still applies.
func queryAppliedVersions(ctx context.Context, pool *pgxpool.Pool) (map[int]struct{}, error) {
	rows, err := pool.Query(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		// 42P01 is postgres "undefined_table" — treat as "no migrations
		// applied yet" which is a distinct subtype of drift that deserves
		// its own error message so operators do not hunt for connection
		// problems when the real issue is an empty database.
		if isUndefinedTableError(err) {
			return nil, fmt.Errorf(
				"%w: schema_migrations table missing, run 'make migrate-up'",
				ErrSchemaDrift,
			)
		}
		return nil, fmt.Errorf("query schema_migrations: %w", err)
	}
	defer rows.Close()
	applied := map[int]struct{}{}
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("scan schema_migrations row: %w", err)
		}
		applied[v] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate schema_migrations: %w", err)
	}
	return applied, nil
}

// isUndefinedTableError unwraps a pgx error and returns true when the server
// reported SQLSTATE 42P01 (undefined_table).
func isUndefinedTableError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "42P01"
	}
	return false
}
