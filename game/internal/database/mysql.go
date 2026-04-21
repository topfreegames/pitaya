package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type MySQLConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
}

type MySQLDatabase struct {
	db *sql.DB
}

func NewMySQLDatabase(config *MySQLConfig) (*MySQLDatabase, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		config.User,
		config.Password,
		config.Host,
		config.Port,
		config.Database,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db.SetMaxOpenConns(100)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	return &MySQLDatabase{db: db}, nil
}

func (md *MySQLDatabase) Close() error {
	return md.db.Close()
}

func (md *MySQLDatabase) GetDB() *sql.DB {
	return md.db
}

func (md *MySQLDatabase) Ping() error {
	return md.db.Ping()
}

func (md *MySQLDatabase) BeginTx() (*sql.Tx, error) {
	return md.db.Begin()
}

func (md *MySQLDatabase) Exec(query string, args ...interface{}) (sql.Result, error) {
	return md.db.Exec(query, args...)
}

func (md *MySQLDatabase) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return md.db.Query(query, args...)
}

func (md *MySQLDatabase) QueryRow(query string, args ...interface{}) *sql.Row {
	return md.db.QueryRow(query, args...)
}

func (md *MySQLDatabase) Prepare(query string) (*sql.Stmt, error) {
	return md.db.Prepare(query)
}

func (md *MySQLDatabase) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return md.db.ExecContext(ctx, query, args...)
}

func (md *MySQLDatabase) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return md.db.QueryContext(ctx, query, args...)
}

func (md *MySQLDatabase) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return md.db.QueryRowContext(ctx, query, args...)
}

func (md *MySQLDatabase) HealthCheck() error {
	return md.db.Ping()
}

func (md *MySQLDatabase) GetStats() map[string]interface{} {
	stats := md.db.Stats()
	return map[string]interface{}{
		"max_open_connections": stats.MaxOpenConnections,
		"open_connections":     stats.OpenConnections,
		"in_use":              stats.InUse,
		"idle":                stats.Idle,
		"wait_count":           stats.WaitCount,
		"wait_duration":        stats.WaitDuration,
		"max_idle_closed":      stats.MaxIdleClosed,
		"max_lifetime_closed":  stats.MaxLifetimeClosed,
	}
}
