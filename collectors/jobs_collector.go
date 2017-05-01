package collectors

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"github.com/starkandwayne/shield/api"
)

type JobsCollector struct {
	namespace                           string
	environment                         string
	backendName                         string
	jobsTotalMetric                     *prometheus.GaugeVec
	jobsScrapesTotalMetric              prometheus.Counter
	jobsScrapeErrorsTotalMetric         prometheus.Counter
	lastJobsScrapeErrorMetric           prometheus.Gauge
	lastJobsScrapeTimestampMetric       prometheus.Gauge
	lastJobsScrapeDurationSecondsMetric prometheus.Gauge
}

func NewJobsCollector(
	namespace string,
	environment string,
	backendName string,
) *JobsCollector {
	jobsTotalMetric := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "jobs",
			Name:        "total",
			Help:        "Labeled total number of Shield Jobs.",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
		[]string{"job_paused", "store_plugin", "target_plugin"},
	)

	jobsScrapesTotalMetric := prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace:   namespace,
			Subsystem:   "jobs",
			Name:        "scrapes_total",
			Help:        "Total number of scrapes for Shield Jobs.",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
	)

	jobsScrapeErrorsTotalMetric := prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace:   namespace,
			Subsystem:   "jobs",
			Name:        "scrape_errors_total",
			Help:        "Total number of scrape errors of Shield Jobs.",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
	)

	lastJobsScrapeErrorMetric := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "",
			Name:        "last_jobs_scrape_error",
			Help:        "Whether the last scrape of Job metrics from Shield resulted in an error (1 for error, 0 for success).",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
	)

	lastJobsScrapeTimestampMetric := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "",
			Name:        "last_jobs_scrape_timestamp",
			Help:        "Number of seconds since 1970 since last scrape of Job metrics from Shield.",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
	)

	lastJobsScrapeDurationSecondsMetric := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "",
			Name:        "last_jobs_scrape_duration_seconds",
			Help:        "Duration of the last scrape of Job metrics from Shield.",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
	)

	return &JobsCollector{
		namespace:                           namespace,
		environment:                         environment,
		backendName:                         backendName,
		jobsTotalMetric:                     jobsTotalMetric,
		jobsScrapesTotalMetric:              jobsScrapesTotalMetric,
		jobsScrapeErrorsTotalMetric:         jobsScrapeErrorsTotalMetric,
		lastJobsScrapeErrorMetric:           lastJobsScrapeErrorMetric,
		lastJobsScrapeTimestampMetric:       lastJobsScrapeTimestampMetric,
		lastJobsScrapeDurationSecondsMetric: lastJobsScrapeDurationSecondsMetric,
	}
}

func (c JobsCollector) Collect(ch chan<- prometheus.Metric) {
	var begun = time.Now()

	errorMetric := float64(0)
	if err := c.reportJobsMetrics(ch); err != nil {
		errorMetric = float64(1)
		c.jobsScrapeErrorsTotalMetric.Inc()
	}
	c.jobsScrapeErrorsTotalMetric.Collect(ch)

	c.jobsScrapesTotalMetric.Inc()
	c.jobsScrapesTotalMetric.Collect(ch)

	c.lastJobsScrapeErrorMetric.Set(errorMetric)
	c.lastJobsScrapeErrorMetric.Collect(ch)

	c.lastJobsScrapeTimestampMetric.Set(float64(time.Now().Unix()))
	c.lastJobsScrapeTimestampMetric.Collect(ch)

	c.lastJobsScrapeDurationSecondsMetric.Set(time.Since(begun).Seconds())
	c.lastJobsScrapeDurationSecondsMetric.Collect(ch)
}

func (c JobsCollector) Describe(ch chan<- *prometheus.Desc) {
	c.jobsTotalMetric.Describe(ch)
	c.jobsScrapesTotalMetric.Describe(ch)
	c.jobsScrapeErrorsTotalMetric.Describe(ch)
	c.lastJobsScrapeErrorMetric.Describe(ch)
	c.lastJobsScrapeTimestampMetric.Describe(ch)
	c.lastJobsScrapeDurationSecondsMetric.Describe(ch)
}

func (c JobsCollector) reportJobsMetrics(ch chan<- prometheus.Metric) error {
	c.jobsTotalMetric.Reset()

	jobs, err := api.GetJobs(api.JobFilter{})
	if err != nil {
		log.Errorf("Error while listing jobs: %v", err)
		return err
	}

	for _, job := range jobs {
		c.jobsTotalMetric.WithLabelValues(strconv.FormatBool(job.Paused), job.StorePlugin, job.TargetPlugin).Inc()
	}

	c.jobsTotalMetric.Collect(ch)

	return nil
}
