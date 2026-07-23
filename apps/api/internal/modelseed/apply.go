package modelseed

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"time"

	"github.com/lwy183178053/modelroute-platform/apps/api/internal/httpapi"
)

const (
	MetadataVersionKey = "official_model_seed_version"
	MetadataDigestKey  = "official_model_seed_digest"
)

type Repository interface {
	ListCanonicalModels(context.Context) ([]httpapi.CanonicalModel, error)
	CreateCanonicalModel(context.Context, httpapi.CreateCanonicalModelCommand) (httpapi.CanonicalModel, error)
	PublishOfficialPrice(context.Context, int64, int, httpapi.OfficialPrice) (httpapi.OfficialPrice, error)
	SetRuntimeMetadata(context.Context, map[string]string) error
}

type Result struct {
	Created   int
	Published int
	Unchanged int
}

func Apply(ctx context.Context, repository Repository, catalog Catalog, administratorID int) (Result, error) {
	if repository == nil || administratorID <= 0 {
		return Result{}, errors.New("模型种子运行参数无效")
	}
	if err := validateCatalog(&catalog); err != nil {
		return Result{}, err
	}
	existing, err := repository.ListCanonicalModels(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("读取现有标准模型：%w", err)
	}
	byName := make(map[string]httpapi.CanonicalModel, len(existing))
	for _, model := range existing {
		byName[model.PublicName] = model
	}
	result := Result{}
	for _, seed := range catalog.Models {
		seed.Price = clonePrice(seed.Price)
		model, exists := byName[seed.PublicName]
		if !exists {
			created, err := repository.CreateCanonicalModel(ctx, httpapi.CreateCanonicalModelCommand{
				PublicName: seed.PublicName,
				Provider:   seed.Provider,
				AdminID:    administratorID,
				Price:      seed.Price,
			})
			if err != nil {
				return Result{}, fmt.Errorf("创建种子模型 %s：%w", seed.PublicName, err)
			}
			byName[seed.PublicName] = created
			result.Created++
			continue
		}
		if model.Provider != seed.Provider {
			return Result{}, fmt.Errorf("模型 %s 的厂商与种子不一致", seed.PublicName)
		}
		if pricesEqual(model.Price, seed.Price) {
			result.Unchanged++
			continue
		}
		if _, err := repository.PublishOfficialPrice(ctx, model.ID, administratorID, seed.Price); err != nil {
			return Result{}, fmt.Errorf("发布种子模型 %s 的新价格：%w", seed.PublicName, err)
		}
		result.Published++
	}
	if err := repository.SetRuntimeMetadata(ctx, map[string]string{
		MetadataVersionKey: catalog.Version,
		MetadataDigestKey:  catalog.Digest,
	}); err != nil {
		return Result{}, fmt.Errorf("记录模型种子版本：%w", err)
	}
	return result, nil
}

func pricesEqual(left httpapi.OfficialPrice, right httpapi.OfficialPrice) bool {
	left = comparablePrice(left)
	right = comparablePrice(right)
	return reflect.DeepEqual(left, right)
}

func comparablePrice(price httpapi.OfficialPrice) httpapi.OfficialPrice {
	price.Tiers = append([]httpapi.PriceTier(nil), price.Tiers...)
	price.ImagePrices = append([]httpapi.ImagePriceItem(nil), price.ImagePrices...)
	price.ID = 0
	price.Version = 0
	price.PublishedAt = time.Time{}
	price.PriceKind = "text"
	if len(price.ImagePrices) > 0 {
		price.PriceKind = "image"
	}
	for index := range price.Tiers {
		price.Tiers[index].ID = 0
		price.Tiers[index].TierOrder = 0
	}
	sort.Slice(price.ImagePrices, func(left int, right int) bool {
		if price.ImagePrices[left].Quality != price.ImagePrices[right].Quality {
			return price.ImagePrices[left].Quality < price.ImagePrices[right].Quality
		}
		return price.ImagePrices[left].Size < price.ImagePrices[right].Size
	})
	return price
}

func clonePrice(price httpapi.OfficialPrice) httpapi.OfficialPrice {
	price.Tiers = append([]httpapi.PriceTier(nil), price.Tiers...)
	price.ImagePrices = append([]httpapi.ImagePriceItem(nil), price.ImagePrices...)
	return price
}
