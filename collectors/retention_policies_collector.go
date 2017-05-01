package collectors

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"github.com/starkandwayne/shield/api"
)

type RetentionPoliciesCollector struct {
	namespace                                        string
	environment                                      string
	backendName                                      string
	retentionPoliciesTotalMetric                     prometheus.Gauge
	retentionPoliciesScrapesTotalMetric              prometheus.Counter
	retentionPoliciesScrapeErrorsTotalMetric         prometheus.Counter
	lastRetentionPoliciesScrapeErrorMetric           prometheus.Gauge
	lastRetentionPoliciesScrapeTimestampMetric       prometheus.Gauge
	lastRetentionPoliciesScrapeDurationSecondsMetric prometheus.Gauge
}

func NewRetentionPoliciesCollector(
	namespace string,
	environment string,
	backendName string,
) *RetentionPoliciesCollector {
	retentionPoliciesTotalMetric := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "retention_policies",
			Name:        "total",
			Help:        "Total number of Shield Retention Policies.",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
	)

	retentionPoliciesScrapesTotalMetric := prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace:   namespace,
			Subsystem:   "retention_policies",
			Name:        "scrapes_total",
			Help:        "Total number of scrapes for Shield Retention Policies.",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
	)

	retentionPoliciesScrapeErrorsTotalMetric := prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace:   namespace,
			Subsystem:   "retention_policies",
			Name:        "scrape_errors_total",
			Help:        "Total number of scrape errors of Shield Retention Policies.",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
	)

	lastRetentionPoliciesScrapeErrorMetric := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "",
			Name:        "last_retention_policies_scrape_error",
			Help:        "Whether the last scrape of Retention Policies metrics from Shield resulted in an error (1 for error, 0 for success).",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
	)

	lastRetentionPoliciesScrapeTimestampMetric := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "",
			Name:        "last_retention_policies_scrape_timestamp",
			Help:        "Number of seconds since 1970 since last scrape of Retention Policies metrics from Shield.",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
	)

	lastRetentionPoliciesScrapeDurationSecondsMetric := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "",
			Name:        "last_retention_policies_scrape_duration_seconds",
			Help:        "Duration of the last scrape of Retention Policies metrics from Shield.",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
	)

	return &RetentionPoliciesCollector{
		namespace:                                        namespace,
		environment:                                      environment,
		backendName:                                      backendName,
		retentionPoliciesTotalMetric:                     retentionPoliciesTotalMetric,
		retentionPoliciesScrapesTotalMetric:              retentionPoliciesScrapesTotalMetric,
		retentionPoliciesScrapeErrorsTotalMetric:         retentionPoliciesScrapeErrorsTotalMetric,
		lastRetentionPoliciesScrapeErrorMetric:           lastRetentionPoliciesScrapeErrorMetric,
		lastRetentionPoliciesScrapeTimestampMetric:       lastRetentionPoliciesScrapeTimestampMetric,
		lastRetentionPoliciesScrapeDurationSecondsMetric: lastRetentionPoliciesScrapeDurationSecondsMetric,
	}
}

func (c RetentionPoliciesCollector) Collect(ch chan<- prometheus.Metric) {
	var begun = time.Now()

	errorMetric := float64(0)
	if err := c.reportRetentionPoliciesMetrics(ch); err != nil {
		errorMetric = float64(1)
		c.retentionPoliciesScrapeErrorsTotalMetric.Inc()
	}
	c.retentionPoliciesScrapeErrorsTotalMetric.Collect(ch)

	c.retentionPoliciesScrapesTotalMetric.Inc()
	c.retentionPoliciesScrapesTotalMetric.Collect(ch)

	c.lastRetentionPoliciesScrapeErrorMetric.Set(errorMetric)
	c.lastRetentionPoliciesScrapeErrorMetric.Collect(ch)

	c.lastRetentionPoliciesScrapeTimestampMetric.Set(float64(time.Now().Unix()))
	c.lastRetentionPoliciesScrapeTimestampMetric.Collect(ch)

	c.lastRetentionPoliciesScrapeDurationSecondsMetric.Set(time.Since(begun).Seconds())
	c.lastRetentionPoliciesScrapeDurationSecondsMetric.Collect(ch)
}

func (c RetentionPoliciesCollector) Describe(ch chan<- *prometheus.Desc) {
	c.retentionPoliciesTotalMetric.Describe(ch)
	c.retentionPoliciesScrapesTotalMetric.Describe(ch)
	c.retentionPoliciesScrapeErrorsTotalMetric.Describe(ch)
	c.lastRetentionPoliciesScrapeErrorMetric.Describe(ch)
	c.lastRetentionPoliciesScrapeTimestampMetric.Describe(ch)
	c.lastRetentionPoliciesScrapeDurationSecondsMetric.Describe(ch)
}

func (c RetentionPoliciesCollector) reportRetentionPoliciesMetrics(ch chan<- prometheus.Metric) error {
	retentionPolicies, err := api.GetRetentionPolicies(api.RetentionPolicyFilter{})
	if err != nil {
		log.Errorf("Error while listing retention policies: %v", err)
		return err
	}

	c.retentionPoliciesTotalMetric.Set(float64(len(retentionPolicies)))
	c.retentionPoliciesTotalMetric.Collect(ch)

	return nil
}
