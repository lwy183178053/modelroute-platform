package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/lwy183178053/modelroute-platform/apps/api/internal/newapi"
	"github.com/lwy183178053/modelroute-platform/apps/api/internal/platformauth"
)

var internalOfferAliasPattern = regexp.MustCompile(`mkt-(?:ov-[0-9a-fA-F]{24}|o[0-9]+-c[0-9]+)`)

func (s *server) usage(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleCommon)
	if !ok {
		return
	}
	s.writeUsage(w, r, session, false)
}

func (s *server) adminChannelLogs(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleAdmin)
	if !ok {
		return
	}
	if rawChannelID := strings.TrimSpace(r.URL.Query().Get("channel_id")); rawChannelID != "" {
		channelID, err := strconv.ParseInt(rawChannelID, 10, 64)
		lookup, ready := s.services.Merchants.(MerchantChannelLookup)
		if err != nil || channelID <= 0 {
			writeAPIError(w, http.StatusBadRequest, "invalid_channel_id", "渠道 ID 无效")
			return
		}
		if !ready {
			writeAPIError(w, http.StatusServiceUnavailable, "service_not_ready", "渠道服务正在初始化")
			return
		}
		channel, err := lookup.Channel(r.Context(), channelID)
		if err != nil {
			s.writeMerchantStoreError(w, err)
			return
		}
		if rawUserID := strings.TrimSpace(r.URL.Query().Get("merchant_user_id")); rawUserID != "" {
			userID, parseErr := strconv.Atoi(rawUserID)
			if parseErr != nil || userID <= 0 || channel.UserID != userID {
				writeAPIError(w, http.StatusBadRequest, "invalid_merchant_user_id", "商户用户 ID 与渠道不匹配")
				return
			}
		}
		clonedRequest := r.Clone(r.Context())
		clonedURL := *r.URL
		query := clonedURL.Query()
		query.Del("channel_id")
		query.Del("merchant_user_id")
		query.Set("channel", strconv.Itoa(channel.NewAPIChannelID))
		clonedURL.RawQuery = query.Encode()
		clonedRequest.URL = &clonedURL
		r = clonedRequest
	}
	s.writeUsage(w, r, session, true)
}

func (s *server) writeUsage(w http.ResponseWriter, r *http.Request, session platformauth.Session, admin bool) {
	start, end, ok := usageTimeRange(w, r)
	if !ok {
		return
	}
	page, pageSize := pagination(r)
	if pageSize > 50 {
		pageSize = 50
	}

	filters := url.Values{}
	filters.Set("start_timestamp", strconv.FormatInt(start, 10))
	filters.Set("end_timestamp", strconv.FormatInt(end, 10))
	for _, key := range []string{"type", "token_name", "group", "request_id", "upstream_request_id"} {
		if value := strings.TrimSpace(r.URL.Query().Get(key)); value != "" {
			filters.Set(key, value)
		}
	}
	if admin {
		for _, key := range []string{"username", "channel"} {
			if value := strings.TrimSpace(r.URL.Query().Get(key)); value != "" {
				filters.Set(key, value)
			}
		}
	}

	logQuery := cloneValues(filters)
	logQuery.Set("p", strconv.Itoa(page))
	logQuery.Set("page_size", strconv.Itoa(pageSize))
	logPath, statPath, trendPath := "/api/log/self?", "/api/log/self/stat?", "/api/data/self?"
	if admin {
		logPath, statPath, trendPath = "/api/log/?", "/api/log/stat?", "/api/data/?"
	}

	logs, err := s.usageNewAPIDo(r, session, admin, logPath+logQuery.Encode())
	if err != nil {
		writeUpstreamError(w, err, http.StatusBadGateway)
		return
	}
	stats, err := s.usageNewAPIDo(r, session, admin, statPath+filters.Encode())
	if err != nil {
		writeUpstreamError(w, err, http.StatusBadGateway)
		return
	}
	trendQuery := url.Values{
		"start_timestamp": []string{strconv.FormatInt(start, 10)},
		"end_timestamp":   []string{strconv.FormatInt(end, 10)},
	}
	trend, err := s.usageNewAPIDo(r, session, admin, trendPath+trendQuery.Encode())
	if err != nil {
		writeUpstreamError(w, err, http.StatusBadGateway)
		return
	}

	logValue, err := decodeJSONValue(logs)
	if err != nil {
		writeAPIError(w, http.StatusBadGateway, "new_api_contract_error", "核心服务调用日志响应格式无效")
		return
	}
	statValue, err := decodeJSONValue(stats)
	if err != nil {
		writeAPIError(w, http.StatusBadGateway, "new_api_contract_error", "核心服务用量统计响应格式无效")
		return
	}
	trendValue, err := decodeJSONValue(trend)
	if err != nil {
		writeAPIError(w, http.StatusBadGateway, "new_api_contract_error", "核心服务用量趋势响应格式无效")
		return
	}
	aliases := collectInternalAliases(logValue, statValue, trendValue)
	mappings := map[string]string{}
	if len(aliases) > 0 && s.services.UsageAliases != nil {
		mappings, err = s.services.UsageAliases.PublicModelNames(r.Context(), aliases)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "usage_model_mapping_failed", "读取公开模型名称失败")
			return
		}
	}
	logValue = sanitizeInternalAliases(logValue, mappings)
	statValue = sanitizeInternalAliases(statValue, mappings)
	trendValue = sanitizeInternalAliases(trendValue, mappings)
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data": map[string]any{
			"start_timestamp": start,
			"end_timestamp":   end,
			"stats":           statValue,
			"trend":           trendValue,
			"logs":            logValue,
		},
	})
}

func (s *server) usageNewAPIDo(r *http.Request, session platformauth.Session, admin bool, path string) ([]byte, error) {
	if admin {
		if s.services.AdminNewAPI == nil {
			return nil, &newapi.APIError{StatusCode: http.StatusServiceUnavailable, Message: "核心管理服务尚未就绪"}
		}
		return s.services.AdminNewAPI.Do(r.Context(), http.MethodGet, path, nil)
	}
	return s.services.NewAPI.Do(r.Context(), http.MethodGet, path, nil, upstreamAuth(session))
}

func usageTimeRange(w http.ResponseWriter, r *http.Request) (int64, int64, bool) {
	now := time.Now().Unix()
	start := now - 7*24*60*60
	end := now
	var err error
	if raw := strings.TrimSpace(r.URL.Query().Get("start_timestamp")); raw != "" {
		start, err = strconv.ParseInt(raw, 10, 64)
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid_time_range", "开始时间无效")
			return 0, 0, false
		}
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("end_timestamp")); raw != "" {
		end, err = strconv.ParseInt(raw, 10, 64)
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid_time_range", "结束时间无效")
			return 0, 0, false
		}
	}
	if start <= 0 || end <= 0 || end < start || end-start > 31*24*60*60 {
		writeAPIError(w, http.StatusBadRequest, "invalid_time_range", "用量查询时间范围无效或超过 31 天")
		return 0, 0, false
	}
	return start, end, true
}

func cloneValues(source url.Values) url.Values {
	copy := make(url.Values, len(source))
	for key, values := range source {
		copy[key] = append([]string(nil), values...)
	}
	return copy
}

func decodeJSONValue(data []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	return value, nil
}

func sanitizedJSONValue(data []byte) (any, error) {
	value, err := decodeJSONValue(data)
	if err != nil {
		return nil, err
	}
	return sanitizeInternalAliases(value, nil), nil
}

func collectInternalAliases(values ...any) []string {
	seen := make(map[string]struct{})
	var visit func(any)
	visit = func(value any) {
		switch typed := value.(type) {
		case string:
			for _, alias := range internalOfferAliasPattern.FindAllString(typed, -1) {
				seen[alias] = struct{}{}
			}
		case []any:
			for _, item := range typed {
				visit(item)
			}
		case map[string]any:
			for _, item := range typed {
				visit(item)
			}
		}
	}
	for _, value := range values {
		visit(value)
	}
	result := make([]string, 0, len(seen))
	for alias := range seen {
		result = append(result, alias)
	}
	sort.Strings(result)
	return result
}

func sanitizeInternalAliases(value any, mappings map[string]string) any {
	switch typed := value.(type) {
	case string:
		return internalOfferAliasPattern.ReplaceAllStringFunc(typed, func(alias string) string {
			if model := mappings[alias]; model != "" {
				return model
			}
			return "平台路由模型"
		})
	case []any:
		for index := range typed {
			typed[index] = sanitizeInternalAliases(typed[index], mappings)
		}
	case map[string]any:
		for key := range typed {
			typed[key] = sanitizeInternalAliases(typed[key], mappings)
		}
	}
	return value
}
