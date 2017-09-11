package hystrix

import (
	"sort"
	"time"

	"github.com/efritz/overcurrent"
)

func sortDurationMap(values map[overcurrent.EventType][]time.Duration) map[overcurrent.EventType][]time.Duration {
	for k, v := range values {
		values[k] = sortDurations(v)
	}

	return values
}

func sortDurations(values []time.Duration) []time.Duration {
	sort.Slice(values, func(i, j int) bool {
		return values[i] < values[j]
	})

	return values
}

func max(a, b int) int {
	if a < b {
		return b
	}

	return a
}

func cloneMap(values map[overcurrent.EventType]int) map[overcurrent.EventType]int {
	clone := map[overcurrent.EventType]int{}
	for k, v := range values {
		clone[k] = v
	}

	return clone
}

func mean(values []time.Duration) time.Duration {
	if len(values) == 0 {
		return 0
	}

	sum := time.Duration(0)
	for _, value := range values {
		sum += value
	}

	return time.Duration(float64(sum) / float64(len(values)))
}

func percentile(values []time.Duration, p float64) time.Duration {
	if len(values) == 0 {
		return 0
	}

	if p == 0 {
		return values[0]
	}

	index := int(round(p*float64(len(values)), 0.05))

	if index >= len(values) {
		index = len(values) - 1
	}

	return values[index]
}

func round(x, unit float64) float64 {
	if x > 0 {
		return float64(int64(x/unit+0.5)) * unit
	}

	return float64(int64(x/unit-0.5)) * unit
}
