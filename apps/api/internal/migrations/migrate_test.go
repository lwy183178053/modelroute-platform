package migrations

import (
	"strings"
	"testing"
)

func TestEmbeddedMigrationsAreOrdered(t *testing.T) {
	files, err := Files()
	if err != nil {
		t.Fatalf("Files() 返回错误：%v", err)
	}
	if len(files) != 2 {
		t.Fatalf("迁移文件数量 = %d", len(files))
	}
	if files[0].Version != 1 || files[0].Name != "000001_foundation.sql" {
		t.Fatalf("迁移文件不符合预期：%+v", files[0])
	}
	if files[0].Checksum == "" || files[0].SQL == "" {
		t.Fatal("迁移内容和校验和必须存在")
	}
	if files[1].Version != 2 || files[1].Name != "000002_merchants.sql" {
		t.Fatalf("商户迁移不符合预期：%+v", files[1])
	}
	for _, table := range []string{"merchant_applications", "merchant_profiles", "merchant_channel_reviews", "merchant_channels"} {
		if !strings.Contains(files[1].SQL, "CREATE TABLE "+table) {
			t.Fatalf("商户迁移缺少表 %s", table)
		}
	}
}
