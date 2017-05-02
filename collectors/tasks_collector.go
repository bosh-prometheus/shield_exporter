package collectors

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"github.com/starkandwayne/shield/api"
)

type TasksCollector struct {
	namespace                            string
	environment                          string
	backendName                          string
	tasksTotalMetric                     *prometheus.GaugeVec
	tasksDurationSecondsMetric           *prometheus.SummaryVec
	tasksScrapesTotalMetric              prometheus.Counter
	tasksScrapeErrorsTotalMetric         prometheus.Counter
	lastTasksScrapeErrorMetric           prometheus.Gauge
	lastTasksScrapeTimestampMetric       prometheus.Gauge
	lastTasksScrapeDurationSecondsMetric prometheus.Gauge
}

func NewTasksCollector(
	namespace string,
	environment string,
	backendName string,
) *TasksCollector {
	tasksTotalMetric := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "tasks",
			Name:        "total",
			Help:        "Labeled total number of Shield Tasks.",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
		[]string{"task_operation", "task_status"},
	)

	tasksDurationSecondsMetric := prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:   namespace,
			Subsystem:   "tasks",
			Name:        "duration_seconds",
			Help:        "Labeled summary of Shield Task durations in seconds.",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
		[]string{"task_operation", "task_status"},
	)

	tasksScrapesTotalMetric := prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace:   namespace,
			Subsystem:   "tasks",
			Name:        "scrapes_total",
			Help:        "Total number of scrapes for Shield Tasks.",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
	)

	tasksScrapeErrorsTotalMetric := prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace:   namespace,
			Subsystem:   "tasks",
			Name:        "scrape_errors_total",
			Help:        "Total number of scrape errors of Shield Tasks.",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
	)

	lastTasksScrapeErrorMetric := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "",
			Name:        "last_tasks_scrape_error",
			Help:        "Whether the last scrape of Task metrics from Shield resulted in an error (1 for error, 0 for success).",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
	)

	lastTasksScrapeTimestampMetric := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "",
			Name:        "last_tasks_scrape_timestamp",
			Help:        "Number of seconds since 1970 since last scrape of Task metrics from Shield.",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
	)

	lastTasksScrapeDurationSecondsMetric := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "",
			Name:        "last_tasks_scrape_duration_seconds",
			Help:        "Duration of the last scrape of Task metrics from Shield.",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
	)

	return &TasksCollector{
		namespace:                            namespace,
		environment:                          environment,
		backendName:                          backendName,
		tasksTotalMetric:                     tasksTotalMetric,
		tasksDurationSecondsMetric:           tasksDurationSecondsMetric,
		tasksScrapesTotalMetric:              tasksScrapesTotalMetric,
		tasksScrapeErrorsTotalMetric:         tasksScrapeErrorsTotalMetric,
		lastTasksScrapeErrorMetric:           lastTasksScrapeErrorMetric,
		lastTasksScrapeTimestampMetric:       lastTasksScrapeTimestampMetric,
		lastTasksScrapeDurationSecondsMetric: lastTasksScrapeDurationSecondsMetric,
	}
}

func (c TasksCollector) Collect(ch chan<- prometheus.Metric) {
	var begun = time.Now()

	errorMetric := float64(0)
	if err := c.reportTasksMetrics(ch); err != nil {
		errorMetric = float64(1)
		c.tasksScrapeErrorsTotalMetric.Inc()
	}
	c.tasksScrapeErrorsTotalMetric.Collect(ch)

	c.tasksScrapesTotalMetric.Inc()
	c.tasksScrapesTotalMetric.Collect(ch)

	c.lastTasksScrapeErrorMetric.Set(errorMetric)
	c.lastTasksScrapeErrorMetric.Collect(ch)

	c.lastTasksScrapeTimestampMetric.Set(float64(time.Now().Unix()))
	c.lastTasksScrapeTimestampMetric.Collect(ch)

	c.lastTasksScrapeDurationSecondsMetric.Set(time.Since(begun).Seconds())
	c.lastTasksScrapeDurationSecondsMetric.Collect(ch)
}

func (c TasksCollector) Describe(ch chan<- *prometheus.Desc) {
	c.tasksTotalMetric.Describe(ch)
	c.tasksDurationSecondsMetric.Describe(ch)
	c.tasksScrapesTotalMetric.Describe(ch)
	c.tasksScrapeErrorsTotalMetric.Describe(ch)
	c.lastTasksScrapeErrorMetric.Describe(ch)
	c.lastTasksScrapeTimestampMetric.Describe(ch)
	c.lastTasksScrapeDurationSecondsMetric.Describe(ch)
}

func (c TasksCollector) reportTasksMetrics(ch chan<- prometheus.Metric) error {
	c.tasksTotalMetric.Reset()
	c.tasksDurationSecondsMetric.Reset()

	tasks, err := api.GetTasks(api.TaskFilter{})
	if err != nil {
		log.Errorf("Error while listing tasks: %v", err)
		return err
	}

	for _, task := range tasks {
		c.tasksTotalMetric.WithLabelValues(task.Op, task.Status).Inc()

		if !task.StartedAt.IsZero() && !task.StoppedAt.IsZero() {
			duration := task.StoppedAt.Time().Unix() - task.StartedAt.Time().Unix()
			if duration >= 0 {
				c.tasksDurationSecondsMetric.WithLabelValues(task.Op, task.Status).Observe(float64(duration))
			}
		}
	}

	c.tasksTotalMetric.Collect(ch)
	c.tasksDurationSecondsMetric.Collect(ch)

	return nil
}
