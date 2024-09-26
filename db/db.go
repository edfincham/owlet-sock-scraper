package db

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresPool struct {
	DB *pgxpool.Pool
}

var (
	pgInstance *PostgresPool
	pgOnce     sync.Once
)

func NewDB(ctx context.Context, connString string) (*PostgresPool, error) {
	pgOnce.Do(func() {
		db, err := pgxpool.New(ctx, connString)
		if err != nil {
			panic(err)
		}

		pgInstance = &PostgresPool{db}
	})

	return pgInstance, nil
}

func (pg *PostgresPool) Ping(ctx context.Context) error {
	return pg.DB.Ping(ctx)
}

func (pg *PostgresPool) Close() {
	pg.DB.Close()
}

func (pg *PostgresPool) Migrate(migrationsPath string) error {
	config := pg.DB.Config().ConnConfig.ConnString()

	log.Println("Running migrations")

	m, err := migrate.New(
		"file://"+migrationsPath,
		config)
	if err != nil {
		return fmt.Errorf("Failed to create migrate instance: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to apply migrations: %v", err)
	}
	if err == migrate.ErrNoChange {
		log.Println("No migrations executed")
	}

	return nil
}
