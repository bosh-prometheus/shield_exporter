package collectors

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"github.com/starkandwayne/shield/api"
)

type SchedulesCollector struct {
	namespace                                string
	environment                              string
	backendName                              string
	schedulesTotalMetric                     prometheus.Gauge
	schedulesScrapesTotalMetric              prometheus.Counter
	schedulesScrapeErrorsTotalMetric         prometheus.Counter
	lastSchedulesScrapeErrorMetric           prometheus.Gauge
	lastSchedulesScrapeTimestampMetric       prometheus.Gauge
	lastSchedulesScrapeDurationSecondsMetric prometheus.Gauge
}

func NewSchedulesCollector(
	namespace string,
	environment string,
	backendName string,
) *SchedulesCollector {
	schedulesTotalMetric := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "schedules",
			Name:        "total",
			Help:        "Total number of Shield Schedules.",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
	)

	schedulesScrapesTotalMetric := prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace:   namespace,
			Subsystem:   "schedules",
			Name:        "scrapes_total",
			Help:        "Total number of scrapes for Shield Schedules.",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
	)

	schedulesScrapeErrorsTotalMetric := prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace:   namespace,
			Subsystem:   "schedules",
			Name:        "scrape_errors_total",
			Help:        "Total number of scrape errors of Shield Schedules.",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
	)

	lastSchedulesScrapeErrorMetric := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "",
			Name:        "last_schedules_scrape_error",
			Help:        "Whether the last scrape of Schedule metrics from Shield resulted in an error (1 for error, 0 for success).",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
	)

	lastSchedulesScrapeTimestampMetric := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "",
			Name:        "last_schedules_scrape_timestamp",
			Help:        "Number of seconds since 1970 since last scrape of Schedule metrics from Shield.",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
	)

	lastSchedulesScrapeDurationSecondsMetric := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "",
			Name:        "last_schedules_scrape_duration_seconds",
			Help:        "Duration of the last scrape of Schedule metrics from Shield.",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
	)

	return &SchedulesCollector{
		namespace:                                namespace,
		environment:                              environment,
		backendName:                              backendName,
		schedulesTotalMetric:                     schedulesTotalMetric,
		schedulesScrapesTotalMetric:              schedulesScrapesTotalMetric,
		schedulesScrapeErrorsTotalMetric:         schedulesScrapeErrorsTotalMetric,
		lastSchedulesScrapeErrorMetric:           lastSchedulesScrapeErrorMetric,
		lastSchedulesScrapeTimestampMetric:       lastSchedulesScrapeTimestampMetric,
		lastSchedulesScrapeDurationSecondsMetric: lastSchedulesScrapeDurationSecondsMetric,
	}
}

func (c SchedulesCollector) Collect(ch chan<- prometheus.Metric) {
	var begun = time.Now()

	errorMetric := float64(0)
	if err := c.reportSchedulesMetrics(ch); err != nil {
		errorMetric = float64(1)
		c.schedulesScrapeErrorsTotalMetric.Inc()
	}
	c.schedulesScrapeErrorsTotalMetric.Collect(ch)

	c.schedulesScrapesTotalMetric.Inc()
	c.schedulesScrapesTotalMetric.Collect(ch)

	c.lastSchedulesScrapeErrorMetric.Set(errorMetric)
	c.lastSchedulesScrapeErrorMetric.Collect(ch)

	c.lastSchedulesScrapeTimestampMetric.Set(float64(time.Now().Unix()))
	c.lastSchedulesScrapeTimestampMetric.Collect(ch)

	c.lastSchedulesScrapeDurationSecondsMetric.Set(time.Since(begun).Seconds())
	c.lastSchedulesScrapeDurationSecondsMetric.Collect(ch)
}

func (c SchedulesCollector) Describe(ch chan<- *prometheus.Desc) {
	c.schedulesTotalMetric.Describe(ch)
	c.schedulesScrapesTotalMetric.Describe(ch)
	c.schedulesScrapeErrorsTotalMetric.Describe(ch)
	c.lastSchedulesScrapeErrorMetric.Describe(ch)
	c.lastSchedulesScrapeTimestampMetric.Describe(ch)
	c.lastSchedulesScrapeDurationSecondsMetric.Describe(ch)
}

func (c SchedulesCollector) reportSchedulesMetrics(ch chan<- prometheus.Metric) error {
	schedules, err := api.GetSchedules(api.ScheduleFilter{})
	if err != nil {
		log.Errorf("Error while listing schedules: %v", err)
		return err
	}

	c.schedulesTotalMetric.Set(float64(len(schedules)))
	c.schedulesTotalMetric.Collect(ch)

	return nil
}
