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

	"github.com/lwy183178053/modelroute-platform/apps/api/internal/platformauth"
)

var internalOfferAliasPattern = regexp.MustCompile(`mkt-(?:ov-[0-9a-fA-F]{24}|o[0-9]+-c[0-9]+)`)

func (s *server) usage(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleCommon)
	if !ok {
		return
	}
	s.writeUserUsage(w, r, session)
}

func (s *server) merchantUsage(w http.ResponseWriter, r *http.Request) {
	_, session, _, ok := s.requireMerchant(w, r)
	if !ok {
		return
	}
	query, ok := usageRecordQueryFromRequest(w, r)
	if !ok {
		return
	}
	if s.services.UsageRecords == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "usage_records_unavailable", "调用记录服务尚未就绪")
		return
	}
	query.MerchantUserID = session.UserID
	query.FilterMerchantUserID = 0
	page, err := s.services.UsageRecords.ListUsageRecords(r.Context(), query)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "usage_records_query_failed", "读取商户调用记录失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{
		"logs": page, "filters": page.Filters,
	}})
}

func (s *server) writeUserUsage(w http.ResponseWriter, r *http.Request, session platformauth.Session) {
	start, end, ok := usageTimeRange(w, r)
	if !ok {
		return
	}
	query, ok := usageRecordQueryFromRequest(w, r)
	if !ok {
		return
	}
	if s.services.UsageRecords == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "usage_records_unavailable", "调用记录服务尚未就绪")
		return
	}
	query.ConsumerUserID = session.UserID
	page, err := s.services.UsageRecords.ListUsageRecords(r.Context(), query)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "usage_records_query_failed", "读取用户调用记录失败")
		return
	}
	filters := url.Values{
		"start_timestamp": []string{strconv.FormatInt(start, 10)},
		"end_timestamp":   []string{strconv.FormatInt(end, 10)},
		"type":            []string{"2"},
	}
	stats, err := s.usageNewAPIDo(r, session, false, "/api/log/self/stat?"+filters.Encode())
	if err != nil {
		writeUpstreamError(w, err, http.StatusBadGateway)
		return
	}
	trendQuery := url.Values{
		"start_timestamp": []string{strconv.FormatInt(start, 10)},
		"end_timestamp":   []string{strconv.FormatInt(end, 10)},
	}
	trend, err := s.usageNewAPIDo(r, session, false, "/api/data/self?"+trendQuery.Encode())
	if err != nil {
		writeUpstreamError(w, err, http.StatusBadGateway)
		return
	}
	statValue, err := decodeJSONValue(stats)
	if err != nil {
		writeAPIError(w, http.StatusBadGateway, "usage_contract_error", "核心服务用量统计响应格式无效")
		return
	}
	trendValue, err := decodeJSONValue(trend)
	if err != nil {
		writeAPIError(w, http.StatusBadGateway, "usage_contract_error", "核心服务用量趋势响应格式无效")
		return
	}
	aliases := collectInternalAliases(trendValue)
	mappings := map[string]string{}
	if len(aliases) > 0 && s.services.UsageAliases != nil {
		mappings, err = s.services.UsageAliases.PublicModelNames(r.Context(), aliases)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "usage_model_mapping_failed", "读取公开模型名称失败")
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data": map[string]any{
			"start_timestamp": start,
			"end_timestamp":   end,
			"stats":           statValue,
			"trend":           sanitizeInternalAliases(trendValue, mappings),
			"logs":            page,
			"filters":         page.Filters,
		},
	})
}

func usageRecordQueryFromRequest(w http.ResponseWriter, r *http.Request) (UsageRecordQuery, bool) {
	page, pageSize := pagination(r)
	query := UsageRecordQuery{
		ModelName: strings.TrimSpace(r.URL.Query().Get("model_name")),
		Page:      page,
		PageSize:  pageSize,
	}
	if len(query.ModelName) > 255 {
		writeAPIError(w, http.StatusBadRequest, "invalid_model_name", "模型筛选名称无效")
		return UsageRecordQuery{}, false
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("merchant_user_id")); raw != "" {
		merchantUserID, err := strconv.Atoi(raw)
		if err != nil || merchantUserID <= 0 {
			writeAPIError(w, http.StatusBadRequest, "invalid_merchant_user_id", "商户用户 ID 无效")
			return UsageRecordQuery{}, false
		}
		query.FilterMerchantUserID = merchantUserID
	}
	var err error
	if raw := strings.TrimSpace(r.URL.Query().Get("start_timestamp")); raw != "" {
		query.StartTimestamp, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || query.StartTimestamp <= 0 {
			writeAPIError(w, http.StatusBadRequest, "invalid_time_range", "开始时间无效")
			return UsageRecordQuery{}, false
		}
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("end_timestamp")); raw != "" {
		query.EndTimestamp, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || query.EndTimestamp <= 0 {
			writeAPIError(w, http.StatusBadRequest, "invalid_time_range", "结束时间无效")
			return UsageRecordQuery{}, false
		}
	}
	if query.StartTimestamp > 0 && query.EndTimestamp > 0 && query.EndTimestamp < query.StartTimestamp {
		writeAPIError(w, http.StatusBadRequest, "invalid_time_range", "调用记录时间范围无效")
		return UsageRecordQuery{}, false
	}
	return query, true
}

func (s *server) adminChannelLogs(w http.ResponseWriter, r *http.Request) {
	_, _, ok := s.requireSession(w, r, roleAdmin)
	if !ok {
		return
	}
	query, ok := usageRecordQueryFromRequest(w, r)
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
		query.ChannelID = channelID
	}
	if s.services.UsageRecords == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "usage_records_unavailable", "调用记录服务尚未就绪")
		return
	}
	query.Admin = true
	page, err := s.services.UsageRecords.ListUsageRecords(r.Context(), query)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "usage_records_query_failed", "读取渠道调用记录失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{
		"logs": page, "filters": page.Filters,
	}})
}

func (s *server) usageNewAPIDo(r *http.Request, session platformauth.Session, _ bool, path string) ([]byte, error) {
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

func decodeJSONValue(data []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	return value, nil
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
