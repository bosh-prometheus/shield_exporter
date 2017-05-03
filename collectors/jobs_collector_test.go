package collectors_test

import (
	"flag"
	"net/http"
	"strconv"

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

var _ = Describe("JobsCollectors", func() {
	var (
		err    error
		server *ghttp.Server

		namespace   = "test_namespace"
		environment = "test_environment"
		backendName = "test_backend"

		username = "fake_username"
		password = "fake_password"

		jobPaused1    = true
		jobPaused2    = false
		storePlugin1  = "store_plugin_1"
		storePlugin2  = "store_plugin_2"
		targetPlugin1 = "target_plugin_1"
		targetPlugin2 = "target_plugin_2"

		jobsTotalMetric                     *prometheus.GaugeVec
		jobsScrapesTotalMetric              prometheus.Counter
		jobsScrapeErrorsTotalMetric         prometheus.Counter
		lastJobsScrapeErrorMetric           prometheus.Gauge
		lastJobsScrapeTimestampMetric       prometheus.Gauge
		lastJobsScrapeDurationSecondsMetric prometheus.Gauge

		jobsCollector *JobsCollector
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

		jobsTotalMetric = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   "jobs",
				Name:        "total",
				Help:        "Labeled total number of Shield Jobs.",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
			[]string{"job_paused", "store_plugin", "target_plugin"},
		)
		jobsTotalMetric.WithLabelValues(strconv.FormatBool(jobPaused1), storePlugin1, targetPlugin1).Set(1)
		jobsTotalMetric.WithLabelValues(strconv.FormatBool(jobPaused2), storePlugin1, targetPlugin2).Set(1)
		jobsTotalMetric.WithLabelValues(strconv.FormatBool(jobPaused1), storePlugin2, targetPlugin1).Set(1)
		jobsTotalMetric.WithLabelValues(strconv.FormatBool(jobPaused2), storePlugin2, targetPlugin2).Set(1)

		jobsScrapesTotalMetric = prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace:   namespace,
				Subsystem:   "jobs",
				Name:        "scrapes_total",
				Help:        "Total number of scrapes for Shield Jobs.",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
		)
		jobsScrapesTotalMetric.Inc()

		jobsScrapeErrorsTotalMetric = prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace:   namespace,
				Subsystem:   "jobs",
				Name:        "scrape_errors_total",
				Help:        "Total number of scrape errors of Shield Jobs.",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
		)

		lastJobsScrapeErrorMetric = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   "",
				Name:        "last_jobs_scrape_error",
				Help:        "Whether the last scrape of Job metrics from Shield resulted in an error (1 for error, 0 for success).",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
		)

		lastJobsScrapeTimestampMetric = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   "",
				Name:        "last_jobs_scrape_timestamp",
				Help:        "Number of seconds since 1970 since last scrape of Job metrics from Shield.",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
		)

		lastJobsScrapeDurationSecondsMetric = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   "",
				Name:        "last_jobs_scrape_duration_seconds",
				Help:        "Duration of the last scrape of Job metrics from Shield.",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
		)
	})

	JustBeforeEach(func() {
		jobsCollector = NewJobsCollector(namespace, environment, backendName)
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
			go jobsCollector.Describe(descriptions)
		})

		It("returns a jobs_total metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(jobsTotalMetric.WithLabelValues(strconv.FormatBool(jobPaused1), storePlugin1, targetPlugin1).Desc())))
		})

		It("returns a jobs_scrapes_total metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(jobsScrapesTotalMetric.Desc())))
		})

		It("returns a jobs_scrape_errors_total metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(jobsScrapeErrorsTotalMetric.Desc())))
		})

		It("returns a last_jobs_scrape_error metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(lastJobsScrapeErrorMetric.Desc())))
		})

		It("returns a last_jobs_scrape_timestamp metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(lastJobsScrapeTimestampMetric.Desc())))
		})

		It("returns a last_jobs_scrape_duration_seconds metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(lastJobsScrapeDurationSecondsMetric.Desc())))
		})
	})

	Describe("Collect", func() {
		var (
			statusCode   int
			jobsResponse []api.Job
			metrics      chan prometheus.Metric
		)

		BeforeEach(func() {
			statusCode = http.StatusOK
			jobsResponse = []api.Job{
				api.Job{
					Paused:       jobPaused1,
					StorePlugin:  storePlugin1,
					TargetPlugin: targetPlugin1,
				},
				api.Job{
					Paused:       jobPaused2,
					StorePlugin:  storePlugin1,
					TargetPlugin: targetPlugin2,
				},
				api.Job{
					Paused:       jobPaused1,
					StorePlugin:  storePlugin2,
					TargetPlugin: targetPlugin1,
				},
				api.Job{
					Paused:       jobPaused2,
					StorePlugin:  storePlugin2,
					TargetPlugin: targetPlugin2,
				},
			}
			metrics = make(chan prometheus.Metric)
		})

		JustBeforeEach(func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/v1/jobs"),
					ghttp.VerifyBasicAuth(username, password),
					ghttp.RespondWithJSONEncodedPtr(&statusCode, &jobsResponse),
				),
			)
			go jobsCollector.Collect(metrics)
		})

		It("returns a jobs_total metric for job paused 1, store plugin 1, target plugin 1", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(jobsTotalMetric.WithLabelValues(strconv.FormatBool(jobPaused1), storePlugin1, targetPlugin1))))
		})

		It("returns a jobs_total metric for job paused 1, store plugin 1, target plugin 2", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(jobsTotalMetric.WithLabelValues(strconv.FormatBool(jobPaused2), storePlugin1, targetPlugin2))))
		})

		It("returns a jobs_total metric for job paused 1, store plugin 2, target plugin 1", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(jobsTotalMetric.WithLabelValues(strconv.FormatBool(jobPaused1), storePlugin2, targetPlugin1))))
		})

		It("returns a jobs_total metric for job paused 1, store plugin 2, target plugin 2", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(jobsTotalMetric.WithLabelValues(strconv.FormatBool(jobPaused2), storePlugin2, targetPlugin2))))
		})

		It("returns a jobs_scrapes_total metric", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(jobsScrapesTotalMetric)))
		})

		It("returns a jobs_scrape_errors_total metric", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(jobsScrapeErrorsTotalMetric)))
		})

		It("returns a last_jobs_scrape_error metric", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(lastJobsScrapeErrorMetric)))
		})

		Context("when it fails to list the jobs", func() {
			BeforeEach(func() {
				statusCode = http.StatusInternalServerError
				jobsScrapeErrorsTotalMetric.Inc()
				lastJobsScrapeErrorMetric.Set(1)
			})

			It("returns a jobs_scrape_errors_total metric", func() {
				Eventually(metrics).Should(Receive(PrometheusMetric(jobsScrapeErrorsTotalMetric)))
			})

			It("returns a last_jobs_scrape_error metric", func() {
				Eventually(metrics).Should(Receive(PrometheusMetric(lastJobsScrapeErrorMetric)))
			})
		})
	})
})
