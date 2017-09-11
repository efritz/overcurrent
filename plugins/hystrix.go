package plugins

import (
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/efritz/overcurrent"
	"github.com/efritz/sse"
)

type HystrixCollector struct {
	registry overcurrent.Registry
	breakers map[string]*BreakerStats
	events   chan interface{}
	halt     chan struct{}
	mutex    *sync.RWMutex
}

func NewHystrixCollector(registry overcurrent.Registry) *HystrixCollector {
	return &HystrixCollector{
		registry: registry,
		breakers: map[string]*BreakerStats{},
		events:   make(chan interface{}),
		halt:     make(chan struct{}),
		mutex:    &sync.RWMutex{},
	}
}

func (c *HystrixCollector) Start() {
	go func() {
		defer close(c.events)

		for {
			for _, name := range c.getNames() {
				stats := c.getStats(name).Freeze()

				if !c.send(c.makeCommandStats(name, stats)) || !c.send(c.makeThreadPoolStats(name, stats)) {
					return
				}
			}

			select {
			case <-c.halt:
				return
			case <-time.After(time.Second):
			}
		}
	}()
}

func (c *HystrixCollector) Stop() {
	close(c.halt)
}

func (c *HystrixCollector) Handler() http.Handler {
	server := sse.NewServer(c.events)
	go server.Start()
	return server
}

func (c *HystrixCollector) ReportNew(name string, config overcurrent.BreakerConfig) {
	c.mutex.Lock()
	c.breakers[name] = NewBreakerStats(config)
	c.mutex.Unlock()
}

func (c *HystrixCollector) ReportCount(name string, eventType overcurrent.EventType) {
	c.getStats(name).Increment(eventType)
}

func (c *HystrixCollector) ReportDuration(name string, eventType overcurrent.EventType, duration time.Duration) {
	c.getStats(name).AddDuration(eventType, duration)
}

func (c *HystrixCollector) ReportState(name string, state overcurrent.CircuitState) {
	c.getStats(name).SetState(state)
}

func (c *HystrixCollector) getNames() []string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	names := []string{}
	for name := range c.breakers {
		names = append(names, name)
	}

	return names
}

func (c *HystrixCollector) send(event map[string]interface{}) bool {
	select {
	case <-c.halt:
		return false
	case c.events <- event:
		return true
	}
}

func (c *HystrixCollector) getStats(name string) *BreakerStats {
	c.mutex.RLock()
	stats := c.breakers[name]
	c.mutex.RUnlock()

	return stats
}

func (c *HystrixCollector) makeCommandStats(name string, stats *FrozenBreakerStats) map[string]interface{} {
	var (
		runDurations    = stats.durations[overcurrent.EventTypeRunDuration]
		totalDurations  = stats.durations[overcurrent.EventTypeTotalDuration]
		numErrors       = stats.counters[overcurrent.EventTypeFailure]
		numRequests     = stats.counters[overcurrent.EventTypeAttempt]
		errorPercentage = 0.0
	)

	if numRequests > 0 {
		errorPercentage = math.Min(1, (float64(numErrors)/float64(numRequests))) * 100
	}

	properties := map[string]interface{}{
		"type":                                  "HystrixCommand",
		"name":                                  name,
		"group":                                 name,
		"currentTime":                           time.Now().Unix(),
		"errorCount":                            numErrors,
		"requestCount":                          numRequests,
		"errorPercentage":                       errorPercentage,
		"rollingCountSuccess":                   stats.counters[overcurrent.EventTypeSuccess],
		"rollingCountFailure":                   stats.counters[overcurrent.EventTypeError],
		"rollingCountBadRequest":                stats.counters[overcurrent.EventTypeBadRequest],
		"rollingCountShortCircuited":            stats.counters[overcurrent.EventTypeShortCircuit],
		"rollingCountTimeout":                   stats.counters[overcurrent.EventTypeTimeout],
		"rollingCountSemaphoreRejected":         stats.counters[overcurrent.EventTypeRejection],
		"rollingCountFallbackSuccess":           stats.counters[overcurrent.EventTypeFallbackSuccess],
		"rollingCountFallbackFailure":           stats.counters[overcurrent.EventTypeFallbackFailure],
		"latencyExecute":                        makeLatencies(runDurations),
		"latencyTotal":                          makeLatencies(totalDurations),
		"latencyExecute_mean":                   int(mean(runDurations) / time.Millisecond),
		"latencyTotal_mean":                     int(mean(totalDurations) / time.Millisecond),
		"isCircuitBreakerOpen":                  stats.state != overcurrent.StateClosed,
		"propertyValue_circuitBreakerForceOpen": stats.state == overcurrent.StateHardOpen,
	}

	for k, v := range constantCommandProperties {
		properties[k] = v
	}

	return properties
}

func (c *HystrixCollector) makeThreadPoolStats(name string, stats *FrozenBreakerStats) map[string]interface{} {
	properties := map[string]interface{}{
		"type":                        "HystrixThreadPool",
		"name":                        name,
		"currentCorePoolSize":         stats.config.MaxConcurrency,
		"currentLargestPoolSize":      stats.config.MaxConcurrency,
		"currentMaximumPoolSize":      stats.config.MaxConcurrency,
		"currentPoolSize":             stats.config.MaxConcurrency,
		"currentActiveCount":          stats.currents[overcurrent.EventTypeSemaphoreAcquired],
		"rollingMaxActiveThreads":     stats.maximums[overcurrent.EventTypeSemaphoreAcquired],
		"rollingCountThreadsExecuted": stats.counters[overcurrent.EventTypeSemaphoreAcquired],
		"currentQueueSize":            stats.maximums[overcurrent.EventTypeSemaphoreQueued],
	}

	for k, v := range constantThreadPoolProperties {
		properties[k] = v
	}

	return properties
}

func makeLatencies(values []time.Duration) map[string]interface{} {
	return map[string]interface{}{
		"0":    int(percentile(values, 0.000) / time.Millisecond),
		"25":   int(percentile(values, 0.250) / time.Millisecond),
		"50":   int(percentile(values, 0.500) / time.Millisecond),
		"75":   int(percentile(values, 0.750) / time.Millisecond),
		"90":   int(percentile(values, 0.900) / time.Millisecond),
		"95":   int(percentile(values, 0.950) / time.Millisecond),
		"99":   int(percentile(values, 0.990) / time.Millisecond),
		"99.5": int(percentile(values, 0.995) / time.Millisecond),
		"100":  int(percentile(values, 1.000) / time.Millisecond),
	}
}

var constantCommandProperties = map[string]interface{}{
	"currentConcurrentExecutionCount":                                0,
	"propertyValue_circuitBreakerEnabled":                            true,
	"propertyValue_circuitBreakerErrorThresholdPercentage":           0,
	"propertyValue_circuitBreakerForceClosed":                        false,
	"propertyValue_circuitBreakerRequestVolumeThreshold":             0,
	"propertyValue_circuitBreakerSleepWindowInMilliseconds":          0,
	"propertyValue_executionIsolationSemaphoreMaxConcurrentRequests": 0,
	"propertyValue_executionIsolationStrategy":                       "SEMAPHORE",
	"propertyValue_executionIsolationThreadInterruptOnTimeout":       false,
	"propertyValue_executionIsolationThreadPoolKeyOverride":          "",
	"propertyValue_executionIsolationThreadTimeoutInMilliseconds":    "",
	"propertyValue_fallbackIsolationSemaphoreMaxConcurrentRequests":  0,
	"propertyValue_metricsRollingStatisticalWindowInMilliseconds":    10000,
	"propertyValue_requestCacheEnabled":                              false,
	"propertyValue_requestLogEnabled":                                false,
	"reportingHosts":                                                 1,
	"rollingCountCollapsedRequests":                                  0,
	"rollingCountExceptionsThrown":                                   0,
	"rollingCountFallbackRejection":                                  0,
	"rollingCountResponsesFromCache":                                 0,
	"rollingCountThreadPoolRejected":                                 0,
}

var constantThreadPoolProperties = map[string]interface{}{
	"currentCompletedTaskCount":                                   15,
	"currentTaskCount":                                            15,
	"propertyValue_metricsRollingStatisticalWindowInMilliseconds": 10000,
	"propertyValue_queueSizeRejectionThreshold":                   "NaN",
	"reportingHosts":                                              1,
}
