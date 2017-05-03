package collectors_test

import (
	"flag"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/starkandwayne/shield/api"

	. "github.com/cloudfoundry-community/shield_exporter/collectors"
	. "github.com/cloudfoundry-community/shield_exporter/utils/test_matchers"
)

func init() {
	flag.Set("log.level", "fatal")
}

var _ = Describe("StatusCollectors", func() {
	var (
		err    error
		server *ghttp.Server

		namespace   = "test_namespace"
		environment = "test_environment"
		backendName = "test_backend"

		username = "fake_username"
		password = "fake_password"

		pendingTasks  = []interface{}{"pending_task1"}
		runningTasks  = []interface{}{"running_task1", "running_task1"}
		scheduleQueue = []interface{}{"schedule_queue_1"}
		runQueue      = []interface{}{"run_queue_1", "run_queue_2"}

		pendingTasksTotalMetric               prometheus.Gauge
		runningTasksTotalMetric               prometheus.Gauge
		scheduleQueueTotalMetric              prometheus.Gauge
		runQueueTotalMetric                   prometheus.Gauge
		statusScrapesTotalMetric              prometheus.Counter
		statusScrapeErrorsTotalMetric         prometheus.Counter
		lastStatusScrapeErrorMetric           prometheus.Gauge
		lastStatusScrapeTimestampMetric       prometheus.Gauge
		lastStatusScrapeDurationSecondsMetric prometheus.Gauge

		statusCollector *StatusCollector
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

		pendingTasksTotalMetric = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   "status",
				Name:        "pending_tasks_total",
				Help:        "Total number of Shield pending Tasks.",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
		)
		pendingTasksTotalMetric.Set(float64(len(pendingTasks)))

		runningTasksTotalMetric = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   "status",
				Name:        "running_tasks_total",
				Help:        "Total number of Shield running Tasks.",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
		)
		runningTasksTotalMetric.Set(float64(len(runningTasks)))

		scheduleQueueTotalMetric = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   "status",
				Name:        "schedule_queue_total",
				Help:        "Total number of Shield Tasks in the supervisor scheduler queue.",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
		)
		scheduleQueueTotalMetric.Set(float64(len(scheduleQueue)))

		runQueueTotalMetric = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   "status",
				Name:        "run_queue_total",
				Help:        "Total number of Shield Tasks in the supervisor run queue.",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
		)
		runQueueTotalMetric.Set(float64(len(runQueue)))

		statusScrapesTotalMetric = prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace:   namespace,
				Subsystem:   "status",
				Name:        "scrapes_total",
				Help:        "Total number of scrapes for Shield Status.",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
		)
		statusScrapesTotalMetric.Inc()

		statusScrapeErrorsTotalMetric = prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace:   namespace,
				Subsystem:   "status",
				Name:        "scrape_errors_total",
				Help:        "Total number of scrape errors of Shield Status.",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
		)

		lastStatusScrapeErrorMetric = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   "",
				Name:        "last_status_scrape_error",
				Help:        "Whether the last scrape of Status metrics from Shield resulted in an error (1 for error, 0 for success).",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
		)

		lastStatusScrapeTimestampMetric = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   "",
				Name:        "last_status_scrape_timestamp",
				Help:        "Number of seconds since 1970 since last scrape of Status metrics from Shield.",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
		)

		lastStatusScrapeDurationSecondsMetric = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   "",
				Name:        "last_status_scrape_duration_seconds",
				Help:        "Duration of the last scrape of Status metrics from Shield.",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
		)
	})

	JustBeforeEach(func() {
		statusCollector = NewStatusCollector(namespace, environment, backendName)
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
			go statusCollector.Describe(descriptions)
		})

		It("returns a status_pending_tasks_total metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(pendingTasksTotalMetric.Desc())))
		})

		It("returns a status_running_tasks_total metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(runningTasksTotalMetric.Desc())))
		})

		It("returns a status_schedule_queue_total metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(scheduleQueueTotalMetric.Desc())))
		})

		It("returns a status_run_queue_total metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(runQueueTotalMetric.Desc())))
		})

		It("returns a status_scrapes_total metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(statusScrapesTotalMetric.Desc())))
		})

		It("returns a status_scrape_errors_total metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(statusScrapeErrorsTotalMetric.Desc())))
		})

		It("returns a last_status_scrape_error metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(lastStatusScrapeErrorMetric.Desc())))
		})

		It("returns a last_status_scrape_timestamp metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(lastStatusScrapeTimestampMetric.Desc())))
		})

		It("returns a last_status_scrape_duration_seconds metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(lastStatusScrapeDurationSecondsMetric.Desc())))
		})
	})

	Describe("Collect", func() {
		var (
			statusCode     int
			statusResponse InternalStatus
			metrics        chan prometheus.Metric
		)

		BeforeEach(func() {
			statusCode = http.StatusOK
			statusResponse = InternalStatus{
				PendingTasks:  pendingTasks,
				RunningTasks:  runningTasks,
				ScheduleQueue: scheduleQueue,
				RunQueue:      runQueue,
			}
			metrics = make(chan prometheus.Metric)
		})

		JustBeforeEach(func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/v1/status/internal"),
					ghttp.VerifyBasicAuth(username, password),
					ghttp.RespondWithJSONEncodedPtr(&statusCode, &statusResponse),
				),
			)
			go statusCollector.Collect(metrics)
		})

		It("returns a status_pending_tasks_total metric", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(pendingTasksTotalMetric)))
		})

		It("returns a status_running_tasks_total metric", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(runningTasksTotalMetric)))
		})

		It("returns a status_schedule_queue_total metric", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(scheduleQueueTotalMetric)))
		})

		It("returns a status_run_queue_total metric", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(runQueueTotalMetric)))
		})

		It("returns a status_scrapes_total metric", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(statusScrapesTotalMetric)))
		})

		It("returns a status_scrape_errors_total metric", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(statusScrapeErrorsTotalMetric)))
		})

		It("returns a last_status_scrape_error metric", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(lastStatusScrapeErrorMetric)))
		})

		Context("when it fails to the the internal status", func() {
			BeforeEach(func() {
				statusCode = http.StatusInternalServerError
				statusScrapeErrorsTotalMetric.Inc()
				lastStatusScrapeErrorMetric.Set(1)
			})

			It("returns a status_scrape_errors_total metric", func() {
				Eventually(metrics).Should(Receive(PrometheusMetric(statusScrapeErrorsTotalMetric)))
			})

			It("returns a last_status_scrape_error metric", func() {
				Eventually(metrics).Should(Receive(PrometheusMetric(lastStatusScrapeErrorMetric)))
			})
		})
	})
})
