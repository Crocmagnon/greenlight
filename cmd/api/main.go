package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Crocmagnon/greenlight/internal/data"
	"github.com/Crocmagnon/greenlight/internal/jsonlog"
	_ "github.com/lib/pq"
)

const version = "1.0.0"

const (
	readTimeoutSeconds          = 5
	writeTimeoutSeconds         = 10
	dbMaxConns                  = 25
	dbInitialConnTimeoutSeconds = 5
)

type config struct {
	addr string
	env  string
	db   struct {
		dsn          string
		maxOpenConns int
		maxIdleConns int
		maxIdleTime  string
	}
}

type application struct {
	config config
	logger *jsonlog.Logger
	models data.Models
}

func main() {
	var cfg config

	flag.StringVar(&cfg.addr, "addr", "127.0.0.1:4000", "API server listen addr")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")

	flag.StringVar(&cfg.db.dsn, "db-dsn", os.Getenv("GREENLIGHT_DB_DSN"), "PostgreSQL DSN")
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", dbMaxConns, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", dbMaxConns, "PostgreSQL max idle connections")
	flag.StringVar(&cfg.db.maxIdleTime, "db-max-idle-time", "15m", "PostgreSQL max connection idle time")

	flag.Parse()

	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)

	db, err := openDB(cfg)
	if err != nil {
		logger.PrintFatal(err, nil)
	}

	defer db.Close() //nolint:errcheck

	logger.PrintInfo("database connection established", nil)

	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(db),
	}

	srv := &http.Server{
		Addr:         cfg.addr,
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  readTimeoutSeconds * time.Second,
		WriteTimeout: writeTimeoutSeconds * time.Second,
		ErrorLog:     log.New(logger, "", 0),
	}

	logger.PrintInfo("starting server", jsonlog.Properties{
		"addr": srv.Addr,
		"env":  cfg.env,
	})

	err = srv.ListenAndServe()
	logger.PrintFatal(err, nil)
}

func openDB(cfg config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.db.dsn)
	if err != nil {
		return nil, fmt.Errorf("opening DB connection: %w", err)
	}

	// Set the maximum number of open (in-use + idle) connections in the pool. Note that
	// passing a value less than or equal to 0 will mean there is no limit.
	// Tweak this value. Max enforced by DB: psql -c "SHOW max_connections;" (default 100).
	db.SetMaxOpenConns(cfg.db.maxOpenConns)

	// Set the maximum number of idle connections in the pool. Again, passing a value
	// less than or equal to 0 will mean there is no limit.
	db.SetMaxIdleConns(cfg.db.maxIdleConns)

	duration, err := time.ParseDuration(cfg.db.maxIdleTime)
	if err != nil {
		return nil, fmt.Errorf("parsing duration: %w", err)
	}

	db.SetConnMaxIdleTime(duration)

	ctx, cancel := context.WithTimeout(context.Background(), dbInitialConnTimeoutSeconds*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("connecting to DB: %w", err)
	}

	return db, nil
}
