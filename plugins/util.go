package plugins

import (
	"math"
	"sort"
	"time"
)

func sortDurations(values []time.Duration) []time.Duration {
	sort.Slice(values, func(i, j int) bool {
		return values[i] < values[j]
	})

	return values
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

	return values[int(math.Ceil(p*float64(len(values))))-1]
}
