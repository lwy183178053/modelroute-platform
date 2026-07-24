package httpapi

import "time"

const modelHealthCheckInterval = 3 * time.Minute

type ModelHealthState struct {
	State                      string
	ConsecutiveRequestFailures int
	AutomaticCheckFailures     int
	NextCheckAt                time.Time
}

func nextModelHealthAfterRequest(current ModelHealthState, class healthFailureClass, priceKind string, _ int, now time.Time) ModelHealthState {
	result := current
	switch class {
	case healthFailureNone:
		result.ConsecutiveRequestFailures = 0
		result.State = "healthy"
		result.NextCheckAt = time.Time{}
	case healthFailureHard:
		result.State = "error"
		result.NextCheckAt = time.Time{}
	case healthFailureTemporary:
		result.ConsecutiveRequestFailures++
		if result.ConsecutiveRequestFailures < 3 {
			return result
		}
		if priceKind == "image" {
			result.State = "error"
			result.NextCheckAt = time.Time{}
		} else {
			result.State = "cooling"
			result.AutomaticCheckFailures = 0
			result.NextCheckAt = now.Add(modelHealthCheckInterval)
		}
	}
	return result
}

func nextModelHealthAfterAutomaticCheck(current ModelHealthState, class healthFailureClass, now time.Time) ModelHealthState {
	if class == healthFailureNone {
		return ModelHealthState{State: "healthy"}
	}
	result := current
	result.AutomaticCheckFailures++
	if result.AutomaticCheckFailures >= 10 {
		result.AutomaticCheckFailures = 10
		result.State = "error"
		result.NextCheckAt = time.Time{}
		return result
	}
	result.State = "cooling"
	result.NextCheckAt = now.Add(modelHealthCheckInterval)
	return result
}

func nextModelHealthAfterManualCheck(current ModelHealthState, class healthFailureClass, _ time.Time) ModelHealthState {
	if class == healthFailureNone {
		return ModelHealthState{State: "healthy"}
	}
	current.State = "error"
	current.NextCheckAt = time.Time{}
	return current
}
