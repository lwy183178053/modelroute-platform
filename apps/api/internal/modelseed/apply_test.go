package modelseed

import (
	"context"
	"testing"

	"github.com/lwy183178053/modelroute-platform/apps/api/internal/httpapi"
)

type repositoryStub struct {
	models        []httpapi.CanonicalModel
	created       []httpapi.CreateCanonicalModelCommand
	published     []publishedPrice
	metadata      map[string]string
	nextModelID   int64
	nextVersionID int64
}

type publishedPrice struct {
	modelID int64
	adminID int
	price   httpapi.OfficialPrice
}

func (s *repositoryStub) ListCanonicalModels(context.Context) ([]httpapi.CanonicalModel, error) {
	return append([]httpapi.CanonicalModel(nil), s.models...), nil
}

func (s *repositoryStub) CreateCanonicalModel(_ context.Context, command httpapi.CreateCanonicalModelCommand) (httpapi.CanonicalModel, error) {
	s.nextModelID++
	model := httpapi.CanonicalModel{
		ID: commandID(s.nextModelID), PublicName: command.PublicName, Provider: command.Provider,
		Status: "active", Price: command.Price,
	}
	model.Price.Version = 1
	s.created = append(s.created, command)
	s.models = append(s.models, model)
	return model, nil
}

func (s *repositoryStub) PublishOfficialPrice(_ context.Context, modelID int64, adminID int, price httpapi.OfficialPrice) (httpapi.OfficialPrice, error) {
	s.nextVersionID++
	price.Version = int(s.nextVersionID + 1)
	s.published = append(s.published, publishedPrice{modelID: modelID, adminID: adminID, price: price})
	for index := range s.models {
		if s.models[index].ID == modelID {
			s.models[index].Price = price
		}
	}
	return price, nil
}

func (s *repositoryStub) SetRuntimeMetadata(_ context.Context, values map[string]string) error {
	if s.metadata == nil {
		s.metadata = map[string]string{}
	}
	for key, value := range values {
		s.metadata[key] = value
	}
	return nil
}

func TestApplyCreatesModelsAndRecordsSeed(t *testing.T) {
	catalog := mustParseCatalog(t, validCatalogJSON("gpt-5.4", "OpenAI"))
	repository := &repositoryStub{}

	result, err := Apply(context.Background(), repository, catalog, 19)
	if err != nil {
		t.Fatalf("应用种子：%v", err)
	}
	if result.Created != 1 || result.Published != 0 || result.Unchanged != 0 {
		t.Fatalf("首次导入结果=%+v", result)
	}
	if len(repository.created) != 1 || repository.created[0].AdminID != 19 {
		t.Fatalf("创建命令=%+v", repository.created)
	}
	if repository.metadata[MetadataVersionKey] != catalog.Version || repository.metadata[MetadataDigestKey] != catalog.Digest {
		t.Fatalf("种子元数据=%+v", repository.metadata)
	}
}

func TestApplySkipsUnchangedPriceOnRepeatedRun(t *testing.T) {
	catalog := mustParseCatalog(t, validCatalogJSON("gpt-5.4", "OpenAI"))
	repository := &repositoryStub{}
	if _, err := Apply(context.Background(), repository, catalog, 19); err != nil {
		t.Fatal(err)
	}

	result, err := Apply(context.Background(), repository, catalog, 19)
	if err != nil {
		t.Fatal(err)
	}
	if result.Unchanged != 1 || len(repository.created) != 1 || len(repository.published) != 0 {
		t.Fatalf("重复导入不应创建价格版本：结果=%+v 创建=%d 发布=%d", result, len(repository.created), len(repository.published))
	}
}

func TestPricesEqualTreatsDerivedPriceKindAsEquivalent(t *testing.T) {
	output := int64(3_000_000)
	seed := httpapi.OfficialPrice{Tiers: []httpapi.PriceTier{{MinimumInputTokens: 0, InputPriceMicros: 1_000_000, OutputPriceMicros: &output}}}
	stored := seed
	stored.PriceKind = "text"
	if !pricesEqual(stored, seed) {
		t.Fatal("数据库派生的 text 价格类型不应导致种子重复发布")
	}
}

func TestApplyPublishesNewPriceWithoutChangingModelIdentity(t *testing.T) {
	catalog := mustParseCatalog(t, validCatalogJSON("gpt-5.4", "OpenAI"))
	repository := &repositoryStub{}
	if _, err := Apply(context.Background(), repository, catalog, 19); err != nil {
		t.Fatal(err)
	}
	catalog.Models[0].Price.Tiers[0].InputPriceMicros++

	result, err := Apply(context.Background(), repository, catalog, 19)
	if err != nil {
		t.Fatal(err)
	}
	if result.Published != 1 || len(repository.created) != 1 || len(repository.published) != 1 {
		t.Fatalf("调价应只追加价格版本：结果=%+v 创建=%d 发布=%d", result, len(repository.created), len(repository.published))
	}
	if repository.published[0].modelID != repository.models[0].ID || repository.published[0].adminID != 19 {
		t.Fatalf("价格发布参数=%+v", repository.published[0])
	}
}

func mustParseCatalog(t *testing.T, body string) Catalog {
	t.Helper()
	catalog, err := ParseCatalog([]byte(body))
	if err != nil {
		t.Fatalf("解析测试种子：%v", err)
	}
	return catalog
}

func commandID(value int64) int64 {
	if value == 0 {
		return 1
	}
	return value
}
