package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/templatedop/ftptemplate/config"
)

/**
 * DB is a wrapper for PostgreSQL database connection
 * that uses pgxpool as database driver
 */
type DB struct {
	*pgxpool.Pool
}

type DBInterface interface {
	Close()
	WithTx(ctx context.Context, fn func(tx pgx.Tx) error, levels ...pgx.TxIsoLevel) error
	ReadTx(ctx context.Context, fn func(tx pgx.Tx) error) error
	// You might have other methods that you want to expose through the interface
}

var _ DBInterface = (*DB)(nil)

type DBConfig struct {
	DBUsername        string
	DBPassword        string
	DBHost            string
	DBPort            string
	DBDatabase        string
	Schema            string
	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
	AppName           string
}

func NewDBConfig(c *config.Config) *DBConfig {
	return &DBConfig{
		DBUsername:        c.GetString("modules.db.username"),
		DBPassword:        c.GetString("modules.db.password"),
		DBHost:            c.GetString("modules.db.host"),
		DBPort:            c.GetString("modules.db.port"),
		DBDatabase:        c.GetString("modules.db.database"),
		Schema:            c.GetString("modules.db.schema"),
		MaxConns:          c.GetInt32("modules.db.maxconns"),
		MinConns:          c.GetInt32("modules.db.minconns"),
		MaxConnLifetime:   time.Duration(c.GetInt("modules.db.maxconnlifetime")) * time.Minute,
		MaxConnIdleTime:   time.Duration(c.GetInt("modules.db.maxconnidletime")) * time.Minute,
		HealthCheckPeriod: time.Duration(c.GetInt("modules.db.healthcheckperiod")) * time.Minute,
		AppName:           c.AppName(),
	}
}

func Pgxconfig(cfg *DBConfig) (*pgxpool.Config, error) {
	dsn := fmt.Sprintf("user=%s password=%s host=%s port=%s dbname=%s search_path=%s sslmode=disable",
		cfg.DBUsername,
		cfg.DBPassword,
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBDatabase,
		cfg.Schema,
	)

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}

	config.MaxConns = cfg.MaxConns                   // Maximum number of connections in the pool.
	config.MinConns = cfg.MinConns                   // Minimum number of connections to keep in the pool.
	config.MaxConnLifetime = cfg.MaxConnLifetime     // Maximum lifetime of a connection.
	config.MaxConnIdleTime = cfg.MaxConnIdleTime     // Maximum idle time of a connection in the pool.
	config.HealthCheckPeriod = cfg.HealthCheckPeriod // Period between connection health checks.
	config.ConnConfig.ConnectTimeout = 10 * time.Second
	config.ConnConfig.RuntimeParams = map[string]string{
		"application_name": cfg.AppName,
		"search_path":      cfg.Schema,
	}

	config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeCacheStatement
	config.ConnConfig.StatementCacheCapacity = 100
	config.ConnConfig.DescriptionCacheCapacity = 0

	return config, nil
}

func NewDB(cfg *DBConfig, pcfg *pgxpool.Config) (*DB, error) {

	ctx := context.Background()
	config, err := Pgxconfig(cfg)
	if err != nil {
		return nil, err
	}

	db, err := pgxpool.NewWithConfig(ctx, config)

	if err != nil {
		return nil, err
	}

	return &DB{
		db,
	}, nil
}

func (db *DB) Close() {
	db.Pool.Close()
}

func (db *DB) WithTx(ctx context.Context, fn func(tx pgx.Tx) error, levels ...pgx.TxIsoLevel) error {
	var level pgx.TxIsoLevel
	if len(levels) > 0 {
		level = levels[0]
	} else {
		level = pgx.ReadCommitted // Default value
	}
	return db.inTx(ctx, level, "", fn)
}

func (db *DB) ReadTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	return db.inTx(ctx, pgx.ReadCommitted, pgx.ReadOnly, fn)

}

func (db *DB) inTx(ctx context.Context, level pgx.TxIsoLevel, access pgx.TxAccessMode,
	fn func(tx pgx.Tx) error) (err error) {

	conn, errAcq := db.Pool.Acquire(ctx)
	if errAcq != nil {
		return fmt.Errorf("acquiring connection: %w", errAcq)
	}
	defer conn.Release()

	opts := pgx.TxOptions{
		IsoLevel:   level,
		AccessMode: access,
	}

	tx, errBegin := conn.BeginTx(ctx, opts)
	if errBegin != nil {
		return fmt.Errorf("begin tx: %w", errBegin)
	}

	defer func() {
		errRollback := tx.Rollback(ctx)
		if !(errRollback == nil || errors.Is(errRollback, pgx.ErrTxClosed)) {
			err = errRollback
		}
	}()

	if err := fn(tx); err != nil {
		if errRollback := tx.Rollback(ctx); errRollback != nil {
			return fmt.Errorf("rollback tx: %v (original: %w)", errRollback, err)
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}
