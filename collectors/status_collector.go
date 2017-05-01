package collectors

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"github.com/starkandwayne/shield/api"
)

type InternalStatus struct {
	PendingTasks  []interface{} `json:"pending_tasks"`
	RunningTasks  []interface{} `json:"running_tasks"`
	ScheduleQueue []interface{} `json:"schedule_queue"`
	RunQueue      []interface{} `json:"run_queue"`
}

type StatusCollector struct {
	namespace                             string
	environment                           string
	backendName                           string
	pendingTasksTotalMetric               prometheus.Gauge
	runningTasksTotalMetric               prometheus.Gauge
	scheduleQueueTotalMetric              prometheus.Gauge
	runQueueTotalMetric                   prometheus.Gauge
	statusScrapesTotalMetric              prometheus.Counter
	statusScrapeErrorsTotalMetric         prometheus.Counter
	lastStatusScrapeErrorMetric           prometheus.Gauge
	lastStatusScrapeTimestampMetric       prometheus.Gauge
	lastStatusScrapeDurationSecondsMetric prometheus.Gauge
}

func NewStatusCollector(
	namespace string,
	environment string,
	backendName string,
) *StatusCollector {
	pendingTasksTotalMetric := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "status",
			Name:        "pending_tasks_total",
			Help:        "Total number of Shield pending Tasks.",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
	)

	runningTasksTotalMetric := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "status",
			Name:        "running_tasks_total",
			Help:        "Total number of Shield running Tasks.",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
	)

	scheduleQueueTotalMetric := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "status",
			Name:        "schedule_queue_total",
			Help:        "Total number of Shield Tasks in the supervisor scheduler queue.",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
	)

	runQueueTotalMetric := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "status",
			Name:        "run_queue_total",
			Help:        "Total number of Shield Tasks in the supervisor run queue.",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
	)

	statusScrapesTotalMetric := prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace:   namespace,
			Subsystem:   "status",
			Name:        "scrapes_total",
			Help:        "Total number of scrapes for Shield Status.",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
	)

	statusScrapeErrorsTotalMetric := prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace:   namespace,
			Subsystem:   "status",
			Name:        "scrape_errors_total",
			Help:        "Total number of scrape errors of Shield Status.",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
	)

	lastStatusScrapeErrorMetric := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "",
			Name:        "last_status_scrape_error",
			Help:        "Whether the last scrape of Status metrics from Shield resulted in an error (1 for error, 0 for success).",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
	)

	lastStatusScrapeTimestampMetric := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "",
			Name:        "last_status_scrape_timestamp",
			Help:        "Number of seconds since 1970 since last scrape of Status metrics from Shield.",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
	)

	lastStatusScrapeDurationSecondsMetric := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "",
			Name:        "last_status_scrape_duration_seconds",
			Help:        "Duration of the last scrape of Status metrics from Shield.",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
	)

	return &StatusCollector{
		namespace:                             namespace,
		environment:                           environment,
		backendName:                           backendName,
		pendingTasksTotalMetric:               pendingTasksTotalMetric,
		runningTasksTotalMetric:               runningTasksTotalMetric,
		scheduleQueueTotalMetric:              scheduleQueueTotalMetric,
		runQueueTotalMetric:                   runQueueTotalMetric,
		statusScrapesTotalMetric:              statusScrapesTotalMetric,
		statusScrapeErrorsTotalMetric:         statusScrapeErrorsTotalMetric,
		lastStatusScrapeErrorMetric:           lastStatusScrapeErrorMetric,
		lastStatusScrapeTimestampMetric:       lastStatusScrapeTimestampMetric,
		lastStatusScrapeDurationSecondsMetric: lastStatusScrapeDurationSecondsMetric,
	}
}

func (c StatusCollector) Collect(ch chan<- prometheus.Metric) {
	var begun = time.Now()

	errorMetric := float64(0)
	if err := c.reportStatusMetrics(ch); err != nil {
		errorMetric = float64(1)
		c.statusScrapeErrorsTotalMetric.Inc()
	}
	c.statusScrapeErrorsTotalMetric.Collect(ch)

	c.statusScrapesTotalMetric.Inc()
	c.statusScrapesTotalMetric.Collect(ch)

	c.lastStatusScrapeErrorMetric.Set(errorMetric)
	c.lastStatusScrapeErrorMetric.Collect(ch)

	c.lastStatusScrapeTimestampMetric.Set(float64(time.Now().Unix()))
	c.lastStatusScrapeTimestampMetric.Collect(ch)

	c.lastStatusScrapeDurationSecondsMetric.Set(time.Since(begun).Seconds())
	c.lastStatusScrapeDurationSecondsMetric.Collect(ch)
}

func (c StatusCollector) Describe(ch chan<- *prometheus.Desc) {
	c.pendingTasksTotalMetric.Describe(ch)
	c.runningTasksTotalMetric.Describe(ch)
	c.scheduleQueueTotalMetric.Describe(ch)
	c.runQueueTotalMetric.Describe(ch)
	c.statusScrapesTotalMetric.Describe(ch)
	c.statusScrapeErrorsTotalMetric.Describe(ch)
	c.lastStatusScrapeErrorMetric.Describe(ch)
	c.lastStatusScrapeTimestampMetric.Describe(ch)
	c.lastStatusScrapeDurationSecondsMetric.Describe(ch)
}

func (c StatusCollector) reportStatusMetrics(ch chan<- prometheus.Metric) error {
	var internalStatus InternalStatus

	uri := api.ShieldURI("/v1/status/internal")
	if err := uri.Get(&internalStatus); err != nil {
		log.Errorf("Error while getting internal status: %v", err)
		return err
	}

	c.pendingTasksTotalMetric.Set(float64(len(internalStatus.PendingTasks)))
	c.pendingTasksTotalMetric.Collect(ch)

	c.runningTasksTotalMetric.Set(float64(len(internalStatus.RunningTasks)))
	c.runningTasksTotalMetric.Collect(ch)

	c.scheduleQueueTotalMetric.Set(float64(len(internalStatus.ScheduleQueue)))
	c.scheduleQueueTotalMetric.Collect(ch)

	c.runQueueTotalMetric.Set(float64(len(internalStatus.RunQueue)))
	c.runQueueTotalMetric.Collect(ch)

	return nil
}
