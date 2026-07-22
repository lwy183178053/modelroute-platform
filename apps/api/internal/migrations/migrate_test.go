package migrations

import "testing"

func TestEmbeddedMigrationsAreOrdered(t *testing.T) {
	files, err := Files()
	if err != nil {
		t.Fatalf("Files() 返回错误：%v", err)
	}
	if len(files) != 1 {
		t.Fatalf("迁移文件数量 = %d", len(files))
	}
	if files[0].Version != 1 || files[0].Name != "000001_foundation.sql" {
		t.Fatalf("迁移文件不符合预期：%+v", files[0])
	}
	if files[0].Checksum == "" || files[0].SQL == "" {
		t.Fatal("迁移内容和校验和必须存在")
	}
}
