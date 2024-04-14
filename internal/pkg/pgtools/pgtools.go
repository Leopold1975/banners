package pgtools

import (
	"context"
	"fmt"
	"time"

	"github.com/Leopold1975/banners_control/internal/pkg/config"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" // driver for migrations
	"github.com/pressly/goose/v3"
)

func Connect(ctx context.Context, connString string) (*pgxpool.Pool, error) {
	errCh := make(chan error)
	db := new(pgxpool.Pool)

	go func() {
		defer close(errCh)

		dbc, err := pgxpool.New(ctx, connString)
		if err != nil {
			errCh <- fmt.Errorf("cannot create db pool error: %w", err)

			return
		}

		defaultDelay := time.Second

		for {
			if err := dbc.Ping(ctx); err != nil {
				time.Sleep(defaultDelay)
				defaultDelay += time.Second

				if defaultDelay > time.Second*10 {
					errCh <- fmt.Errorf("cannot ping db error: %w", err)

					return
				}

				continue
			}

			break
		}

		db = dbc
	}()
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("context error: %w", ctx.Err())
	case err := <-errCh:
		if err != nil {
			return nil, err
		}

		return db, nil
	}
}

func ApplyMigration(cfg config.PostgresDB) error {
	migrationsDir := "./migrations"
	defaultVersion := 0

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("goose set dialect error: %w", err)
	}

	connString := "postgres://" + cfg.Username + ":" + cfg.Password + "@" +
		cfg.Addr + "/" + cfg.DB

	dbM, err := goose.OpenDBWithDriver("pgx", connString)
	if err != nil {
		return fmt.Errorf("goose open pgx db error: %w", err)
	}
	defer dbM.Close()

	// if cfg.Reload {
	if cfg.Reload {
		if err := goose.DownTo(dbM, migrationsDir, int64(defaultVersion)); err != nil {
			return fmt.Errorf("goose down error: %w", err)
		}
	}

	if err := goose.UpTo(dbM, migrationsDir, int64(cfg.Version)); err != nil {
		return fmt.Errorf("goose up error: %w", err)
	}

	return nil
}

func CommitOrRollback(ctx context.Context, tx pgx.Tx, err error, where string) error {
	if err == nil {
		if errT := tx.Commit(ctx); errT != nil {
			err = fmt.Errorf("commit error: %w", errT)
		}
	} else {
		if errT := tx.Rollback(ctx); errT != nil {
			err = fmt.Errorf("%s error: %w rollback error: %w", where, err, errT)
		} else {
			err = fmt.Errorf("%s error: %w", where, err)
		}
	}

	return err
}
