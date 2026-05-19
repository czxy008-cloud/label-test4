package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"clinic-appointment/internal/config"
	"clinic-appointment/internal/logger"

	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
)

var db *sql.DB

func Init(cfg config.DatabaseConfig) error {
	var err error
	db, err = sql.Open("pgx", cfg.DSN())
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping db: %w", err)
	}

	logger.Info("database connected successfully")
	return nil
}

func GetDB() *sql.DB {
	return db
}

func Close() {
	if db != nil {
		if err := db.Close(); err != nil {
			logger.Error("failed to close database", zap.Error(err))
		}
	}
}

func BeginTx(ctx context.Context) (*sql.Tx, error) {
	return db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
}
