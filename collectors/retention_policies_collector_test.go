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

var _ = Describe("RetentionPoliciesCollectors", func() {
	var (
		err    error
		server *ghttp.Server

		namespace   = "test_namespace"
		environment = "test_environment"
		backendName = "test_backend"

		username = "fake_username"
		password = "fake_password"

		retentionPoliciesTotalMetric                     prometheus.Gauge
		retentionPoliciesScrapesTotalMetric              prometheus.Counter
		retentionPoliciesScrapeErrorsTotalMetric         prometheus.Counter
		lastRetentionPoliciesScrapeErrorMetric           prometheus.Gauge
		lastRetentionPoliciesScrapeTimestampMetric       prometheus.Gauge
		lastRetentionPoliciesScrapeDurationSecondsMetric prometheus.Gauge

		retentionPoliciesCollector *RetentionPoliciesCollector
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

		retentionPoliciesTotalMetric = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   "retention_policies",
				Name:        "total",
				Help:        "Total number of Shield Retention Policies.",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
		)
		retentionPoliciesTotalMetric.Set(2)

		retentionPoliciesScrapesTotalMetric = prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace:   namespace,
				Subsystem:   "retention_policies",
				Name:        "scrapes_total",
				Help:        "Total number of scrapes for Shield Retention Policies.",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
		)
		retentionPoliciesScrapesTotalMetric.Inc()

		retentionPoliciesScrapeErrorsTotalMetric = prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace:   namespace,
				Subsystem:   "retention_policies",
				Name:        "scrape_errors_total",
				Help:        "Total number of scrape errors of Shield Retention Policies.",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
		)

		lastRetentionPoliciesScrapeErrorMetric = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   "",
				Name:        "last_retention_policies_scrape_error",
				Help:        "Whether the last scrape of Retention Policies metrics from Shield resulted in an error (1 for error, 0 for success).",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
		)

		lastRetentionPoliciesScrapeTimestampMetric = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   "",
				Name:        "last_retention_policies_scrape_timestamp",
				Help:        "Number of seconds since 1970 since last scrape of Retention Policies metrics from Shield.",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
		)

		lastRetentionPoliciesScrapeDurationSecondsMetric = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   "",
				Name:        "last_retention_policies_scrape_duration_seconds",
				Help:        "Duration of the last scrape of Retention Policies metrics from Shield.",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
		)
	})

	JustBeforeEach(func() {
		retentionPoliciesCollector = NewRetentionPoliciesCollector(namespace, environment, backendName)
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
			go retentionPoliciesCollector.Describe(descriptions)
		})

		It("returns a retention_policies_total metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(retentionPoliciesTotalMetric.Desc())))
		})

		It("returns a retention_policies_scrapes_total metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(retentionPoliciesScrapesTotalMetric.Desc())))
		})

		It("returns a retention_policies_scrape_errors_total metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(retentionPoliciesScrapeErrorsTotalMetric.Desc())))
		})

		It("returns a last_retention_policies_scrape_error metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(lastRetentionPoliciesScrapeErrorMetric.Desc())))
		})

		It("returns a last_retention_policies_scrape_timestamp metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(lastRetentionPoliciesScrapeTimestampMetric.Desc())))
		})

		It("returns a last_retention_policies_scrape_duration_seconds metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(lastRetentionPoliciesScrapeDurationSecondsMetric.Desc())))
		})
	})

	Describe("Collect", func() {
		var (
			statusCode                int
			retentionPoliciesResponse []api.RetentionPolicy
			metrics                   chan prometheus.Metric
		)

		BeforeEach(func() {
			statusCode = http.StatusOK
			retentionPoliciesResponse = []api.RetentionPolicy{
				api.RetentionPolicy{
					Name: "fake_retention_policiy_1",
				},
				api.RetentionPolicy{
					Name: "fake_retention_policiy_2",
				},
			}
			metrics = make(chan prometheus.Metric)
		})

		JustBeforeEach(func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/v1/retention"),
					ghttp.VerifyBasicAuth(username, password),
					ghttp.RespondWithJSONEncodedPtr(&statusCode, &retentionPoliciesResponse),
				),
			)
			go retentionPoliciesCollector.Collect(metrics)
		})

		It("returns a retention_policies_total metric", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(retentionPoliciesTotalMetric)))
		})

		It("returns a retention_policiess_scrapes_total metric", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(retentionPoliciesScrapesTotalMetric)))
		})

		It("returns a retention_policies_scrape_errors_total metric", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(retentionPoliciesScrapeErrorsTotalMetric)))
		})

		It("returns a last_retention_policies_scrape_error metric", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(lastRetentionPoliciesScrapeErrorMetric)))
		})

		Context("when it fails to list the retention policies", func() {
			BeforeEach(func() {
				statusCode = http.StatusInternalServerError
				retentionPoliciesScrapeErrorsTotalMetric.Inc()
				lastRetentionPoliciesScrapeErrorMetric.Set(1)
			})

			It("returns a retention_policies_scrape_errors_total metric", func() {
				Eventually(metrics).Should(Receive(PrometheusMetric(retentionPoliciesScrapeErrorsTotalMetric)))
			})

			It("returns a last_retention_policies_scrape_error metric", func() {
				Eventually(metrics).Should(Receive(PrometheusMetric(lastRetentionPoliciesScrapeErrorMetric)))
			})
		})
	})
})
