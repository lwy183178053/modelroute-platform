package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

func ReadCoreQuotaPerUnit(ctx context.Context, client NewAPIClient) (int64, error) {
	if client == nil {
		return 0, errors.New("核心额度单位读取服务未配置")
	}
	data, err := client.Do(ctx, http.MethodGet, "/api/status", nil, nil)
	if err != nil {
		return 0, fmt.Errorf("读取核心额度单位：%w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var status struct {
		QuotaPerUnit json.Number `json:"quota_per_unit"`
	}
	if err := decoder.Decode(&status); err != nil {
		return 0, fmt.Errorf("解析核心额度单位：%w", err)
	}
	unit, err := status.QuotaPerUnit.Int64()
	if err != nil || unit <= 0 || unit%100 != 0 {
		return 0, errors.New("核心额度单位必须是能精确表示人民币分的正整数")
	}
	return unit, nil
}
