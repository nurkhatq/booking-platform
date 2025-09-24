package database

import (
    "fmt"
    "log"
    "time"
    
    "github.com/jmoiron/sqlx"
    _ "github.com/lib/pq"
    
    "booking-platform/shared/config"
)

var DB *sqlx.DB

func Initialize(cfg *config.Config) error {
    dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
        cfg.Database.Host,
        cfg.Database.Port,
        cfg.Database.User,
        cfg.Database.Password,
        cfg.Database.Name,
    )
    
    var err error
    DB, err = sqlx.Connect("postgres", dsn)
    if err != nil {
        return fmt.Errorf("failed to connect to database: %w", err)
    }
    
    // Configure connection pool
    DB.SetMaxOpenConns(cfg.Database.MaxConnections)
    DB.SetMaxIdleConns(cfg.Database.MaxIdle)
    DB.SetConnMaxLifetime(time.Hour)
    
    // Test connection
    if err = DB.Ping(); err != nil {
        return fmt.Errorf("failed to ping database: %w", err)
    }
    
    log.Println("Database connection established successfully")
    return nil
}

func Close() error {
    if DB != nil {
        return DB.Close()
    }
    return nil
}

func GetDB() *sqlx.DB {
    return DB
}
