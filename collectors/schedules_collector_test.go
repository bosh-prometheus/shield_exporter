package collectors_test

import (
	"flag"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/starkandwayne/shield/api"

	. "github.com/bosh-prometheus/shield_exporter/collectors"
	. "github.com/bosh-prometheus/shield_exporter/utils/test_matchers"
)

func init() {
	flag.Set("log.level", "fatal")
}

var _ = Describe("SchedulesCollectors", func() {
	var (
		err    error
		server *ghttp.Server

		namespace   = "test_namespace"
		environment = "test_environment"
		backendName = "test_backend"

		username = "fake_username"
		password = "fake_password"

		schedulesTotalMetric                     prometheus.Gauge
		schedulesScrapesTotalMetric              prometheus.Counter
		schedulesScrapeErrorsTotalMetric         prometheus.Counter
		lastSchedulesScrapeErrorMetric           prometheus.Gauge
		lastSchedulesScrapeTimestampMetric       prometheus.Gauge
		lastSchedulesScrapeDurationSecondsMetric prometheus.Gauge

		schedulesCollector *SchedulesCollector
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

		schedulesTotalMetric = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   "schedules",
				Name:        "total",
				Help:        "Total number of Shield Schedules.",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
		)

		schedulesTotalMetric.Set(2)

		schedulesScrapesTotalMetric = prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace:   namespace,
				Subsystem:   "schedules",
				Name:        "scrapes_total",
				Help:        "Total number of scrapes for Shield Schedules.",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
		)
		schedulesScrapesTotalMetric.Inc()

		schedulesScrapeErrorsTotalMetric = prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace:   namespace,
				Subsystem:   "schedules",
				Name:        "scrape_errors_total",
				Help:        "Total number of scrape errors of Shield Schedules.",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
		)

		lastSchedulesScrapeErrorMetric = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   "",
				Name:        "last_schedules_scrape_error",
				Help:        "Whether the last scrape of Schedule metrics from Shield resulted in an error (1 for error, 0 for success).",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
		)

		lastSchedulesScrapeTimestampMetric = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   "",
				Name:        "last_schedules_scrape_timestamp",
				Help:        "Number of seconds since 1970 since last scrape of Schedule metrics from Shield.",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
		)

		lastSchedulesScrapeDurationSecondsMetric = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   "",
				Name:        "last_schedules_scrape_duration_seconds",
				Help:        "Duration of the last scrape of Schedule metrics from Shield.",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
		)
	})

	JustBeforeEach(func() {
		schedulesCollector = NewSchedulesCollector(namespace, environment, backendName)
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
			go schedulesCollector.Describe(descriptions)
		})

		It("returns a schedules_total metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(schedulesTotalMetric.Desc())))
		})

		It("returns a schedules_scrapes_total metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(schedulesScrapesTotalMetric.Desc())))
		})

		It("returns a schedules_scrape_errors_total metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(schedulesScrapeErrorsTotalMetric.Desc())))
		})

		It("returns a last_schedules_scrape_error metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(lastSchedulesScrapeErrorMetric.Desc())))
		})

		It("returns a last_schedules_scrape_timestamp metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(lastSchedulesScrapeTimestampMetric.Desc())))
		})

		It("returns a last_schedules_scrape_duration_seconds metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(lastSchedulesScrapeDurationSecondsMetric.Desc())))
		})
	})

	Describe("Collect", func() {
		var (
			statusCode        int
			schedulesResponse []api.Schedule
			metrics           chan prometheus.Metric
		)

		BeforeEach(func() {
			statusCode = http.StatusOK
			schedulesResponse = []api.Schedule{
				api.Schedule{
					Name: "fake_schedule_1",
				},
				api.Schedule{
					Name: "fake_schedule_2",
				},
			}
			metrics = make(chan prometheus.Metric)
		})

		JustBeforeEach(func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/v1/schedules"),
					ghttp.VerifyBasicAuth(username, password),
					ghttp.RespondWithJSONEncodedPtr(&statusCode, &schedulesResponse),
				),
			)
			go schedulesCollector.Collect(metrics)
		})

		It("returns a schedules_total metric", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(schedulesTotalMetric)))
		})

		It("returns a schedules_scrapes_total metric", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(schedulesScrapesTotalMetric)))
		})

		It("returns a schedules_scrape_errors_total metric", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(schedulesScrapeErrorsTotalMetric)))
		})

		It("returns a last_schedules_scrape_error metric", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(lastSchedulesScrapeErrorMetric)))
		})

		Context("when it fails to list the schedules", func() {
			BeforeEach(func() {
				statusCode = http.StatusInternalServerError
				schedulesScrapeErrorsTotalMetric.Inc()
				lastSchedulesScrapeErrorMetric.Set(1)
			})

			It("returns a schedules_scrape_errors_total metric", func() {
				Eventually(metrics).Should(Receive(PrometheusMetric(schedulesScrapeErrorsTotalMetric)))
			})

			It("returns a last_schedules_scrape_error metric", func() {
				Eventually(metrics).Should(Receive(PrometheusMetric(lastSchedulesScrapeErrorMetric)))
			})
		})
	})
})
