package collectors_test

import (
	"flag"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/starkandwayne/goutils/timestamp"
	"github.com/starkandwayne/shield/api"

	. "github.com/cloudfoundry-community/shield_exporter/collectors"
	. "github.com/cloudfoundry-community/shield_exporter/utils/test_matchers"
)

func init() {
	flag.Set("log.level", "fatal")
}

var _ = Describe("TasksCollectors", func() {
	var (
		err    error
		server *ghttp.Server

		namespace   = "test_namespace"
		environment = "test_environment"
		backendName = "test_backend"

		username = "fake_username"
		password = "fake_password"

		TaskOperation1 = "task_operation_1"
		TaskOperation2 = "task_operation_2"
		TaskStatus1    = "task_status_1"
		TaskStatus2    = "task_status_2"

		tasksTotalMetric                     *prometheus.GaugeVec
		tasksDurationSecondsMetric           *prometheus.SummaryVec
		tasksScrapesTotalMetric              prometheus.Counter
		tasksScrapeErrorsTotalMetric         prometheus.Counter
		lastTasksScrapeErrorMetric           prometheus.Gauge
		lastTasksScrapeTimestampMetric       prometheus.Gauge
		lastTasksScrapeDurationSecondsMetric prometheus.Gauge

		tasksCollector *TasksCollector
	)

	BeforeEach(func() {
		server = ghttp.NewServer()
		server.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/v1/ping"),
				ghttp.RespondWith(http.StatusOK, "{}"),
			),
		)

		api.Cfg = &api.Config{
			Backend:  "default",
			Backends: map[string]string{},
			Aliases:  map[string]string{},
		}

		err = api.Cfg.AddBackend(server.URL(), "default")
		Expect(err).ToNot(HaveOccurred())

		authToken := api.BasicAuthToken(username, password)
		err = api.Cfg.UpdateBackend("default", authToken)
		Expect(err).ToNot(HaveOccurred())

		tasksTotalMetric = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   "tasks",
				Name:        "total",
				Help:        "Labeled total number of Shield Tasks.",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
			[]string{"task_operation", "task_status"},
		)
		tasksTotalMetric.WithLabelValues(TaskOperation1, TaskStatus1).Set(1)
		tasksTotalMetric.WithLabelValues(TaskOperation1, TaskStatus2).Set(1)
		tasksTotalMetric.WithLabelValues(TaskOperation2, TaskStatus1).Set(1)
		tasksTotalMetric.WithLabelValues(TaskOperation2, TaskStatus2).Set(1)

		tasksDurationSecondsMetric = prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Namespace:   namespace,
				Subsystem:   "tasks",
				Name:        "duration_seconds",
				Help:        "Labeled summary of Shield Task durations in seconds.",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
			[]string{"task_operation", "task_status"},
		)
		tasksDurationSecondsMetric.WithLabelValues(TaskOperation1, TaskStatus1).Observe(1)
		tasksDurationSecondsMetric.WithLabelValues(TaskOperation1, TaskStatus2).Observe(0)
		tasksDurationSecondsMetric.WithLabelValues(TaskOperation2, TaskStatus1).Observe(0)
		tasksDurationSecondsMetric.WithLabelValues(TaskOperation2, TaskStatus2).Observe(0)

		tasksScrapesTotalMetric = prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace:   namespace,
				Subsystem:   "tasks",
				Name:        "scrapes_total",
				Help:        "Total number of scrapes for Shield Tasks.",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
		)
		tasksScrapesTotalMetric.Inc()

		tasksScrapeErrorsTotalMetric = prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace:   namespace,
				Subsystem:   "tasks",
				Name:        "scrape_errors_total",
				Help:        "Total number of scrape errors of Shield Tasks.",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
		)

		lastTasksScrapeErrorMetric = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   "",
				Name:        "last_tasks_scrape_error",
				Help:        "Whether the last scrape of Task metrics from Shield resulted in an error (1 for error, 0 for success).",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
		)

		lastTasksScrapeTimestampMetric = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   "",
				Name:        "last_tasks_scrape_timestamp",
				Help:        "Number of seconds since 1970 since last scrape of Task metrics from Shield.",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
		)

		lastTasksScrapeDurationSecondsMetric = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   "",
				Name:        "last_tasks_scrape_duration_seconds",
				Help:        "Duration of the last scrape of Task metrics from Shield.",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
		)
	})

	JustBeforeEach(func() {
		tasksCollector = NewTasksCollector(namespace, environment, backendName)
	})

	AfterEach(func() {
		server.Close()
	})

	Describe("Describe", func() {
		var (
			descriptions chan *prometheus.Desc
		)

		BeforeEach(func() {
			descriptions = make(chan *prometheus.Desc)
		})

		JustBeforeEach(func() {
			go tasksCollector.Describe(descriptions)
		})

		It("returns a tasks_total metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(tasksTotalMetric.WithLabelValues(TaskOperation1, TaskStatus1).Desc())))
		})

		It("returns a tasks_duration_seconds metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(tasksDurationSecondsMetric.WithLabelValues(TaskOperation1, TaskStatus1).Desc())))
		})

		It("returns a tasks_scrapes_total metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(tasksScrapesTotalMetric.Desc())))
		})

		It("returns a tasks_scrape_errors_total metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(tasksScrapeErrorsTotalMetric.Desc())))
		})

		It("returns a last_tasks_scrape_error metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(lastTasksScrapeErrorMetric.Desc())))
		})

		It("returns a last_tasks_scrape_timestamp metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(lastTasksScrapeTimestampMetric.Desc())))
		})

		It("returns a last_tasks_scrape_duration_seconds metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(lastTasksScrapeDurationSecondsMetric.Desc())))
		})
	})

	Describe("Collect", func() {
		var (
			statusCode    int
			tasksResponse []api.Task
			metrics       chan prometheus.Metric
		)

		BeforeEach(func() {
			statusCode = http.StatusOK
			tasksResponse = []api.Task{
				api.Task{
					Op:        TaskOperation1,
					Status:    TaskStatus1,
					StartedAt: timestamp.NewTimestamp(time.Unix(1, 0)),
					StoppedAt: timestamp.NewTimestamp(time.Unix(2, 0)),
				},
				api.Task{
					Op:        TaskOperation1,
					Status:    TaskStatus2,
					StartedAt: timestamp.NewTimestamp(time.Unix(1, 0)),
					StoppedAt: timestamp.NewTimestamp(time.Unix(1, 0)),
				},
				api.Task{
					Op:        TaskOperation2,
					Status:    TaskStatus1,
					StartedAt: timestamp.NewTimestamp(time.Unix(1, 0)),
				},
				api.Task{
					Op:        TaskOperation2,
					Status:    TaskStatus2,
					StoppedAt: timestamp.NewTimestamp(time.Unix(1, 0)),
				},
			}
			metrics = make(chan prometheus.Metric)
		})

		JustBeforeEach(func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/v1/tasks"),
					ghttp.VerifyBasicAuth(username, password),
					ghttp.RespondWithJSONEncodedPtr(&statusCode, &tasksResponse),
				),
			)
			go tasksCollector.Collect(metrics)
		})

		It("returns a tasks_total metric for task operation 1, task status 1", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(tasksTotalMetric.WithLabelValues(TaskOperation1, TaskStatus1))))
		})

		It("returns a tasks_total metric for task operation 1, task status 2", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(tasksTotalMetric.WithLabelValues(TaskOperation1, TaskStatus2))))
		})

		It("returns a tasks_total metric for task operation 2, task status 1", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(tasksTotalMetric.WithLabelValues(TaskOperation2, TaskStatus1))))
		})

		It("returns a tasks_total metric for task operation 2, task status 2", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(tasksTotalMetric.WithLabelValues(TaskOperation2, TaskStatus2))))
		})

		It("returns a tasks_duration_seconds metric for task operation 1, task status 1", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(tasksDurationSecondsMetric.WithLabelValues(TaskOperation1, TaskStatus1))))
		})

		It("returns a tasks_duration_seconds metric for task operation 1, task status 2", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(tasksDurationSecondsMetric.WithLabelValues(TaskOperation1, TaskStatus2))))
		})

		It("returns a tasks_scrapes_total metric", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(tasksScrapesTotalMetric)))
		})

		It("returns a tasks_scrape_errors_total metric", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(tasksScrapeErrorsTotalMetric)))
		})

		It("returns a last_tasks_scrape_error metric", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(lastTasksScrapeErrorMetric)))
		})

		Context("when it fails to list the tasks", func() {
			BeforeEach(func() {
				statusCode = http.StatusInternalServerError
				tasksScrapeErrorsTotalMetric.Inc()
				lastTasksScrapeErrorMetric.Set(1)
			})

			It("returns a tasks_scrape_errors_total metric", func() {
				Eventually(metrics).Should(Receive(PrometheusMetric(tasksScrapeErrorsTotalMetric)))
			})

			It("returns a last_tasks_scrape_error metric", func() {
				Eventually(metrics).Should(Receive(PrometheusMetric(lastTasksScrapeErrorMetric)))
			})
		})
	})
})
