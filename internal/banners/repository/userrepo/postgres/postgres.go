package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/Leopold1975/banners_control/internal/banners/domain/models"
	"github.com/Leopold1975/banners_control/internal/banners/repository/userrepo"
	"github.com/Leopold1975/banners_control/internal/pkg/config"
	"github.com/Leopold1975/banners_control/internal/pkg/pgtools"
	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UsersPostgresRepo struct {
	db *pgxpool.Pool
}

func New(ctx context.Context, cfg config.PostgresDB) (UsersPostgresRepo, error) {
	connString := "postgres://" + cfg.Username + ":" + cfg.Password + "@" +
		cfg.Addr + "/" + cfg.DB + "?" + "sslmode=" + cfg.SSLmode + "&pool_max_conns=" + cfg.MaxConns

	db, err := pgtools.Connect(ctx, connString)
	if err != nil {
		return UsersPostgresRepo{}, fmt.Errorf("connect to db error: %w", err)
	}

	if err := pgtools.ApplyMigration(cfg); err != nil {
		return UsersPostgresRepo{}, fmt.Errorf("apply migration error: %w", err)
	}

	return UsersPostgresRepo{
		db: db,
	}, nil
}

func (ur UsersPostgresRepo) CreateUser(ctx context.Context, u models.User) error {
	tx, err := ur.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("cannot begin transaction error: %w", err)
	}

	defer func() {
		err = pgtools.CommitOrRollback(ctx, tx, err, "create")
	}()

	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

	query, args, err := psql.Insert("users").
		Columns("username", "password_hash", "user_role", "feature_id", "tag_ids").
		Values(u.Username, u.PasswordHash, u.Role, u.Feature, u.Tags).ToSql()
	if err != nil {
		return fmt.Errorf("to sql error: %w", err)
	}

	_, err = tx.Exec(ctx, query, args...)
	if err != nil {
		target := new(pgconn.PgError)
		if errors.As(err, &target) {
			// другие коды будут добавлены по необходимости.
			switch target.Code { //nolint:gocritic
			case "23505":
				return userrepo.ErrAleradyExists
			}
		}

		return fmt.Errorf("exec error: %w", err)
	}

	return nil
}

func (ur UsersPostgresRepo) GetUser(ctx context.Context, username string) (models.User, error) {
	tx, err := ur.db.Begin(ctx)
	if err != nil {
		return models.User{}, fmt.Errorf("cannot begin transaction error: %w", err)
	}

	defer func() {
		err = pgtools.CommitOrRollback(ctx, tx, err, "create")
	}()

	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

	query, args, err := psql.Select("id", "username", "password_hash", "user_role", "feature_id", "tag_ids").
		From("users").
		Where(squirrel.Eq{"username": username}).ToSql()
	if err != nil {
		return models.User{}, fmt.Errorf("to sql error: %w", err)
	}

	var u models.User

	if err := tx.QueryRow(ctx, query, args...).Scan(
		&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.Feature, &u.Tags); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return u, userrepo.ErrNotFound
		}

		return u, fmt.Errorf("scan error: %w", err)
	}

	return u, nil
}
