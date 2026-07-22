package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

type HealthProbe struct {
	OfferID             int64
	MerchantChannelID   int64
	CanonicalModelID    int64
	NewAPIChannelID     int
	InternalAlias       string
	Protocol            string
	State               string
	ConsecutiveFailures int
	CooldownLevel       int
	RecoverySuccesses   int
	HealthySince        time.Time
	NextProbeAt         time.Time
}

type RouteCandidate struct {
	OfferID           int64  `json:"offer_id"`
	MerchantID        int64  `json:"merchant_id"`
	MerchantChannelID int64  `json:"merchant_channel_id"`
	NewAPIChannelID   int    `json:"newapi_channel_id"`
	CanonicalModelID  int64  `json:"canonical_model_id"`
	InternalAlias     string `json:"-"`
	PublicModel       string `json:"model"`
	Protocol          string `json:"protocol"`
	MultiplierBPS     int    `json:"multiplier_bps"`
	HealthState       string `json:"health_state"`
	AvailabilityPPM   int    `json:"availability_ppm"`
	TTFTP95MS         int    `json:"ttft_p95_ms"`
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

func applyBasisPoints(value int64, basisPoints int) int64 {
	return value * int64(basisPoints) / 10_000
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

func sortCandidates(candidates []RouteCandidate, priority string) {
	sort.SliceStable(candidates, func(i, j int) bool {
		left, right := candidates[i], candidates[j]
		if priority == "price" && left.MultiplierBPS != right.MultiplierBPS {
			return left.MultiplierBPS < right.MultiplierBPS
		}
		if healthRank(left.HealthState) != healthRank(right.HealthState) {
			return healthRank(left.HealthState) < healthRank(right.HealthState)
		}
		if left.AvailabilityPPM != right.AvailabilityPPM {
			return left.AvailabilityPPM > right.AvailabilityPPM
		}
		if left.TTFTP95MS != right.TTFTP95MS {
			return left.TTFTP95MS < right.TTFTP95MS
		}
		if priority != "price" && left.MultiplierBPS != right.MultiplierBPS {
			return left.MultiplierBPS < right.MultiplierBPS
		}
		return left.OfferID < right.OfferID
	})
}

func healthRank(state string) int {
	if state == "healthy" {
		return 0
	}
	return 1
}

func nextProbeState(probe HealthProbe, class healthFailureClass, now time.Time, retryAfter time.Duration) HealthProbe {
	result := probe
	if !result.HealthySince.IsZero() && now.Sub(result.HealthySince) >= 24*time.Hour {
		result.CooldownLevel = 0
	}
	switch class {
	case healthFailureHard:
		result.State = "disabled"
		result.NextProbeAt = time.Time{}
		result.RecoverySuccesses = 0
	case healthFailureNone:
		result.ConsecutiveFailures = 0
		if probe.State == "cooling" || probe.State == "recovering" {
			result.RecoverySuccesses++
			if result.RecoverySuccesses < 2 {
				result.State = "recovering"
				result.NextProbeAt = now.Add(10 * time.Second)
				return result
			}
		}
		result.State = "healthy"
		result.RecoverySuccesses = 0
		result.HealthySince = now
		result.NextProbeAt = now.Add(5 * time.Minute)
	case healthFailureTemporary:
		result.ConsecutiveFailures++
		result.RecoverySuccesses = 0
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
			result.NextProbeAt = now.Add(5 * time.Minute)
		}
	case healthFailureIgnored:
		result.NextProbeAt = now.Add(5 * time.Minute)
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
	default:
		return 60 * time.Minute
	}
}
