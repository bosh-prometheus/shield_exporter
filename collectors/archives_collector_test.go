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

var _ = Describe("ArchivesCollectors", func() {
	var (
		err    error
		server *ghttp.Server

		namespace   = "test_namespace"
		environment = "test_environment"
		backendName = "test_backend"

		username = "fake_username"
		password = "fake_password"

		archiveStatus1 = "done"
		archiveStatus2 = "purged"
		storePlugin1   = "store_plugin_1"
		storePlugin2   = "store_plugin_2"
		targetPlugin1  = "target_plugin_1"
		targetPlugin2  = "target_plugin_2"

		archivesTotalMetric                     *prometheus.GaugeVec
		archivesScrapesTotalMetric              prometheus.Counter
		archivesScrapeErrorsTotalMetric         prometheus.Counter
		lastArchivesScrapeErrorMetric           prometheus.Gauge
		lastArchivesScrapeTimestampMetric       prometheus.Gauge
		lastArchivesScrapeDurationSecondsMetric prometheus.Gauge

		archivesCollector *ArchivesCollector
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

		archivesTotalMetric = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   "archives",
				Name:        "total",
				Help:        "Labeled total number of Shield Archives.",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
			[]string{"archive_status", "store_plugin", "target_plugin"},
		)
		archivesTotalMetric.WithLabelValues(archiveStatus1, storePlugin1, targetPlugin1).Set(1)
		archivesTotalMetric.WithLabelValues(archiveStatus2, storePlugin1, targetPlugin2).Set(1)
		archivesTotalMetric.WithLabelValues(archiveStatus1, storePlugin2, targetPlugin1).Set(1)
		archivesTotalMetric.WithLabelValues(archiveStatus2, storePlugin2, targetPlugin2).Set(1)

		archivesScrapesTotalMetric = prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace:   namespace,
				Subsystem:   "archives",
				Name:        "scrapes_total",
				Help:        "Total number of scrapes for Shield Archives.",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
		)
		archivesScrapesTotalMetric.Inc()

		archivesScrapeErrorsTotalMetric = prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace:   namespace,
				Subsystem:   "archives",
				Name:        "scrape_errors_total",
				Help:        "Total number of scrape errors of Shield Archives.",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
		)

		lastArchivesScrapeErrorMetric = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   "",
				Name:        "last_archives_scrape_error",
				Help:        "Whether the last scrape of Archive metrics from Shield resulted in an error (1 for error, 0 for success).",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
		)

		lastArchivesScrapeTimestampMetric = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   "",
				Name:        "last_archives_scrape_timestamp",
				Help:        "Number of seconds since 1970 since last scrape of Archive metrics from Shield.",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
		)

		lastArchivesScrapeDurationSecondsMetric = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   "",
				Name:        "last_archives_scrape_duration_seconds",
				Help:        "Duration of the last scrape of Archive metrics from Shield.",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
		)
	})

	JustBeforeEach(func() {
		archivesCollector = NewArchivesCollector(namespace, environment, backendName)
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
			go archivesCollector.Describe(descriptions)
		})

		It("returns a archives_total metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(archivesTotalMetric.WithLabelValues(archiveStatus1, storePlugin1, targetPlugin1).Desc())))
		})

		It("returns a archives_scrapes_total metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(archivesScrapesTotalMetric.Desc())))
		})

		It("returns a archives_scrape_errors_total metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(archivesScrapeErrorsTotalMetric.Desc())))
		})

		It("returns a last_archives_scrape_error metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(lastArchivesScrapeErrorMetric.Desc())))
		})

		It("returns a last_archives_scrape_timestamp metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(lastArchivesScrapeTimestampMetric.Desc())))
		})

		It("returns a last_archives_scrape_duration_seconds metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(lastArchivesScrapeDurationSecondsMetric.Desc())))
		})
	})

	Describe("Collect", func() {
		var (
			statusCode       int
			archivesResponse []api.Archive
			metrics          chan prometheus.Metric
		)

		BeforeEach(func() {
			statusCode = http.StatusOK
			archivesResponse = []api.Archive{
				api.Archive{
					Status:       archiveStatus1,
					StorePlugin:  storePlugin1,
					TargetPlugin: targetPlugin1,
				},
				api.Archive{
					Status:       archiveStatus2,
					StorePlugin:  storePlugin1,
					TargetPlugin: targetPlugin2,
				},
				api.Archive{
					Status:       archiveStatus1,
					StorePlugin:  storePlugin2,
					TargetPlugin: targetPlugin1,
				},
				api.Archive{
					Status:       archiveStatus2,
					StorePlugin:  storePlugin2,
					TargetPlugin: targetPlugin2,
				},
			}
			metrics = make(chan prometheus.Metric)
		})

		JustBeforeEach(func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/v1/archives"),
					ghttp.VerifyBasicAuth(username, password),
					ghttp.RespondWithJSONEncodedPtr(&statusCode, &archivesResponse),
				),
			)
			go archivesCollector.Collect(metrics)
		})

		It("returns a archives_total metric for archive status 1, store plugin 1, target plugin 1", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(archivesTotalMetric.WithLabelValues(archiveStatus1, storePlugin1, targetPlugin1))))
		})

		It("returns a archives_total metric for archive status 1, store plugin 1, target plugin 2", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(archivesTotalMetric.WithLabelValues(archiveStatus2, storePlugin1, targetPlugin2))))
		})

		It("returns a archives_total metric for archive status 1, store plugin 2, target plugin 1", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(archivesTotalMetric.WithLabelValues(archiveStatus1, storePlugin2, targetPlugin1))))
		})

		It("returns a archives_total metric for archive status 1, store plugin 2, target plugin 2", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(archivesTotalMetric.WithLabelValues(archiveStatus2, storePlugin2, targetPlugin2))))
		})

		It("returns a archives_scrapes_total metric", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(archivesScrapesTotalMetric)))
		})

		It("returns a archives_scrape_errors_total metric", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(archivesScrapeErrorsTotalMetric)))
		})

		It("returns a last_archives_scrape_error metric", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(lastArchivesScrapeErrorMetric)))
		})

		Context("when it fails to list the archives", func() {
			BeforeEach(func() {
				statusCode = http.StatusInternalServerError
				archivesScrapeErrorsTotalMetric.Inc()
				lastArchivesScrapeErrorMetric.Set(1)
			})

			It("returns a archives_scrape_errors_total metric", func() {
				Eventually(metrics).Should(Receive(PrometheusMetric(archivesScrapeErrorsTotalMetric)))
			})

			It("returns a last_archives_scrape_error metric", func() {
				Eventually(metrics).Should(Receive(PrometheusMetric(lastArchivesScrapeErrorMetric)))
			})
		})
	})
})
