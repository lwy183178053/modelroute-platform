package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"sort"
	"strconv"
	"strings"
	"time"
)

type HealthProbe struct {
	OfferChannelAliasID int64
	OfferID             int64
	MerchantChannelID   int64
	CanonicalModelID    int64
	NewAPIChannelID     int
	InternalAlias       string
	PriceKind           string
	State               string
	ConsecutiveFailures int
	CooldownLevel       int
	NextProbeAt         time.Time
}

type RouteCandidate struct {
	OfferID                  int64  `json:"offer_id"`
	OfferChannelAliasID      int64  `json:"offer_channel_alias_id"`
	MerchantID               int64  `json:"merchant_id"`
	MerchantChannelID        int64  `json:"merchant_channel_id"`
	NewAPIChannelID          int    `json:"newapi_channel_id"`
	CanonicalModelID         int64  `json:"canonical_model_id"`
	InternalAlias            string `json:"-"`
	PublicModel              string `json:"model"`
	MultiplierBPS            int    `json:"multiplier_bps"`
	HealthState              string `json:"health_state"`
	AvailabilityPPM          int    `json:"availability_ppm"`
	TTFTP95MS                int    `json:"ttft_p95_ms"`
	MetricRequestCount       int64  `json:"metric_request_count"`
	CacheHitPPM              int    `json:"cache_hit_ppm"`
	CacheSampleInputTokens   int64  `json:"cache_sample_input_tokens"`
	CacheWeightReady         bool   `json:"cache_weight_ready"`
	MerchantCacheHitPPM      int    `json:"merchant_cache_hit_ppm"`
	MerchantCacheWeightReady bool   `json:"merchant_cache_weight_ready"`
}

func routeCandidateMeetsCacheFloor(selection RouteSelection, candidate RouteCandidate) bool {
	if selection.MinimumCacheHitPPM == nil || selection.PriceKind == "image" {
		return true
	}
	return candidate.MerchantCacheWeightReady && candidate.MerchantCacheHitPPM >= *selection.MinimumCacheHitPPM
}

type healthFailureClass string

const (
	healthFailureNone      healthFailureClass = "none"
	healthFailureIgnored   healthFailureClass = "ignored"
	healthFailureTemporary healthFailureClass = "temporary"
	healthFailureHard      healthFailureClass = "hard"
)

func scaledDecimal(value int64, scale int64) string {
	if scale <= 0 {
		return "0"
	}
	whole := value / scale
	fraction := value % scale
	if fraction == 0 {
		return strconv.FormatInt(whole, 10)
	}
	digits := len(strconv.FormatInt(scale, 10)) - 1
	fractionText := strings.TrimRight(fmt.Sprintf("%0*d", digits, fraction), "0")
	return strconv.FormatInt(whole, 10) + "." + fractionText
}

func rewritePublicModel(body []byte, internalAlias string) ([]byte, string, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	var request map[string]json.RawMessage
	if err := decoder.Decode(&request); err != nil {
		return nil, "", errors.New("请求正文不是有效 JSON")
	}
	if decoder.More() {
		return nil, "", errors.New("请求正文只能包含一个 JSON 对象")
	}
	rawModel, ok := request["model"]
	if !ok {
		return nil, "", errors.New("请求缺少 model")
	}
	var publicModel string
	if err := json.Unmarshal(rawModel, &publicModel); err != nil || strings.TrimSpace(publicModel) == "" {
		return nil, "", errors.New("model 必须是非空字符串")
	}
	encodedAlias, _ := json.Marshal(internalAlias)
	request["model"] = encodedAlias
	rewritten, err := json.Marshal(request)
	if err != nil {
		return nil, "", err
	}
	return rewritten, publicModel, nil
}

func classifyProbeStatus(status int) healthFailureClass {
	switch {
	case status >= 200 && status < 300:
		return healthFailureNone
	case status == 401 || status == 402 || status == 403:
		return healthFailureHard
	case status == 429 || status >= 500:
		return healthFailureTemporary
	default:
		return healthFailureIgnored
	}
}

const cacheWeightMinimumInputTokens int64 = 1_000_000

type routeRandom interface{ Float64() float64 }

type systemRouteRandom struct{}

func (systemRouteRandom) Float64() float64 { return rand.Float64() }

type rankedMetric struct {
	id    int64
	value float64
	known bool
}

type merchantRouteMetric struct {
	id                                     int64
	price, success, latency, cache         float64
	successKnown, latencyKnown, cacheKnown bool
}

func merchantCandidateWeights(candidates []RouteCandidate, priority string) map[int64]float64 {
	grouped := make(map[int64][]RouteCandidate)
	for _, candidate := range candidates {
		grouped[candidate.MerchantID] = append(grouped[candidate.MerchantID], candidate)
	}
	metrics := make([]merchantRouteMetric, 0, len(grouped))
	for merchantID, items := range grouped {
		metric := merchantRouteMetric{id: merchantID, price: float64(items[0].MultiplierBPS)}
		var requestTotal, latencyTotal, cacheInput, cacheHit int64
		for _, item := range items {
			if float64(item.MultiplierBPS) < metric.price {
				metric.price = float64(item.MultiplierBPS)
			}
			if item.MetricRequestCount > 0 {
				requestTotal += item.MetricRequestCount
				metric.success += float64(item.AvailabilityPPM) * float64(item.MetricRequestCount)
				if item.TTFTP95MS > 0 {
					latencyTotal += int64(item.TTFTP95MS) * item.MetricRequestCount
				}
			}
			if item.CacheSampleInputTokens > 0 {
				cacheInput += item.CacheSampleInputTokens
				cacheHit += int64(item.CacheHitPPM) * item.CacheSampleInputTokens
			}
		}
		if requestTotal > 0 {
			metric.success = metric.success / float64(requestTotal)
			metric.successKnown = true
			if latencyTotal > 0 {
				metric.latency = float64(latencyTotal) / float64(requestTotal)
				metric.latencyKnown = true
			}
		}
		if cacheInput >= cacheWeightMinimumInputTokens {
			metric.cache = float64(cacheHit) / float64(cacheInput)
			metric.cacheKnown = true
		}
		metrics = append(metrics, metric)
	}
	priceScores := rankMetricScores(metricsToRanked(metrics, func(item merchantRouteMetric) (float64, bool) { return item.price, true }), false)
	successScores := rankMetricScores(metricsToRanked(metrics, func(item merchantRouteMetric) (float64, bool) { return item.success, item.successKnown }), true)
	latencyScores := rankMetricScores(metricsToRanked(metrics, func(item merchantRouteMetric) (float64, bool) { return item.latency, item.latencyKnown }), false)
	cacheScores := rankMetricScores(metricsToRanked(metrics, func(item merchantRouteMetric) (float64, bool) { return item.cache, item.cacheKnown }), true)
	weights := make(map[int64]float64, len(metrics))
	for _, metric := range metrics {
		score := compositeRouteScore(priority, priceScores[metric.id], successScores[metric.id], latencyScores[metric.id], cacheScores[metric.id])
		weights[metric.id] = 1 + 3*score
	}
	return weights
}

func metricsToRanked(metrics []merchantRouteMetric, value func(merchantRouteMetric) (float64, bool)) []rankedMetric {
	result := make([]rankedMetric, 0, len(metrics))
	for _, metric := range metrics {
		metricValue, known := value(metric)
		result = append(result, rankedMetric{id: metric.id, value: metricValue, known: known})
	}
	return result
}

func rankMetricScores(metrics []rankedMetric, higherBetter bool) map[int64]float64 {
	known := make([]rankedMetric, 0, len(metrics))
	result := make(map[int64]float64, len(metrics))
	for _, metric := range metrics {
		result[metric.id] = 0.5
		if metric.known {
			known = append(known, metric)
		}
	}
	if len(known) == 0 {
		return result
	}
	sort.Slice(known, func(i, j int) bool {
		if higherBetter {
			return known[i].value > known[j].value
		}
		return known[i].value < known[j].value
	})
	if len(known) == 1 {
		result[known[0].id] = 1
		return result
	}
	for start := 0; start < len(known); {
		end := start
		for end+1 < len(known) && known[end+1].value == known[start].value {
			end++
		}
		averageRank := float64(start+end) / 2
		score := 1 - averageRank/float64(len(known)-1)
		for index := start; index <= end; index++ {
			result[known[index].id] = score
		}
		start = end + 1
	}
	return result
}

func compositeRouteScore(priority string, price, success, latency, cache float64) float64 {
	if priority == "balanced" {
		return (price + success + latency + cache) / 4
	}
	values := map[string]float64{"price": price, "success": success, "latency": latency, "cache": cache}
	primary, exists := values[priority]
	if !exists {
		return (price + success + latency + cache) / 4
	}
	return primary*0.7 + (price+success+latency+cache-primary)*0.1
}

func weightedCandidateOrder(candidates []RouteCandidate, priority string, random routeRandom) []RouteCandidate {
	remaining := append([]RouteCandidate(nil), candidates...)
	ordered := make([]RouteCandidate, 0, len(candidates))
	for len(remaining) > 0 {
		merchantWeights := merchantCandidateWeights(remaining, priority)
		merchantIDs := make([]int64, 0, len(merchantWeights))
		for merchantID := range merchantWeights {
			merchantIDs = append(merchantIDs, merchantID)
		}
		sort.Slice(merchantIDs, func(i, j int) bool { return merchantIDs[i] < merchantIDs[j] })
		for len(merchantIDs) > 0 {
			merchantOffset := weightedIDIndex(merchantIDs, merchantWeights, random.Float64())
			merchantID := merchantIDs[merchantOffset]
			merchantIDs = append(merchantIDs[:merchantOffset], merchantIDs[merchantOffset+1:]...)
			channelIndexes := make([]int, 0)
			for index := range remaining {
				if remaining[index].MerchantID == merchantID {
					channelIndexes = append(channelIndexes, index)
				}
			}
			channelWeights := channelCandidateWeights(remaining, channelIndexes, priority)
			selectedOffset := weightedFloatIndex(channelWeights, random.Float64())
			selectedIndex := channelIndexes[selectedOffset]
			ordered = append(ordered, remaining[selectedIndex])
			remaining = append(remaining[:selectedIndex], remaining[selectedIndex+1:]...)
		}
	}
	return ordered
}

func channelCandidateWeights(candidates []RouteCandidate, indexes []int, priority string) []float64 {
	rankedSuccess, rankedLatency, rankedCache := make([]rankedMetric, 0, len(indexes)), make([]rankedMetric, 0, len(indexes)), make([]rankedMetric, 0, len(indexes))
	for _, index := range indexes {
		candidate := candidates[index]
		id := int64(index + 1)
		rankedSuccess = append(rankedSuccess, rankedMetric{id: id, value: float64(candidate.AvailabilityPPM), known: candidate.MetricRequestCount > 0})
		rankedLatency = append(rankedLatency, rankedMetric{id: id, value: float64(candidate.TTFTP95MS), known: candidate.MetricRequestCount > 0 && candidate.TTFTP95MS > 0})
		rankedCache = append(rankedCache, rankedMetric{id: id, value: float64(candidate.CacheHitPPM), known: candidate.CacheSampleInputTokens >= cacheWeightMinimumInputTokens})
	}
	successScores, latencyScores, cacheScores := rankMetricScores(rankedSuccess, true), rankMetricScores(rankedLatency, false), rankMetricScores(rankedCache, true)
	weights := make([]float64, len(indexes))
	for offset, index := range indexes {
		id := int64(index + 1)
		success, latency, cache := successScores[id], latencyScores[id], cacheScores[id]
		score := (success + latency + cache) / 3
		switch priority {
		case "success":
			score = success*0.8 + latency*0.1 + cache*0.1
		case "latency":
			score = latency*0.8 + success*0.1 + cache*0.1
		case "cache":
			score = cache*0.8 + success*0.1 + latency*0.1
		}
		weights[offset] = 1 + 3*score
	}
	return weights
}

func weightedIDIndex(ids []int64, weights map[int64]float64, random float64) int {
	values := make([]float64, len(ids))
	for index, id := range ids {
		values[index] = weights[id]
	}
	return weightedFloatIndex(values, random)
}

func weightedFloatIndex(weights []float64, random float64) int {
	total := 0.0
	for _, weight := range weights {
		total += weight
	}
	point := random * total
	for index, weight := range weights {
		if point < weight {
			return index
		}
		point -= weight
	}
	return len(weights) - 1
}

func nextProbeState(probe HealthProbe, class healthFailureClass, now time.Time, retryAfter time.Duration) HealthProbe {
	result := probe
	switch class {
	case healthFailureHard:
		result.State = "disabled"
		result.NextProbeAt = time.Time{}
	case healthFailureNone:
		result.ConsecutiveFailures = 0
		result.CooldownLevel = 0
		result.State = "healthy"
		result.NextProbeAt = time.Time{}
	case healthFailureTemporary:
		result.ConsecutiveFailures++
		if probe.State == "cooling" || probe.State == "recovering" || result.ConsecutiveFailures >= 3 {
			result.State = "cooling"
			result.CooldownLevel++
			cooldown := cooldownForLevel(result.CooldownLevel)
			if retryAfter > cooldown {
				cooldown = retryAfter
			}
			result.NextProbeAt = now.Add(cooldown)
		} else {
			result.State = "degraded"
			result.NextProbeAt = time.Time{}
		}
	case healthFailureIgnored:
		if probe.State == "cooling" || probe.State == "recovering" {
			result.State = "cooling"
			result.CooldownLevel++
			result.NextProbeAt = now.Add(cooldownForLevel(result.CooldownLevel))
		} else {
			result.NextProbeAt = time.Time{}
		}
	}
	return result
}

func cooldownForLevel(level int) time.Duration {
	switch level {
	case 1:
		return time.Minute
	case 2:
		return 5 * time.Minute
	case 3:
		return 15 * time.Minute
	case 4:
		return 30 * time.Minute
	case 5:
		return 60 * time.Minute
	default:
		return 120 * time.Minute
	}
}
