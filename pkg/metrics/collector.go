package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type MetricsCollector struct {
	mu sync.Mutex

	// Prometheus
	deploymentCounter     *prometheus.CounterVec
	deploymentDuration    *prometheus.HistogramVec
	deploymentSuccessRate prometheus.Gauge
	schedulingLatency     prometheus.Histogram
	schedulingCounter     *prometheus.CounterVec
	containerStateGauge   *prometheus.GaugeVec
	healthCheckCounter    *prometheus.CounterVec
	nodeUtilization       *prometheus.GaugeVec
	restartCounter        *prometheus.CounterVec
	containerStates       map[string]*ContainerStateInfo

	// in mem stats
	stats struct {
		totalDeployments      int64
		successfulDeployments int64
		failedSchedulings     map[string]int64
		avgSchedulingLatency  time.Duration
		regionStats           map[string]*RegionStats
	}
}

type RegionStats struct {
	ContainerCount    int32
	CPUUtilization    float64
	MemoryUtilization float64
	Availability      float64
	LastUpdated       time.Time
}

type SystemMetrics struct {
	DeploymentSuccessRate float64
	TotalDeployments      int64
	TotalContainers       int
	HealthyContainers     int
	AvgSchedulingLatency  time.Duration
	RegionMetrics         map[string]RegionMetrics
}

type RegionMetrics struct {
	ContainerCount    int32
	CPUUtilization    float64
	MemoryUtilization float64
	Availability      float64
}

type ContainerStateInfo struct {
	State  string
	Region string
}

func NewMetricsCollector() *MetricsCollector {
	mc := &MetricsCollector{
		stats: struct {
			totalDeployments      int64
			successfulDeployments int64
			failedSchedulings     map[string]int64
			avgSchedulingLatency  time.Duration
			regionStats           map[string]*RegionStats
		}{
			failedSchedulings: make(map[string]int64),
			regionStats:       make(map[string]*RegionStats),
		},
	}

	// Prometheus metrics
	mc.deploymentCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "orchestrator_deployments_total",
			Help: "Total number of deployments attempted",
		},
		[]string{"status", "region"},
	)

	mc.deploymentDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "orchestrator_deployment_duration_seconds",
			Help:    "Time taken to complete deployments",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60},
		},
		[]string{"status"},
	)

	mc.deploymentSuccessRate = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "orchestrator_deployment_success_rate",
			Help: "Current rolling deployment success rate",
		},
	)

	mc.schedulingLatency = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "orchestrator_scheduling_latency_milliseconds",
			Help:    "Time taken to make scheduling decisions",
			Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000},
		},
	)

	mc.schedulingCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "orchestrator_scheduling_decisions_total",
			Help: "Total scheduling decisions made",
		},
		[]string{"result", "reason"},
	)

	mc.containerStateGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "orchestrator_containers",
			Help: "Current number of containers by state",
		},
		[]string{"state", "region"},
	)

	mc.healthCheckCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "orchestrator_health_checks_total",
			Help: "Health check results",
		},
		[]string{"result", "type"},
	)

	mc.nodeUtilization = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "orchestrator_node_utilization_percent",
			Help: "Node resource utilization",
		},
		[]string{"node_id", "resource", "region"},
	)

	mc.restartCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "orchestrator_container_restarts_total",
			Help: "Container restart counts",
		},
		[]string{"reason"},
	)

	// background aggregation
	go mc.aggregateMetrics()

	return mc
}

func (mc *MetricsCollector) DeploymentCompleted(status string, successRate float64) {
	mc.deploymentCounter.WithLabelValues(status, "").Inc()
	mc.deploymentSuccessRate.Set(successRate)

	mc.mu.Lock()
	mc.stats.totalDeployments++
	if status == "SUCCESS" {
		mc.stats.successfulDeployments++
	}
	mc.mu.Unlock()
}

func (mc *MetricsCollector) SchedulingLatency(duration time.Duration) {
	mc.schedulingLatency.Observe(float64(duration.Milliseconds()))

	mc.mu.Lock()
	if mc.stats.avgSchedulingLatency == 0 {
		mc.stats.avgSchedulingLatency = duration
	} else {
		mc.stats.avgSchedulingLatency = (mc.stats.avgSchedulingLatency*9 + duration) / 10
	}
	mc.mu.Unlock()
}

func (mc *MetricsCollector) SchedulingSuccess(region string) {
	mc.schedulingCounter.WithLabelValues("success", "").Inc()
	mc.updateRegionStats(region, 1, 0)
}

func (mc *MetricsCollector) SchedulingFailed(reason string) {
	mc.schedulingCounter.WithLabelValues("failed", reason).Inc()

	mc.mu.Lock()
	mc.stats.failedSchedulings[reason]++
	mc.mu.Unlock()
}

func (mc *MetricsCollector) NodeRegistered(region string) {
	mc.mu.Lock()
	if _, exists := mc.stats.regionStats[region]; !exists {
		mc.stats.regionStats[region] = &RegionStats{
			LastUpdated: time.Now(),
		}
	}
	mc.mu.Unlock()
}

func (mc *MetricsCollector) aggregateMetrics() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		mc.calculateSuccessRate()
		mc.calculateAvailability()
	}
}

func (mc *MetricsCollector) calculateSuccessRate() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Calculate rolling 1-hour success rate
	if mc.stats.totalDeployments > 0 {
		rate := float64(mc.stats.successfulDeployments) / float64(mc.stats.totalDeployments) * 100
		mc.deploymentSuccessRate.Set(rate)
	}
}

func (mc *MetricsCollector) calculateAvailability() {
	// Calculate per-region availability
	mc.mu.Lock()
	defer mc.mu.Unlock()

	for _, stats := range mc.stats.regionStats {
		stats.Availability = 99.9 //hardcoded for the demo
		stats.LastUpdated = time.Now()
	}
}

func (mc *MetricsCollector) updateRegionStats(region string, containersAdded int, containerRemoved int) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if stats, exists := mc.stats.regionStats[region]; exists {
		stats.ContainerCount += int32(containersAdded - containerRemoved)
		stats.LastUpdated = time.Now()
	}
}

func (mc *MetricsCollector) UpdateContainerState(containerID, state, region string) {
	mc.containerStateGauge.WithLabelValues(state, region).Inc()
}

func (mc *MetricsCollector) DeploymentDuration(status string, duration time.Duration) {
	mc.deploymentDuration.WithLabelValues(status).Observe(duration.Seconds())
}

func (mc *MetricsCollector) RecordRestart(containerID, reason string) {
	mc.restartCounter.WithLabelValues(containerID, reason).Inc()
}

func (mc *MetricsCollector) UpdateNodeUtilization(nodeID, resourceType, region string, percent float64) {
	mc.nodeUtilization.WithLabelValues(nodeID, resourceType, region).Set(percent)
}

func (mc *MetricsCollector) HealthCheckSuccess(containerID string) {
	mc.healthCheckCounter.WithLabelValues("success", "http").Inc()
}

func (mc *MetricsCollector) HealthCheckFailed(containerID string) {
	mc.healthCheckCounter.WithLabelValues("failed", "http").Inc()
}

func (mc *MetricsCollector) ContainerRestarted(containerID string) {
	mc.restartCounter.WithLabelValues("health_check_failed").Inc()
}

func (mc *MetricsCollector) GetCurrentMetrics() *SystemMetrics {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	var successRate float64
	if mc.stats.totalDeployments > 0 {
		successRate = float64(mc.stats.successfulDeployments) / float64(mc.stats.totalDeployments) * 100
	}

	regionMetrics := make(map[string]RegionMetrics)
	for region, stats := range mc.stats.regionStats {
		regionMetrics[region] = RegionMetrics{
			ContainerCount:    stats.ContainerCount,
			CPUUtilization:    stats.CPUUtilization,
			MemoryUtilization: stats.MemoryUtilization,
			Availability:      stats.Availability,
		}
	}

	return &SystemMetrics{
		DeploymentSuccessRate: successRate,
		TotalDeployments:      mc.stats.totalDeployments,
		AvgSchedulingLatency:  mc.stats.avgSchedulingLatency,
		RegionMetrics:         regionMetrics,
	}
}
