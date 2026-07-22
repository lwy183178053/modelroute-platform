package modelseed

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lwy183178053/modelroute-platform/apps/api/internal/httpapi"
)

type SQLRepository struct {
	database *sql.DB
	routing  *httpapi.SQLRoutingStore
}

func NewSQLRepository(database *sql.DB) *SQLRepository {
	return &SQLRepository{database: database, routing: httpapi.NewSQLRoutingStore(database)}
}

func (r *SQLRepository) ListCanonicalModels(ctx context.Context) ([]httpapi.CanonicalModel, error) {
	return r.routing.ListCanonicalModels(ctx)
}

func (r *SQLRepository) CreateCanonicalModel(ctx context.Context, command httpapi.CreateCanonicalModelCommand) (httpapi.CanonicalModel, error) {
	return r.routing.CreateCanonicalModel(ctx, command)
}

func (r *SQLRepository) PublishOfficialPrice(ctx context.Context, modelID int64, adminID int, price httpapi.OfficialPrice) (httpapi.OfficialPrice, error) {
	return r.routing.PublishOfficialPrice(ctx, modelID, adminID, price)
}

func (r *SQLRepository) SetRuntimeMetadata(ctx context.Context, values map[string]string) error {
	tx, err := r.database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开始运行元数据事务：%w", err)
	}
	defer tx.Rollback()
	for key, value := range values {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO platform_runtime_metadata (key, value, updated_at)
VALUES ($1, $2, now())
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()
`, key, value); err != nil {
			return fmt.Errorf("写入运行元数据 %s：%w", key, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交运行元数据事务：%w", err)
	}
	return nil
}
