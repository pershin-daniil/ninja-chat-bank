package store

import (
	"database/sql"
	"fmt"
	"net/url"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/jackc/pgx/v5/stdlib" // third party driver pgx
	"go.uber.org/zap"
)

//go:generate options-gen -out-filename=client_psql_options.gen.go -from-struct=PSQLOptions
type PSQLOptions struct {
	address    string `option:"mandatory" validate:"required,hostname_port"`
	username   string `option:"mandatory" validate:"required"`
	password   string `option:"mandatory" validate:"required"`
	database   string `option:"mandatory" validate:"required"`
	production bool   `option:"mandatory"`
	debugMode  bool
}

func NewPSQLClient(opts PSQLOptions) (*Client, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate PSQLOptions store: %v", err)
	}

	if opts.production && opts.debugMode {
		zap.L().Warn("Debug mode is enabled for Postgres client in production environment. Review configuration settings.")
	}

	db, err := NewPgxDB(NewPgxOptions(opts.address, opts.username, opts.password, opts.database))
	if err != nil {
		return nil, fmt.Errorf("failed to init db driver: %v", err)
	}

	var clientOpts []Option

	drv := Driver(entsql.OpenDB(dialect.Postgres, db))

	clientOpts = append(clientOpts, drv)

	if opts.debugMode {
		clientOpts = append(clientOpts, Debug())
	}

	return NewClient(clientOpts...), nil
}

//go:generate options-gen -out-filename=client_psql_pgx_options.gen.go -from-struct=PgxOptions
type PgxOptions struct {
	address  string `option:"mandatory" validate:"required,hostname_port"`
	username string `option:"mandatory" validate:"required"`
	password string `option:"mandatory" validate:"required"`
	database string `option:"mandatory" validate:"required"`
}

func NewPgxDB(opts PgxOptions) (*sql.DB, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate pgxOptions store: %v", err)
	}

	dbURL := url.URL{
		Scheme: "postgres",
		Host:   opts.address,
		User:   url.UserPassword(opts.username, opts.password),
		Path:   opts.database,
	}

	db, err := sql.Open("pgx", dbURL.String())
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %v", err)
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping db: %v", err)
	}

	return db, nil
}
