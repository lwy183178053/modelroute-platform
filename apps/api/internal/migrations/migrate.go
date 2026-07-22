package migrations

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
)

//go:embed sql/*.sql
var migrationFS embed.FS

type File struct {
	Version  int
	Name     string
	Checksum string
	SQL      string
}

func Files() ([]File, error) {
	paths, err := fs.Glob(migrationFS, "sql/*.sql")
	if err != nil {
		return nil, fmt.Errorf("读取迁移目录：%w", err)
	}
	sort.Strings(paths)

	files := make([]File, 0, len(paths))
	for _, path := range paths {
		name := strings.TrimPrefix(path, "sql/")
		if len(name) < 7 || name[6] != '_' {
			return nil, fmt.Errorf("迁移文件名格式错误：%s", name)
		}
		version, err := strconv.Atoi(name[:6])
		if err != nil || version != len(files)+1 {
			return nil, fmt.Errorf("迁移版本必须从 000001 连续编号：%s", name)
		}
		body, err := migrationFS.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("读取迁移文件 %s：%w", name, err)
		}
		hash := sha256.Sum256(body)
		files = append(files, File{
			Version:  version,
			Name:     name,
			Checksum: hex.EncodeToString(hash[:]),
			SQL:      string(body),
		})
	}
	return files, nil
}

func Up(ctx context.Context, db *sql.DB) error {
	files, err := Files()
	if err != nil {
		return err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开始迁移事务：%w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(782403221)`); err != nil {
		return fmt.Errorf("获取迁移锁：%w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS platform_schema_migrations (
			version integer PRIMARY KEY,
			name text NOT NULL,
			checksum text NOT NULL,
			applied_at timestamptz NOT NULL DEFAULT now()
		)`); err != nil {
		return fmt.Errorf("创建迁移记录表：%w", err)
	}

	for _, file := range files {
		var checksum string
		err := tx.QueryRowContext(ctx,
			`SELECT checksum FROM platform_schema_migrations WHERE version = $1`,
			file.Version,
		).Scan(&checksum)
		switch {
		case err == nil && checksum != file.Checksum:
			return fmt.Errorf("已执行迁移 %s 的校验和发生变化", file.Name)
		case err == nil:
			continue
		case err != sql.ErrNoRows:
			return fmt.Errorf("查询迁移 %s：%w", file.Name, err)
		}

		if _, err := tx.ExecContext(ctx, file.SQL); err != nil {
			return fmt.Errorf("执行迁移 %s：%w", file.Name, err)
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO platform_schema_migrations (version, name, checksum) VALUES ($1, $2, $3)`,
			file.Version, file.Name, file.Checksum,
		); err != nil {
			return fmt.Errorf("记录迁移 %s：%w", file.Name, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交迁移事务：%w", err)
	}
	return nil
}
