# nanovms Data Layer

`sqlx` for type-safe SQL, `goose` for migrations, embedded SQLite for dev.

## Stack

- `sqlx` v1.4+ (named params, struct scan, tx helpers)
- `goose` v3+ (versioned migrations, up/down)
- `modernc.org/sqlite` (pure-Go SQLite, no CGo)

## Schema versioning

```sql
-- migrations/00001_init.sql
-- +goose Up
CREATE TABLE sandboxes (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    image       TEXT NOT NULL,
    status      TEXT NOT NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (name)
);

-- +goose Down
DROP TABLE sandboxes;
```

## Migrations

```go
import _ "embed"
//go:embed migrations/*.sql
var migrationsFS embed.FS

func Migrate(db *sql.DB) error {
    return goose.Up(db, "migrations")
}
```

## Type-safe queries

```go
import "github.com/jmoiron/sqlx"

type SandboxRow struct {
    ID        string    `db:"id"`
    Name      string    `db:"name"`
    Image     string    `db:"image"`
    Status    string    `db:"status"`
    CreatedAt time.Time `db:"created_at"`
}

func FindByName(ctx context.Context, q sqlx.QueryerContext, name string) (*SandboxRow, error) {
    var sb SandboxRow
    err := sqlx.GetContext(ctx, q, &sb, "SELECT * FROM sandboxes WHERE name = ?", name)
    if err != nil { return nil, err }
    return &sb, nil
}
```

## Tx helpers

```go
func WithTx(ctx context.Context, db *sqlx.DB, fn func(*sqlx.Tx) error) error {
    tx, err := db.BeginTxx(ctx, nil)
    if err != nil { return err }
    defer func() {
        if p := recover(); p != nil {
            _ = tx.Rollback()
            panic(p)
        }
    }()
    if err := fn(tx); err != nil {
        _ = tx.Rollback()
        return err
    }
    return tx.Commit()
}
```

## Backups

- Daily: `sqlite3 .backup /var/lib/nanovms/state.db`
- WAL mode: continuous
- Retention: 30 days hot, 1 year cold

## Determinism in tests

```go
func newTestDB(t *testing.T) *sqlx.DB {
    db, err := sqlx.Connect("sqlite", ":memory:")
    if err != nil { t.Fatal(err) }
    if err := Migrate(db.DB); err != nil { t.Fatal(err) }
    return db
}
```
