package collectors

import (
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"github.com/starkandwayne/shield/api"
)

const (
	PendingStatus  = "pending"
	RunningStatus  = "running"
	CanceledStatus = "canceled"
	FailedStatus   = "failed"
	DoneStatus     = "done"
)

type JobsCollector struct {
	namespace                           string
	environment                         string
	backendName                         string
	jobLastRunMetric                    *prometheus.GaugeVec
	jobNextRunMetric                    *prometheus.GaugeVec
	jobStatusMetric                     *prometheus.GaugeVec
	jobPausedMetric                     *prometheus.GaugeVec
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
	jobLastRunMetric := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "job",
			Name:        "last_run",
			Help:        "Number of seconds since 1970 since last run of a Shield Job.",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
		[]string{"job_name"},
	)

	jobNextRunMetric := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "job",
			Name:        "next_run",
			Help:        "Number of seconds since 1970 until next run of a Shield Job.",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
		[]string{"job_name"},
	)

	jobStatusMetric := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "job",
			Name:        "status",
			Help:        "Shield Job status (0 for unknow, 1 for pending, 2 for running, 3 for canceled, 4 for failed, 5 for done).",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
		[]string{"job_name"},
	)

	jobPausedMetric := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "job",
			Name:        "paused",
			Help:        "Shield Job pause status (1 for paused, 0 for unpaused).",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
		[]string{"job_name"},
	)

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
		jobLastRunMetric:                    jobLastRunMetric,
		jobNextRunMetric:                    jobNextRunMetric,
		jobStatusMetric:                     jobStatusMetric,
		jobPausedMetric:                     jobPausedMetric,
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
	c.jobLastRunMetric.Describe(ch)
	c.jobNextRunMetric.Describe(ch)
	c.jobStatusMetric.Describe(ch)
	c.jobPausedMetric.Describe(ch)
	c.jobsTotalMetric.Describe(ch)
	c.jobsScrapesTotalMetric.Describe(ch)
	c.jobsScrapeErrorsTotalMetric.Describe(ch)
	c.lastJobsScrapeErrorMetric.Describe(ch)
	c.lastJobsScrapeTimestampMetric.Describe(ch)
	c.lastJobsScrapeDurationSecondsMetric.Describe(ch)
}

func (c JobsCollector) reportJobsMetrics(ch chan<- prometheus.Metric) error {
	c.jobLastRunMetric.Reset()
	c.jobNextRunMetric.Reset()
	c.jobStatusMetric.Reset()
	c.jobPausedMetric.Reset()
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

	jobsStatus, err := api.GetJobsStatus()
	if err != nil {
		if strings.Contains(err.Error(), "Error 501 Not Implemented") {
			log.Debug("Shield backend does not implement `/v1/status/jobs` API")
			return nil
		}
		log.Errorf("Error while getting jobs status: %+v", err)
		return err
	}

	for _, jobHealth := range jobsStatus {
		c.jobLastRunMetric.WithLabelValues(jobHealth.Name).Set(float64(jobHealth.LastRun))
		c.jobNextRunMetric.WithLabelValues(jobHealth.Name).Set(float64(jobHealth.NextRun))
		var jobStatus float64
		switch jobHealth.Status {
		case PendingStatus:
			jobStatus = 1
		case RunningStatus:
			jobStatus = 2
		case CanceledStatus:
			jobStatus = 3
		case FailedStatus:
			jobStatus = 4
		case DoneStatus:
			jobStatus = 5
		default:
			jobStatus = 0
		}
		c.jobStatusMetric.WithLabelValues(jobHealth.Name).Set(jobStatus)
		jobPaused := 0
		if jobHealth.Paused {
			jobPaused = 1
		}
		c.jobPausedMetric.WithLabelValues(jobHealth.Name).Set(float64(jobPaused))
	}

	c.jobLastRunMetric.Collect(ch)
	c.jobNextRunMetric.Collect(ch)
	c.jobStatusMetric.Collect(ch)
	c.jobPausedMetric.Collect(ch)

	return nil
}
