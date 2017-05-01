package collectors

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"github.com/starkandwayne/shield/api"
)

type TargetsCollector struct {
	namespace                              string
	environment                            string
	backendName                            string
	targetsTotalMetric                     *prometheus.GaugeVec
	targetsScrapesTotalMetric              prometheus.Counter
	targetsScrapeErrorsTotalMetric         prometheus.Counter
	lastTargetsScrapeErrorMetric           prometheus.Gauge
	lastTargetsScrapeTimestampMetric       prometheus.Gauge
	lastTargetsScrapeDurationSecondsMetric prometheus.Gauge
}

func NewTargetsCollector(
	namespace string,
	environment string,
	backendName string,
) *TargetsCollector {
	targetsTotalMetric := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "targets",
			Name:        "total",
			Help:        "Labeled total number of Shield Targets.",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
		[]string{"target_plugin"},
	)

	targetsScrapesTotalMetric := prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace:   namespace,
			Subsystem:   "targets",
			Name:        "scrapes_total",
			Help:        "Total number of scrapes for Shield Targets.",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
	)

	targetsScrapeErrorsTotalMetric := prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace:   namespace,
			Subsystem:   "targets",
			Name:        "scrape_errorstotal",
			Help:        "Total number of scrape errors of Shield Targets.",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
	)

	lastTargetsScrapeErrorMetric := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "",
			Name:        "last_targets_scrape_error",
			Help:        "Whether the last scrape of Target metrics from Shield resulted in an error (1 for error, 0 for success).",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
	)

	lastTargetsScrapeTimestampMetric := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "",
			Name:        "last_targets_scrape_timestamp",
			Help:        "Number of seconds since 1970 since last scrape of Target metrics from Shield.",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
	)

	lastTargetsScrapeDurationSecondsMetric := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "",
			Name:        "last_targets_scrape_duration_seconds",
			Help:        "Duration of the last scrape of Target metrics from Shield.",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
	)

	return &TargetsCollector{
		namespace:                              namespace,
		environment:                            environment,
		backendName:                            backendName,
		targetsTotalMetric:                     targetsTotalMetric,
		targetsScrapesTotalMetric:              targetsScrapesTotalMetric,
		targetsScrapeErrorsTotalMetric:         targetsScrapeErrorsTotalMetric,
		lastTargetsScrapeErrorMetric:           lastTargetsScrapeErrorMetric,
		lastTargetsScrapeTimestampMetric:       lastTargetsScrapeTimestampMetric,
		lastTargetsScrapeDurationSecondsMetric: lastTargetsScrapeDurationSecondsMetric,
	}
}

func (c TargetsCollector) Collect(ch chan<- prometheus.Metric) {
	var begun = time.Now()

	errorMetric := float64(0)
	if err := c.reportTargetsMetrics(ch); err != nil {
		errorMetric = float64(1)
		c.targetsScrapeErrorsTotalMetric.Inc()
	}
	c.targetsScrapeErrorsTotalMetric.Collect(ch)

	c.targetsScrapesTotalMetric.Inc()
	c.targetsScrapesTotalMetric.Collect(ch)

	c.lastTargetsScrapeErrorMetric.Set(errorMetric)
	c.lastTargetsScrapeErrorMetric.Collect(ch)

	c.lastTargetsScrapeTimestampMetric.Set(float64(time.Now().Unix()))
	c.lastTargetsScrapeTimestampMetric.Collect(ch)

	c.lastTargetsScrapeDurationSecondsMetric.Set(time.Since(begun).Seconds())
	c.lastTargetsScrapeDurationSecondsMetric.Collect(ch)
}

func (c TargetsCollector) Describe(ch chan<- *prometheus.Desc) {
	c.targetsTotalMetric.Describe(ch)
	c.targetsScrapesTotalMetric.Describe(ch)
	c.targetsScrapeErrorsTotalMetric.Describe(ch)
	c.lastTargetsScrapeErrorMetric.Describe(ch)
	c.lastTargetsScrapeTimestampMetric.Describe(ch)
	c.lastTargetsScrapeDurationSecondsMetric.Describe(ch)
}

func (c TargetsCollector) reportTargetsMetrics(ch chan<- prometheus.Metric) error {
	c.targetsTotalMetric.Reset()

	targets, err := api.GetTargets(api.TargetFilter{})
	if err != nil {
		log.Errorf("Error while listing targets: %v", err)
		return err
	}

	for _, target := range targets {
		c.targetsTotalMetric.WithLabelValues(target.Plugin).Inc()
	}

	c.targetsTotalMetric.Collect(ch)

	return nil
}
