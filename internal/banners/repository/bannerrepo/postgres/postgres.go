package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Leopold1975/banners_control/internal/banners/domain/models"
	repo "github.com/Leopold1975/banners_control/internal/banners/repository/bannerrepo"
	"github.com/Leopold1975/banners_control/internal/pkg/config"
	"github.com/Leopold1975/banners_control/internal/pkg/pgtools"
	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" // driver for migrations
)

type BannersPostgresRepo struct {
	db *pgxpool.Pool
}

func New(ctx context.Context, cfg config.PostgresDB) (BannersPostgresRepo, error) {
	connString := "postgres://" + cfg.Username + ":" + cfg.Password + "@" +
		cfg.Addr + "/" + cfg.DB + "?" + "sslmode=" + cfg.SSLmode + "&pool_max_conns=" + cfg.MaxConns

	db, err := pgtools.Connect(ctx, connString)
	if err != nil {
		return BannersPostgresRepo{}, fmt.Errorf("connect to db error: %w", err)
	}

	if err := pgtools.ApplyMigration(cfg); err != nil {
		return BannersPostgresRepo{}, fmt.Errorf("apply migration error: %w", err)
	}

	return BannersPostgresRepo{
		db: db,
	}, nil
}

func (br BannersPostgresRepo) CreateBanner(ctx context.Context, //nolint:nonamedreturns
	banner models.Banner,
) (id int, err error) {
	contentJSON, err := json.Marshal(banner.Content)
	if err != nil {
		return 0, fmt.Errorf("marshall content error: %w", err)
	}

	tx, err := br.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("cannot begin transaction error: %w", err)
	}

	defer func() {
		err = pgtools.CommitOrRollback(ctx, tx, err, "create")
	}()

	cols := []string{"feature_id", "tag_ids", "is_active", "content"}
	vals := []interface{}{banner.FeatureID, banner.Tags, banner.Active, contentJSON}

	zeroTime := time.Time{}
	if banner.CreatedAt != zeroTime {
		cols = append(cols, "created_at")
		vals = append(vals, banner.CreatedAt)
	}

	if banner.UpdatedAt != zeroTime {
		cols = append(cols, "updated_at")
		vals = append(vals, banner.UpdatedAt)
	}

	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

	query, args, err := psql.Insert("banners").
		Columns(cols...).
		Values(vals...).
		Suffix("RETURNING id").ToSql()
	if err != nil {
		return 0, fmt.Errorf("to sql error: %w", err)
	}

	row := tx.QueryRow(ctx, query, args...)

	err = row.Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("scan error: %w", err)
	}

	return id, nil
}

func (br BannersPostgresRepo) UpdateBanner(ctx context.Context, banner models.Banner) (err error) {
	contentJSON, err := json.Marshal(banner.Content)
	if err != nil {
		return fmt.Errorf("marshall content error: %w", err)
	}

	tx, err := br.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("cannot begin transaction error: %w", err)
	}

	defer func() {
		err = pgtools.CommitOrRollback(ctx, tx, err, "update")
	}()

	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

	query, args, err := psql.Update("banners").
		Set("feature_id", banner.FeatureID).
		Set("tag_ids", banner.Tags).
		Set("is_active", banner.Active).
		Set("content", contentJSON).
		Set("updated_at", banner.UpdatedAt).
		Where(squirrel.Eq{"id": banner.ID}).ToSql()
	if err != nil {
		return fmt.Errorf("to sql error: %w", err)
	}

	ct, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("exec error: %w", err)
	}

	if ct.RowsAffected() == 0 {
		return repo.ErrNotFound
	}

	return nil
}

func (br BannersPostgresRepo) DeleteBanner(ctx context.Context, bannerID int) (err error) {
	tx, err := br.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("cannot begin transaction error: %w", err)
	}

	defer func() {
		err = pgtools.CommitOrRollback(ctx, tx, err, "delete")
	}()

	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

	query, args, err := psql.Delete("banners").
		Where(squirrel.Eq{"id": bannerID}).ToSql()
	if err != nil {
		return fmt.Errorf("to sql error: %w", err)
	}

	ct, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("exec error: %w", err)
	}

	if ct.RowsAffected() == 0 {
		return repo.ErrNotFound
	}

	return nil
}

func (br BannersPostgresRepo) GetBannerByFeatureAndTags(ctx context.Context, //nolint:cyclop,nonamedreturns
	req repo.GetBannerRequest,
) (banners []models.Banner, err error) {
	tx, err := br.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot begin transaction error: %w", err)
	}

	defer func() {
		err = pgtools.CommitOrRollback(ctx, tx, err, "delete")
	}()

	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
	sb := psql.Select("id", "feature_id", "tag_ids", "is_active", "updated_at", "created_at", "content").
		From("banners")

	if req.FeatureID != -1 {
		sb = sb.Where(squirrel.Eq{"feature_id": req.FeatureID})
	}

	if len(req.Tags) != 0 {
		sb = sb.Where("(tag_ids && ?)", req.Tags)
	}

	if req.OnlyActive {
		sb = sb.Where(squirrel.Eq{"is_active": true}).
			OrderBy("id ASC")
	} else {
		sb = sb.OrderBy("id ASC")
	}

	if req.Offset != 0 {
		sb = sb.Offset(uint64(req.Offset))
	}

	if req.Limit != 0 {
		sb = sb.Limit(uint64(req.Limit))
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("to sql error: %w", err)
	}

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}
	defer rows.Close()

	banners = make([]models.Banner, 0, 10) //nolint:gomnd

	for rows.Next() {
		var b models.Banner

		var contentJSON string

		err = rows.Scan(&b.ID, &b.FeatureID, &b.Tags, &b.Active, &b.UpdatedAt, &b.CreatedAt, &contentJSON)
		if err != nil {
			return nil, fmt.Errorf("scan error %w", err)
		}

		tmp := make(map[string]interface{})

		err = json.Unmarshal([]byte(contentJSON), &tmp)
		if err != nil {
			return nil, fmt.Errorf("unmarshal error %w", err)
		}

		b.Content = tmp

		banners = append(banners, b)
	}

	return banners, nil
}

func (br BannersPostgresRepo) Shutdown(ctx context.Context) error {
	done := make(chan struct{})

	go func() {
		br.db.Close()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("context error: %w", ctx.Err())
	case <-done:
		return nil
	}
}
