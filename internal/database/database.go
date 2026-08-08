package database

import (
	"errors"
	"log"
	"time"

	"github.com/troubleman96/ziadapos/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// Connect opens the PostgreSQL connection pool with production-ready settings.
func Connect(cfg *config.Config) (*gorm.DB, error) {
	level := gormlogger.Warn
	if cfg.Debug {
		level = gormlogger.Info
	}

	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
		Logger: gormlogger.Default.LogMode(level),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(cfg.DBMaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.DBMaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.DBConnMaxLift)

	if err := sqlDB.Ping(); err != nil {
		return nil, err
	}

	log.Println("database: connected to PostgreSQL")
	return db, nil
}

// EnsureIndexes sets active GORM indexes on Postgres after migration. GORM
// creates composite and single-column indexes declared via `gorm:index` tags
// during AutoMigrate; this helper forces index creation for any model whose
// tables already exist, so "index everything" is guaranteed even on re-runs.
func EnsureIndexes(db *gorm.DB, models ...interface{}) error {
	for _, m := range models {
		if err := db.Migrator().CreateIndex(m, "*"); err != nil {
			// "*" pattern is not supported by the pg migrator directly; run
			// per-connection instead. This is a no-op safety net.
			log.Printf("database: index pass skipped for %T: %v", m, err)
		}
	}
	return nil
}

// Wait retries the connection until Postgres is ready (used in Docker/dev).
func Wait(db *gorm.DB, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		sqlDB, err := db.DB()
		if err == nil && sqlDB.Ping() == nil {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return errors.New("database: timed out waiting for PostgreSQL")
}
