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
				stats := c.getStats(name)
				state, semaphoreQueue, semaphoreCurrent, semaphoreMax, counts, durations := stats.GetAndReset()

				event1 := makeCommandStats(
					name,
					state,
					counts,
					durations,
					c.registry,
				)

				select {
				case <-c.halt:
					return
				case c.events <- event1:
				}

				event2 := makeThreadPoolStats(
					name,
					counts,
					stats.config.MaxConcurrency,
					semaphoreQueue,
					semaphoreCurrent,
					semaphoreMax,
				)

				select {
				case <-c.halt:
					return
				case c.events <- event2:
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

func (c *HystrixCollector) Server() http.Handler {
	server := sse.NewServer(c.events)
	server.Start()
	return server
}

func (c *HystrixCollector) ReportNew(name string, config overcurrent.BreakerConfig) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.breakers[name] = NewBreakerStats(config)
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

func (c *HystrixCollector) getStats(name string) *BreakerStats {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.breakers[name]
}

//
//

func makeCommandStats(
	name string,
	state overcurrent.CircuitState,
	counts map[overcurrent.EventType]int,
	durations map[overcurrent.EventType][]time.Duration,
	registry overcurrent.Registry,
) map[string]interface{} {
	var (
		runDurations    = sortDurations(durations[overcurrent.EventTypeRunDuration])
		totalDurations  = sortDurations(durations[overcurrent.EventTypeTotalDuration])
		numErrors       = counts[overcurrent.EventTypeFailure]
		numRequests     = counts[overcurrent.EventTypeAttempt]
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
		"rollingCountSuccess":                   counts[overcurrent.EventTypeSuccess],
		"rollingCountFailure":                   counts[overcurrent.EventTypeError],
		"rollingCountBadRequest":                counts[overcurrent.EventTypeBadRequest],
		"rollingCountShortCircuited":            counts[overcurrent.EventTypeShortCircuit],
		"rollingCountTimeout":                   counts[overcurrent.EventTypeTimeout],
		"rollingCountSemaphoreRejected":         counts[overcurrent.EventTypeRejection],
		"rollingCountFallbackSuccess":           counts[overcurrent.EventTypeFallbackSuccess],
		"rollingCountFallbackFailure":           counts[overcurrent.EventTypeFallbackFailure],
		"latencyExecute":                        makeLatencies(runDurations),
		"latencyTotal":                          makeLatencies(totalDurations),
		"latencyExecute_mean":                   int(mean(runDurations) / time.Millisecond),
		"latencyTotal_mean":                     int(mean(totalDurations) / time.Millisecond),
		"isCircuitBreakerOpen":                  state != overcurrent.StateClosed,
		"propertyValue_circuitBreakerForceOpen": state == overcurrent.StateHardOpen,
	}

	for k, v := range constantCommandProperties {
		properties[k] = v
	}

	return properties
}

func makeThreadPoolStats(
	name string,
	counts map[overcurrent.EventType]int,
	semaphoreCapacity,
	semaphoreQueue,
	semaphoreCurrent,
	semaphoreMax int,
) map[string]interface{} {
	properties := map[string]interface{}{
		"type":                        "HystrixThreadPool",
		"name":                        name,
		"currentCorePoolSize":         semaphoreCapacity,
		"currentLargestPoolSize":      semaphoreCapacity,
		"currentMaximumPoolSize":      semaphoreCapacity,
		"currentPoolSize":             semaphoreCapacity,
		"currentActiveCount":          semaphoreCurrent,
		"rollingMaxActiveThreads":     semaphoreMax,
		"rollingCountThreadsExecuted": counts[overcurrent.EventTypeSemaphoreAcquired],
		"currentQueueSize":            semaphoreQueue,
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
	"propertyValue_metricsRollingStatisticalWindowInMilliseconds":    1000,
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
	"propertyValue_metricsRollingStatisticalWindowInMilliseconds": 1000,
	"propertyValue_queueSizeRejectionThreshold":                   "NaN",
	"reportingHosts":                                              1,
}
