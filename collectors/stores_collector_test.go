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

var _ = Describe("StoresCollectors", func() {
	var (
		err    error
		server *ghttp.Server

		namespace   = "test_namespace"
		environment = "test_environment"
		backendName = "test_backend"

		username = "fake_username"
		password = "fake_password"

		storePlugin1 = "store_plugin_1"
		storePlugin2 = "store_plugin_2"

		storesTotalMetric                     *prometheus.GaugeVec
		storesScrapesTotalMetric              prometheus.Counter
		storesScrapeErrorsTotalMetric         prometheus.Counter
		lastStoresScrapeErrorMetric           prometheus.Gauge
		lastStoresScrapeTimestampMetric       prometheus.Gauge
		lastStoresScrapeDurationSecondsMetric prometheus.Gauge

		storesCollector *StoresCollector
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

		storesTotalMetric = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   "stores",
				Name:        "total",
				Help:        "Labeled total number of Shield Stores.",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
			[]string{"store_plugin"},
		)
		storesTotalMetric.WithLabelValues(storePlugin1).Set(2)
		storesTotalMetric.WithLabelValues(storePlugin2).Set(1)

		storesScrapesTotalMetric = prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace:   namespace,
				Subsystem:   "stores",
				Name:        "scrapes_total",
				Help:        "Total number of scrapes for Shield Stores.",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
		)
		storesScrapesTotalMetric.Inc()

		storesScrapeErrorsTotalMetric = prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace:   namespace,
				Subsystem:   "stores",
				Name:        "scrape_errors_total",
				Help:        "Total number of scrape errors of Shield Stores.",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
		)

		lastStoresScrapeErrorMetric = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   "",
				Name:        "last_stores_scrape_error",
				Help:        "Whether the last scrape of Store metrics from Shield resulted in an error (1 for error, 0 for success).",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
		)

		lastStoresScrapeTimestampMetric = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   "",
				Name:        "last_stores_scrape_timestamp",
				Help:        "Number of seconds since 1970 since last scrape of Store metrics from Shield.",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
		)

		lastStoresScrapeDurationSecondsMetric = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   "",
				Name:        "last_stores_scrape_duration_seconds",
				Help:        "Duration of the last scrape of Store metrics from Shield.",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
		)
	})

	JustBeforeEach(func() {
		storesCollector = NewStoresCollector(namespace, environment, backendName)
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
			go storesCollector.Describe(descriptions)
		})

		It("returns a stores_total metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(storesTotalMetric.WithLabelValues(storePlugin1).Desc())))
		})

		It("returns a stores_scrapes_total metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(storesScrapesTotalMetric.Desc())))
		})

		It("returns a stores_scrape_errors_total metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(storesScrapeErrorsTotalMetric.Desc())))
		})

		It("returns a last_stores_scrape_error metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(lastStoresScrapeErrorMetric.Desc())))
		})

		It("returns a last_stores_scrape_timestamp metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(lastStoresScrapeTimestampMetric.Desc())))
		})

		It("returns a last_stores_scrape_duration_seconds metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(lastStoresScrapeDurationSecondsMetric.Desc())))
		})
	})

	Describe("Collect", func() {
		var (
			statusCode     int
			storesResponse []api.Store
			metrics        chan prometheus.Metric
		)

		BeforeEach(func() {
			statusCode = http.StatusOK
			storesResponse = []api.Store{
				api.Store{
					Plugin: storePlugin1,
				},
				api.Store{
					Plugin: storePlugin1,
				},
				api.Store{
					Plugin: storePlugin2,
				},
			}
			metrics = make(chan prometheus.Metric)
		})

		JustBeforeEach(func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/v1/stores"),
					ghttp.VerifyBasicAuth(username, password),
					ghttp.RespondWithJSONEncodedPtr(&statusCode, &storesResponse),
				),
			)
			go storesCollector.Collect(metrics)
		})

		It("returns a stores_total metric for store plugin 1", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(storesTotalMetric.WithLabelValues(storePlugin1))))
		})

		It("returns a stores_total metric for store plugin 2", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(storesTotalMetric.WithLabelValues(storePlugin2))))
		})

		It("returns a stores_scrapes_total metric", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(storesScrapesTotalMetric)))
		})

		It("returns a stores_scrape_errors_total metric", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(storesScrapeErrorsTotalMetric)))
		})

		It("returns a last_stores_scrape_error metric", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(lastStoresScrapeErrorMetric)))
		})

		Context("when it fails to list the stores", func() {
			BeforeEach(func() {
				statusCode = http.StatusInternalServerError
				storesScrapeErrorsTotalMetric.Inc()
				lastStoresScrapeErrorMetric.Set(1)
			})

			It("returns a stores_scrape_errors_total metric", func() {
				Eventually(metrics).Should(Receive(PrometheusMetric(storesScrapeErrorsTotalMetric)))
			})

			It("returns a last_stores_scrape_error metric", func() {
				Eventually(metrics).Should(Receive(PrometheusMetric(lastStoresScrapeErrorMetric)))
			})
		})
	})
})
