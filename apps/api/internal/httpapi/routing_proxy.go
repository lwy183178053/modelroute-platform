package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
)

const maxModelRequestBytes = 64 << 20
const maxModelListBytes = 4 << 20
const routeAlgorithmVersion = "weighted-v2-merchant-rounds"

func (s *server) modelListProxy(w http.ResponseWriter, r *http.Request) {
	if s.services.NewAPI == nil {
		writeOpenAIError(w, http.StatusServiceUnavailable, "service_not_ready", "模型列表服务尚未就绪")
		return
	}
	bearer, ok := bearerToken(r.Header.Get("Authorization"))
	if !ok {
		writeOpenAIError(w, http.StatusUnauthorized, "invalid_api_key", "缺少有效的 Bearer 密钥")
		return
	}
	response, err := s.services.NewAPI.Proxy(r.Context(), r.Method, r.URL.Path, r.Header, nil)
	if err != nil {
		writeOpenAIError(w, http.StatusBadGateway, "upstream_connection_error", "模型列表服务连接失败")
		return
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, maxModelListBytes+1))
	if err != nil || len(body) > maxModelListBytes {
		writeOpenAIError(w, http.StatusBadGateway, "upstream_response_error", "模型列表响应无效")
		return
	}
	if response.StatusCode >= http.StatusBadRequest {
		copyResponseHeaders(w.Header(), response.Header)
		w.WriteHeader(response.StatusCode)
		_, _ = w.Write(body)
		return
	}
	value, err := decodeJSONValue(body)
	if err != nil {
		writeOpenAIError(w, http.StatusBadGateway, "upstream_response_error", "模型列表响应格式无效")
		return
	}
	aliases := collectInternalAliases(value)
	mappings := map[string]string{}
	if len(aliases) > 0 && s.services.UsageAliases != nil {
		mappings, err = s.services.UsageAliases.PublicModelNames(r.Context(), aliases)
		if err != nil {
			writeOpenAIError(w, http.StatusBadGateway, "model_mapping_error", "模型列表映射失败")
			return
		}
	}
	value = sanitizeInternalAliases(value, mappings)
	if s.services.Routing != nil && s.services.Credentials != nil {
		value = s.filterRoutableModelList(r, value, s.services.Credentials.Fingerprint([]byte(bearer)))
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		writeOpenAIError(w, http.StatusBadGateway, "upstream_response_error", "模型列表响应编码失败")
		return
	}
	copyResponseHeaders(w.Header(), response.Header)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(response.StatusCode)
	_, _ = io.Copy(w, bytes.NewReader(encoded))
}

func (s *server) filterRoutableModelList(r *http.Request, value any, fingerprint string) any {
	filter := func(items []any) []any {
		result := make([]any, 0, len(items))
		for _, item := range items {
			object, ok := item.(map[string]any)
			if !ok {
				continue
			}
			name, ok := object["id"].(string)
			if !ok || strings.TrimSpace(name) == "" {
				continue
			}
			if _, err := s.services.Routing.ResolveRoute(r.Context(), fingerprint, name); err == nil {
				result = append(result, item)
			}
		}
		return result
	}
	switch typed := value.(type) {
	case map[string]any:
		if items, ok := typed["data"].([]any); ok {
			typed["data"] = filter(items)
		}
	case []any:
		return filter(typed)
	}
	return value
}

func (s *server) modelProxy(w http.ResponseWriter, r *http.Request) {
	if s.services.Routing == nil || s.services.Credentials == nil || s.services.NewAPI == nil {
		writeOpenAIError(w, http.StatusServiceUnavailable, "service_not_ready", "模型路由服务尚未就绪")
		return
	}
	token, ok := bearerToken(r.Header.Get("Authorization"))
	if !ok {
		writeOpenAIError(w, http.StatusUnauthorized, "invalid_api_key", "缺少有效的 Bearer 密钥")
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxModelRequestBytes))
	if err != nil {
		writeOpenAIError(w, http.StatusBadRequest, "invalid_request_error", "模型请求正文无效或超过限制")
		return
	}
	_, publicModel, err := rewritePublicModel(body, "pending-route")
	if err != nil {
		writeOpenAIError(w, http.StatusBadRequest, "invalid_request_error", err.Error())
		return
	}
	selection, err := s.services.Routing.ResolveRoute(r.Context(), s.services.Credentials.Fingerprint([]byte(token)), publicModel)
	if err != nil {
		code := "routing_policy_not_found"
		if !errors.Is(err, ErrRoutingNotFound) {
			code = "routing_store_error"
		}
		writeOpenAIError(w, http.StatusServiceUnavailable, code, "当前密钥没有可用路由策略")
		return
	}
	if len(selection.Candidates) == 0 {
		writeOpenAIError(w, http.StatusServiceUnavailable, "no_available_channel", "当前策略范围内没有可用渠道")
		return
	}
	requestID, _ := r.Context().Value(requestIDContextKey{}).(string)
	if err := s.services.Routing.StartRouteDecision(r.Context(), RouteDecision{
		RequestID: requestID, TokenBindingID: selection.TokenBindingID,
		PolicyID:         selection.PolicyID,
		CanonicalModelID: selection.CanonicalModelID, OfferChannelAliasID: selection.Candidates[0].OfferChannelAliasID,
		Priority: selection.Priority, AlgorithmVersion: routeAlgorithmVersion,
		CandidateCount: len(selection.Candidates), SelectedMetrics: routeDecisionMetrics(selection.Candidates[0]),
	}); err != nil {
		writeOpenAIError(w, http.StatusServiceUnavailable, "routing_store_error", "记录路由决策失败")
		return
	}

	for index, candidate := range selection.Candidates {
		rewritten, _, err := rewritePublicModel(body, candidate.InternalAlias)
		if err != nil {
			writeOpenAIError(w, http.StatusBadRequest, "invalid_request_error", err.Error())
			return
		}
		started := time.Now()
		response, err := s.services.NewAPI.Proxy(r.Context(), r.Method, r.URL.Path, r.Header, rewritten)
		if err != nil {
			observation := RouteObservation{Candidate: candidate, Outcome: "temporary_failure", RequestID: requestID, Summary: "核心服务连接失败"}
			_ = s.services.Routing.RecordRouteObservation(r.Context(), observation)
			if selection.PriceKind != "image" && index+1 < len(selection.Candidates) {
				continue
			}
			_ = s.services.Routing.CompleteRouteDecision(r.Context(), RouteDecisionResult{RequestID: requestID, OfferChannelAliasID: candidate.OfferChannelAliasID, Attempts: index + 1, Result: "transport_error"})
			writeOpenAIError(w, http.StatusBadGateway, "upstream_connection_error", "模型服务连接失败")
			return
		}

		first := make([]byte, 32*1024)
		n, readErr := response.Body.Read(first)
		firstByteMS := int(time.Since(started).Milliseconds())
		outcome := routeOutcome(response.StatusCode)
		_ = s.services.Routing.RecordRouteObservation(r.Context(), RouteObservation{
			Candidate: candidate, Outcome: outcome, StatusCode: response.StatusCode,
			FirstByteMS: firstByteMS, RequestID: requestID,
		})
		if retryableStatus(response.StatusCode) && index+1 < len(selection.Candidates) {
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
			continue
		}

		copyResponseHeaders(w.Header(), response.Header)
		w.WriteHeader(response.StatusCode)
		if n > 0 {
			_, _ = w.Write(first[:n])
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		}
		copyErr := readErr
		if readErr == nil {
			copyErr = copyStreaming(w, response.Body)
		}
		_ = response.Body.Close()
		result := "completed"
		if copyErr != nil && !errors.Is(copyErr, io.EOF) {
			result = "stream_error"
		}
		_ = s.services.Routing.CompleteRouteDecision(r.Context(), RouteDecisionResult{
			RequestID: requestID, OfferChannelAliasID: candidate.OfferChannelAliasID, Attempts: index + 1,
			StatusCode: response.StatusCode, FirstByteMS: firstByteMS, Result: result,
		})
		return
	}
}

func routeDecisionMetrics(candidate RouteCandidate) map[string]any {
	return map[string]any{
		"merchant_id":                 candidate.MerchantID,
		"merchant_channel_id":         candidate.MerchantChannelID,
		"multiplier_bps":              candidate.MultiplierBPS,
		"availability_ppm":            candidate.AvailabilityPPM,
		"ttft_p95_ms":                 candidate.TTFTP95MS,
		"cache_hit_ppm":               candidate.CacheHitPPM,
		"cache_weight_ready":          candidate.CacheWeightReady,
		"merchant_cache_hit_ppm":      candidate.MerchantCacheHitPPM,
		"merchant_cache_weight_ready": candidate.MerchantCacheWeightReady,
	}
}

func bearerToken(header string) (string, bool) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}

func retryableStatus(status int) bool { return status == http.StatusTooManyRequests || status >= 500 }

func routeOutcome(status int) string {
	switch classifyProbeStatus(status) {
	case healthFailureNone:
		return "success"
	case healthFailureTemporary:
		return "temporary_failure"
	case healthFailureHard:
		return "hard_failure"
	default:
		return "ignored"
	}
}

func copyResponseHeaders(target http.Header, source http.Header) {
	for name, values := range source {
		if isProxyHopHeader(name) {
			continue
		}
		for _, value := range values {
			target.Add(name, value)
		}
	}
}

func isProxyHopHeader(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "connection", "proxy-connection", "keep-alive", "proxy-authenticate", "proxy-authorization", "te", "trailer", "transfer-encoding", "upgrade":
		return true
	default:
		return false
	}
}

func copyStreaming(w http.ResponseWriter, body io.Reader) error {
	buffer := make([]byte, 32*1024)
	for {
		n, err := body.Read(buffer)
		if n > 0 {
			if _, writeErr := w.Write(buffer[:n]); writeErr != nil {
				return writeErr
			}
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		}
		if err != nil {
			return err
		}
	}
}

func writeOpenAIError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]any{
		"message": message, "type": "server_error", "param": nil, "code": code,
	}})
}
