package storage

import (
	"cmp"
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"mmm-revived/internal/database"
)

type Options struct {
	Connections       int
	BusyTimeout       time.Duration
	Synchronous       string
	CacheKiB          int
	WALAutocheckpoint int
}

func DefaultOptions() Options {
	return Options{Connections: 4, BusyTimeout: 5 * time.Second, Synchronous: "FULL", CacheKiB: 8192, WALAutocheckpoint: 1000}
}

func (o Options) normalized() Options {
	defaults := DefaultOptions()
	o.Connections = cmp.Or(max(o.Connections, 0), defaults.Connections)
	o.BusyTimeout = cmp.Or(max(o.BusyTimeout, 0), defaults.BusyTimeout)
	o.Synchronous = strings.ToUpper(cmp.Or(o.Synchronous, defaults.Synchronous))
	o.CacheKiB = cmp.Or(max(o.CacheKiB, 0), defaults.CacheKiB)
	o.WALAutocheckpoint = cmp.Or(max(o.WALAutocheckpoint, 0), defaults.WALAutocheckpoint)
	return o
}

type Database struct {
	Queries *database.Queries
	SQL     *sql.DB
}

func Open(path string, options Options) (*Database, error) {
	options = options.normalized()
	sqlDB, err := openSQLite(path,
		"foreign_keys(1)",
		fmt.Sprintf("busy_timeout(%d)", options.BusyTimeout.Milliseconds()),
		"synchronous("+options.Synchronous+")",
		fmt.Sprintf("wal_autocheckpoint(%d)", options.WALAutocheckpoint),
		"temp_store(MEMORY)",
		fmt.Sprintf("cache_size(-%d)", options.CacheKiB),
	)
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(options.Connections)
	sqlDB.SetMaxIdleConns(options.Connections)
	queries, err := initialize(sqlDB)
	if err != nil {
		sqlDB.Close()
		return nil, err
	}
	return &Database{Queries: queries, SQL: sqlDB}, nil
}

func initialize(db *sql.DB) (*database.Queries, error) {
	for _, statement := range []string{"PRAGMA journal_mode=WAL", database.Schema} {
		if _, err := db.Exec(statement); err != nil {
			return nil, err
		}
	}
	return database.Prepare(context.Background(), db)
}

func (d *Database) Close() error {
	if d.Queries != nil {
		d.Queries.Close()
	}
	return d.SQL.Close()
}
