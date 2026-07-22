package modelseed

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/lwy183178053/modelroute-platform/apps/api/internal/migrations"
)

func TestSQLRepositoryAppliesSeedIdempotentlyAgainstPostgres(t *testing.T) {
	databaseURL := os.Getenv("MODELROUTE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("未设置 PostgreSQL 集成测试地址")
	}
	database, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("打开测试数据库：%v", err)
	}
	defer database.Close()
	ctx := context.Background()
	if err := migrations.Up(ctx, database); err != nil {
		t.Fatalf("执行迁移：%v", err)
	}
	cleanupSeedModel(t, ctx, database, "gpt-5.4")
	defer cleanupSeedModel(t, ctx, database, "gpt-5.4")

	repository := NewSQLRepository(database)
	first := mustParseCatalog(t, validCatalogJSON("gpt-5.4", "OpenAI"))
	result, err := Apply(ctx, repository, first, 19)
	if err != nil || result.Created != 1 {
		t.Fatalf("首次导入结果=%+v 错误=%v", result, err)
	}
	result, err = Apply(ctx, repository, first, 19)
	if err != nil || result.Unchanged != 1 {
		t.Fatalf("重复导入结果=%+v 错误=%v", result, err)
	}
	assertModelAndPriceVersionCounts(t, ctx, database, "gpt-5.4", 1, 1)

	changedBody := strings.Replace(validCatalogJSON("gpt-5.4", "OpenAI"), `"input_price_micros":2500000`, `"input_price_micros":2500001`, 1)
	changed := mustParseCatalog(t, changedBody)
	result, err = Apply(ctx, repository, changed, 19)
	if err != nil || result.Published != 1 {
		t.Fatalf("调价导入结果=%+v 错误=%v", result, err)
	}
	assertModelAndPriceVersionCounts(t, ctx, database, "gpt-5.4", 1, 2)

	var version, digest string
	if err := database.QueryRowContext(ctx, `
SELECT
    (SELECT value FROM platform_runtime_metadata WHERE key = $1),
    (SELECT value FROM platform_runtime_metadata WHERE key = $2)
`, MetadataVersionKey, MetadataDigestKey).Scan(&version, &digest); err != nil {
		t.Fatalf("读取种子元数据：%v", err)
	}
	if version != changed.Version || digest != changed.Digest {
		t.Fatalf("种子元数据版本=%s 摘要=%s", version, digest)
	}
}

func TestCurrentCatalogAppliesIdempotentlyAgainstPostgres(t *testing.T) {
	databaseURL := os.Getenv("MODELROUTE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("未设置 PostgreSQL 集成测试地址")
	}
	database, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("打开测试数据库：%v", err)
	}
	defer database.Close()
	ctx := context.Background()
	if err := migrations.Up(ctx, database); err != nil {
		t.Fatalf("执行迁移：%v", err)
	}
	catalog, err := CurrentCatalog()
	if err != nil {
		t.Fatal(err)
	}
	for _, model := range catalog.Models {
		cleanupSeedModel(t, ctx, database, model.PublicName)
	}
	defer func() {
		for _, model := range catalog.Models {
			cleanupSeedModel(t, ctx, database, model.PublicName)
		}
	}()

	repository := NewSQLRepository(database)
	result, err := Apply(ctx, repository, catalog, 19)
	if err != nil || result.Created != len(catalog.Models) {
		t.Fatalf("完整种子首次导入结果=%+v 错误=%v", result, err)
	}
	result, err = Apply(ctx, repository, catalog, 19)
	if err != nil || result.Unchanged != len(catalog.Models) {
		t.Fatalf("完整种子重复导入结果=%+v 错误=%v", result, err)
	}

	var models, prices, active, inactive int
	if err := database.QueryRowContext(ctx, `
SELECT
    count(*),
    (SELECT count(*) FROM official_price_versions p JOIN canonical_models c ON c.id = p.canonical_model_id WHERE c.public_name = ANY($1)),
    count(*) FILTER (WHERE status = 'active'),
    count(*) FILTER (WHERE status = 'inactive')
FROM canonical_models
WHERE public_name = ANY($1)
`, catalogModelNames(catalog)).Scan(&models, &prices, &active, &inactive); err != nil {
		t.Fatalf("读取完整种子统计：%v", err)
	}
	if models != 21 || prices != 21 || active != 19 || inactive != 2 {
		t.Fatalf("完整种子统计模型=%d 价格=%d 启用=%d 停用=%d", models, prices, active, inactive)
	}
}

func catalogModelNames(catalog Catalog) []string {
	names := make([]string, 0, len(catalog.Models))
	for _, model := range catalog.Models {
		names = append(names, model.PublicName)
	}
	return names
}

func cleanupSeedModel(t *testing.T, ctx context.Context, database *sql.DB, publicName string) {
	t.Helper()
	statements := []struct {
		query string
		args  []any
	}{
		{`UPDATE canonical_models SET current_official_price_version_id = NULL WHERE public_name = $1`, []any{publicName}},
		{`DELETE FROM official_price_versions WHERE canonical_model_id IN (SELECT id FROM canonical_models WHERE public_name = $1)`, []any{publicName}},
		{`DELETE FROM canonical_models WHERE public_name = $1`, []any{publicName}},
		{`DELETE FROM platform_runtime_metadata WHERE key IN ($1, $2)`, []any{MetadataVersionKey, MetadataDigestKey}},
	}
	for _, statement := range statements {
		if _, err := database.ExecContext(ctx, statement.query, statement.args...); err != nil {
			t.Fatalf("清理模型种子测试数据：%v", err)
		}
	}
}

func assertModelAndPriceVersionCounts(t *testing.T, ctx context.Context, database *sql.DB, publicName string, wantModels int, wantPrices int) {
	t.Helper()
	var models, prices int
	if err := database.QueryRowContext(ctx, `
SELECT
    (SELECT count(*) FROM canonical_models WHERE public_name = $1),
    (SELECT count(*) FROM official_price_versions p JOIN canonical_models m ON m.id = p.canonical_model_id WHERE m.public_name = $1)
`, publicName).Scan(&models, &prices); err != nil {
		t.Fatalf("读取模型种子计数：%v", err)
	}
	if models != wantModels || prices != wantPrices {
		t.Fatalf("模型数=%d 价格版本数=%d，期望 %d/%d", models, prices, wantModels, wantPrices)
	}
}
