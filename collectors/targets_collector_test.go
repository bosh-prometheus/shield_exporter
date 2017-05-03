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

var _ = Describe("TargetsCollectors", func() {
	var (
		err    error
		server *ghttp.Server

		namespace   = "test_namespace"
		environment = "test_environment"
		backendName = "test_backend"

		username = "fake_username"
		password = "fake_password"

		targetPlugin1 = "target_plugin_1"
		targetPlugin2 = "target_plugin_2"

		targetsTotalMetric                     *prometheus.GaugeVec
		targetsScrapesTotalMetric              prometheus.Counter
		targetsScrapeErrorsTotalMetric         prometheus.Counter
		lastTargetsScrapeErrorMetric           prometheus.Gauge
		lastTargetsScrapeTimestampMetric       prometheus.Gauge
		lastTargetsScrapeDurationSecondsMetric prometheus.Gauge

		targetsCollector *TargetsCollector
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

		targetsTotalMetric = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   "targets",
				Name:        "total",
				Help:        "Labeled total number of Shield Targets.",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
			[]string{"target_plugin"},
		)
		targetsTotalMetric.WithLabelValues(targetPlugin1).Set(2)
		targetsTotalMetric.WithLabelValues(targetPlugin2).Set(1)

		targetsScrapesTotalMetric = prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace:   namespace,
				Subsystem:   "targets",
				Name:        "scrapes_total",
				Help:        "Total number of scrapes for Shield Targets.",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
		)
		targetsScrapesTotalMetric.Inc()

		targetsScrapeErrorsTotalMetric = prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace:   namespace,
				Subsystem:   "targets",
				Name:        "scrape_errorstotal",
				Help:        "Total number of scrape errors of Shield Targets.",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
		)

		lastTargetsScrapeErrorMetric = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   "",
				Name:        "last_targets_scrape_error",
				Help:        "Whether the last scrape of Target metrics from Shield resulted in an error (1 for error, 0 for success).",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
		)

		lastTargetsScrapeTimestampMetric = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   "",
				Name:        "last_targets_scrape_timestamp",
				Help:        "Number of seconds since 1970 since last scrape of Target metrics from Shield.",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
		)

		lastTargetsScrapeDurationSecondsMetric = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   "",
				Name:        "last_targets_scrape_duration_seconds",
				Help:        "Duration of the last scrape of Target metrics from Shield.",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
		)
	})

	JustBeforeEach(func() {
		targetsCollector = NewTargetsCollector(namespace, environment, backendName)
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
			go targetsCollector.Describe(descriptions)
		})

		It("returns a targets_total metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(targetsTotalMetric.WithLabelValues(targetPlugin1).Desc())))
		})

		It("returns a targets_scrapes_total metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(targetsScrapesTotalMetric.Desc())))
		})

		It("returns a targets_scrape_errors_total metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(targetsScrapeErrorsTotalMetric.Desc())))
		})

		It("returns a last_targets_scrape_error metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(lastTargetsScrapeErrorMetric.Desc())))
		})

		It("returns a last_targets_scrape_timestamp metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(lastTargetsScrapeTimestampMetric.Desc())))
		})

		It("returns a last_targets_scrape_duration_seconds metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(lastTargetsScrapeDurationSecondsMetric.Desc())))
		})
	})

	Describe("Collect", func() {
		var (
			statusCode      int
			targetsResponse []api.Target
			metrics         chan prometheus.Metric
		)

		BeforeEach(func() {
			statusCode = http.StatusOK
			targetsResponse = []api.Target{
				api.Target{
					Plugin: targetPlugin1,
				},
				api.Target{
					Plugin: targetPlugin1,
				},
				api.Target{
					Plugin: targetPlugin2,
				},
			}
			metrics = make(chan prometheus.Metric)
		})

		JustBeforeEach(func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/v1/targets"),
					ghttp.VerifyBasicAuth(username, password),
					ghttp.RespondWithJSONEncodedPtr(&statusCode, &targetsResponse),
				),
			)
			go targetsCollector.Collect(metrics)
		})

		It("returns a targets_total metric for target plugin 1", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(targetsTotalMetric.WithLabelValues(targetPlugin1))))
		})

		It("returns a targets_total metric for target plugin 2", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(targetsTotalMetric.WithLabelValues(targetPlugin2))))
		})

		It("returns a targets_scrapes_total metric", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(targetsScrapesTotalMetric)))
		})

		It("returns a targets_scrape_errors_total metric", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(targetsScrapeErrorsTotalMetric)))
		})

		It("returns a last_targets_scrape_error metric", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(lastTargetsScrapeErrorMetric)))
		})

		Context("when it fails to list the targets", func() {
			BeforeEach(func() {
				statusCode = http.StatusInternalServerError
				targetsScrapeErrorsTotalMetric.Inc()
				lastTargetsScrapeErrorMetric.Set(1)
			})

			It("returns a targets_scrape_errors_total metric", func() {
				Eventually(metrics).Should(Receive(PrometheusMetric(targetsScrapeErrorsTotalMetric)))
			})

			It("returns a last_targets_scrape_error metric", func() {
				Eventually(metrics).Should(Receive(PrometheusMetric(lastTargetsScrapeErrorMetric)))
			})
		})
	})
})
