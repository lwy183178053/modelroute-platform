package modelseed

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"

	"github.com/lwy183178053/modelroute-platform/apps/api/internal/httpapi"
)

const (
	pinnedNewAPIVersion = "v1.0.0-rc.21"
	pinnedNewAPICommit  = "bde9b2f44887d34ec54799ae191d50f97914359e"
	currentCatalogPath  = "data/2026-07-23-v2.json"
)

var (
	//go:embed data/*.json
	catalogFS             embed.FS
	versionPattern        = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}-v[1-9][0-9]*$`)
	officialEvidenceHosts = map[string]string{
		"OpenAI": "developers.openai.com", "Anthropic": "platform.claude.com",
		"Gemini": "ai.google.dev", "DeepSeek": "api-docs.deepseek.com",
		"Qwen": "www.alibabacloud.com", "Moonshot": "platform.kimi.com",
	}
)

type Catalog struct {
	Version            string      `json:"version"`
	NewAPIVersion      string      `json:"new_api_version"`
	NewAPISourceCommit string      `json:"new_api_source_commit"`
	Models             []SeedModel `json:"models"`
	Digest             string      `json:"-"`
}

type SeedModel struct {
	PublicName  string                `json:"public_name"`
	Provider    string                `json:"provider"`
	EvidenceURL string                `json:"evidence_url"`
	Price       httpapi.OfficialPrice `json:"price"`
}

func CurrentCatalog() (Catalog, error) {
	body, err := catalogFS.ReadFile(currentCatalogPath)
	if err != nil {
		return Catalog{}, fmt.Errorf("读取当前模型种子：%w", err)
	}
	return ParseCatalog(body)
}

func ParseCatalog(body []byte) (Catalog, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	var catalog Catalog
	if err := decoder.Decode(&catalog); err != nil {
		return Catalog{}, fmt.Errorf("解析模型种子：%w", err)
	}
	if err := ensureJSONEnd(decoder); err != nil {
		return Catalog{}, err
	}
	hash := sha256.Sum256(body)
	catalog.Digest = hex.EncodeToString(hash[:])
	if err := validateCatalog(&catalog); err != nil {
		return Catalog{}, err
	}
	return catalog, nil
}

func ensureJSONEnd(decoder *json.Decoder) error {
	var extra any
	err := decoder.Decode(&extra)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return errors.New("模型种子只能包含一个 JSON 对象")
	}
	return fmt.Errorf("读取模型种子结尾：%w", err)
}

func validateCatalog(catalog *Catalog) error {
	if !versionPattern.MatchString(catalog.Version) {
		return errors.New("模型种子版本格式无效")
	}
	if len(catalog.Digest) != 64 {
		return errors.New("模型种子摘要无效")
	}
	if catalog.NewAPIVersion != pinnedNewAPIVersion || catalog.NewAPISourceCommit != pinnedNewAPICommit {
		return errors.New("模型种子必须绑定固定 New API rc.21 版本和源码提交")
	}
	if len(catalog.Models) == 0 {
		return errors.New("模型种子不得为空")
	}
	seen := make(map[string]struct{}, len(catalog.Models))
	for index := range catalog.Models {
		model := &catalog.Models[index]
		model.PublicName = strings.TrimSpace(model.PublicName)
		model.Provider = strings.TrimSpace(model.Provider)
		model.EvidenceURL = strings.TrimSpace(model.EvidenceURL)
		if _, exists := seen[model.PublicName]; exists {
			return fmt.Errorf("模型种子存在重复 API 标识：%s", model.PublicName)
		}
		seen[model.PublicName] = struct{}{}
		if !supportedByPinnedNewAPI(model.Provider, model.PublicName) {
			return fmt.Errorf("模型 %s 未进入固定 New API rc.21 支持清单", model.PublicName)
		}
		source, err := url.Parse(model.EvidenceURL)
		if err != nil || source.Scheme != "https" || source.Host == "" {
			return fmt.Errorf("模型 %s 的核价来源无效", model.PublicName)
		}
		if source.Hostname() != officialEvidenceHosts[model.Provider] {
			return fmt.Errorf("模型 %s 的核价来源不是 %s 官方域名", model.PublicName, model.Provider)
		}
		command := httpapi.CreateCanonicalModelCommand{
			PublicName: model.PublicName,
			Provider:   model.Provider,
			AdminID:    1,
			Price:      model.Price,
		}
		if err := httpapi.ValidateCanonicalModel(command); err != nil {
			return fmt.Errorf("模型 %s 的字段或价格无效：%w", model.PublicName, err)
		}
	}
	return nil
}
