package main

import (
	"flag"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ggetzie/badwords_be/internal/data"
)

const version = "1.0.0"

type config struct {
	port       int
	env        string
	db         data.DBConfig
	webBaseURL string

	limiter struct {
		rps     float64
		burst   int
		enabled bool
	}

	defaultPageSize int

	cors struct {
		trustedOrigins []string
	}
}

type application struct {
	config config
	logger *slog.Logger
	models data.Models
	wg     sync.WaitGroup
}

func main() {
	var cfg config
	flag.IntVar(&cfg.port, "port", 8000, "Server port to listen on")
	flag.StringVar(&cfg.env, "env", "development", "Application environment (development|production)")

	// database connection pool settings
	flag.StringVar(&cfg.db.DSN, "db-dsn", "", "PostgreSQL DSN")
	flag.IntVar(&cfg.db.MaxOpenConns, "db-max-open-conns", 25, "PostgreSQL database connection pool max open connections")
	flag.IntVar(&cfg.db.MinConns, "db-min-conns", 4, "PostgreSQL database connection pool minimum connections")
	flag.DurationVar(&cfg.db.MaxIdleTime, "db-max-idle-time", 15*time.Minute, "PostgreSQL database connection pool max connection idle time")

	// rate limiter settings
	flag.Float64Var(&cfg.limiter.rps, "limiter-rps", 2, "Rate limiter maximum requests per second")
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 4, "Rate limiter burst")
	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", false, "Enable rate limiter")

	// AWS Settings
	var AWS_ACCESS_KEY_ID string
	var AWS_SECRET_ACCESS_KEY string
	flag.StringVar(&AWS_ACCESS_KEY_ID, "aws-access-key-id", "", "AWS Access Key ID")
	flag.StringVar(&AWS_SECRET_ACCESS_KEY, "aws-secret-access-key", "", "AWS Secret Access Key")

	// CORS settings
	flag.Func("cors-trusted-origins", "List of trusted origins for CORS (comma separated)", func(val string) error {
		cfg.cors.trustedOrigins = strings.Split(val, ",")
		return nil
	})

	// Base URL - the hostname for the web frontend to build links
	flag.StringVar(&cfg.webBaseURL, "base-url", "http://localhost:3001", "Base URL for the web frontend")

	flag.Parse()

	var logOptions slog.HandlerOptions

	if cfg.env == "development" {
		logOptions = slog.HandlerOptions{
			Level: slog.LevelDebug,
		}
	}
	cfg.defaultPageSize = 20

	logger := slog.New(slog.NewTextHandler(os.Stdout, &logOptions))

	for _, origin := range cfg.cors.trustedOrigins {
		logger.Info("cors setting", "trusted_origin", origin)
	}
	logger.Info("web base url", "url", cfg.webBaseURL)
	dbpool, err := data.OpenDB(cfg.db)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	logger.Info("database connection pool established")

	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(dbpool),
	}

	err = app.serve()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

}
