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

	. "github.com/bosh-prometheus/shield_exporter/collectors"
	. "github.com/bosh-prometheus/shield_exporter/utils/test_matchers"
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

		jobName1      = "fake_job_1"
		jobName2      = "fake_job_2"
		jobStatus1    = "done"
		jobStatus2    = "failed"
		jobPaused1    = true
		jobPaused2    = false
		storePlugin1  = "store_plugin_1"
		storePlugin2  = "store_plugin_2"
		targetPlugin1 = "target_plugin_1"
		targetPlugin2 = "target_plugin_2"
		lastRun1      = 1
		lastRun2      = 2
		nextRun1      = 3
		nextRun2      = 4

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

		jobLastRunMetric = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   "job",
				Name:        "last_run",
				Help:        "Number of seconds since 1970 since last run of a Shield Job.",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
			[]string{"job_name"},
		)
		jobLastRunMetric.WithLabelValues(jobName1).Set(float64(lastRun1))
		jobLastRunMetric.WithLabelValues(jobName2).Set(float64(lastRun2))

		jobNextRunMetric = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   "job",
				Name:        "next_run",
				Help:        "Number of seconds since 1970 until next run of a Shield Job.",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
			[]string{"job_name"},
		)
		jobNextRunMetric.WithLabelValues(jobName1).Set(float64(nextRun1))
		jobNextRunMetric.WithLabelValues(jobName2).Set(float64(nextRun2))

		jobStatusMetric = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   "job",
				Name:        "status",
				Help:        "Shield Job status (0 for unknow, 1 for pending, 2 for running, 3 for canceled, 4 for failed, 5 for done).",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
			[]string{"job_name"},
		)
		jobStatusMetric.WithLabelValues(jobName1).Set(float64(5))
		jobStatusMetric.WithLabelValues(jobName2).Set(float64(4))

		jobPausedMetric = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Subsystem:   "job",
				Name:        "paused",
				Help:        "Shield Job pause status (1 for paused, 0 for unpaused).",
				ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
			},
			[]string{"job_name"},
		)
		jobPausedMetric.WithLabelValues(jobName1).Set(float64(1))
		jobPausedMetric.WithLabelValues(jobName2).Set(float64(0))

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

		It("returns a job_last_run metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(jobLastRunMetric.WithLabelValues(jobName1).Desc())))
		})

		It("returns a job_next_run metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(jobNextRunMetric.WithLabelValues(jobName1).Desc())))
		})

		It("returns a job_status metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(jobStatusMetric.WithLabelValues(jobName1).Desc())))
		})

		It("returns a job_paused metric description", func() {
			Eventually(descriptions).Should(Receive(Equal(jobPausedMetric.WithLabelValues(jobName1).Desc())))
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
			jobsStatusCode       int
			statusJobsStatusCode int
			jobsResponse         []api.Job
			jobsStatusResponse   api.JobsStatus
			metrics              chan prometheus.Metric
		)

		BeforeEach(func() {
			jobsStatusCode = http.StatusOK
			statusJobsStatusCode = http.StatusOK
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
			jobsStatusResponse = api.JobsStatus{
				jobName1: api.JobHealth{
					Name:    jobName1,
					LastRun: int64(lastRun1),
					NextRun: int64(nextRun1),
					Paused:  jobPaused1,
					Status:  jobStatus1,
				},
				jobName2: api.JobHealth{
					Name:    jobName2,
					LastRun: int64(lastRun2),
					NextRun: int64(nextRun2),
					Paused:  jobPaused2,
					Status:  jobStatus2,
				},
			}
			metrics = make(chan prometheus.Metric)
		})

		JustBeforeEach(func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/v1/jobs"),
					ghttp.VerifyBasicAuth(username, password),
					ghttp.RespondWithJSONEncodedPtr(&jobsStatusCode, &jobsResponse),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/v1/ping"),
					ghttp.RespondWith(http.StatusOK, "{}"),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/v1/status/jobs"),
					ghttp.VerifyBasicAuth(username, password),
					ghttp.RespondWithJSONEncodedPtr(&statusJobsStatusCode, &jobsStatusResponse),
				),
			)
			go jobsCollector.Collect(metrics)
		})

		It("returns a job_last_run metric for job name 1", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(jobLastRunMetric.WithLabelValues(jobName1))))
		})

		It("returns a job_last_run metric for job name 2", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(jobLastRunMetric.WithLabelValues(jobName2))))
		})

		It("returns a job_next_run metric job name 1", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(jobNextRunMetric.WithLabelValues(jobName1))))
		})

		It("returns a job_next_run metric job name 2", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(jobNextRunMetric.WithLabelValues(jobName2))))
		})

		It("returns a job_status metric job name 1", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(jobStatusMetric.WithLabelValues(jobName1))))
		})

		It("returns a job_status metric job name 2", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(jobStatusMetric.WithLabelValues(jobName2))))
		})

		It("returns a job_paused metric job name 1", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(jobPausedMetric.WithLabelValues(jobName1))))
		})

		It("returns a job_paused metric job name 2", func() {
			Eventually(metrics).Should(Receive(PrometheusMetric(jobPausedMetric.WithLabelValues(jobName2))))
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
				jobsStatusCode = http.StatusInternalServerError
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

		Context("when it fails to get the job's status", func() {
			BeforeEach(func() {
				statusJobsStatusCode = http.StatusInternalServerError
				jobsScrapeErrorsTotalMetric.Inc()
				lastJobsScrapeErrorMetric.Set(1)
			})

			It("does not returns a job_last_run metric for job name 1", func() {
				Consistently(metrics).ShouldNot(Receive(PrometheusMetric(jobLastRunMetric.WithLabelValues(jobName1))))
			})

			It("does not returns a job_next_run metric job name 1", func() {
				Consistently(metrics).ShouldNot(Receive(PrometheusMetric(jobNextRunMetric.WithLabelValues(jobName1))))
			})

			It("does not returns a job_status metric for job name 1", func() {
				Consistently(metrics).ShouldNot(Receive(PrometheusMetric(jobStatusMetric.WithLabelValues(jobName1))))
			})

			It("does not returns a job_paused metric for job name 1", func() {
				Consistently(metrics).ShouldNot(Receive(PrometheusMetric(jobPausedMetric.WithLabelValues(jobName1))))
			})

			It("returns a jobs_total metric for job paused 1, store plugin 1, target plugin 1", func() {
				Eventually(metrics).Should(Receive(PrometheusMetric(jobsTotalMetric.WithLabelValues(strconv.FormatBool(jobPaused1), storePlugin1, targetPlugin1))))
			})

			It("returns a jobs_scrape_errors_total metric", func() {
				Eventually(metrics).Should(Receive(PrometheusMetric(jobsScrapeErrorsTotalMetric)))
			})

			It("returns a last_jobs_scrape_error metric", func() {
				Eventually(metrics).Should(Receive(PrometheusMetric(lastJobsScrapeErrorMetric)))
			})
		})

		Context("when job's status API is not implemented", func() {
			BeforeEach(func() {
				statusJobsStatusCode = http.StatusNotImplemented
			})

			It("does not returns a job_last_run metric for job name 1", func() {
				Consistently(metrics).ShouldNot(Receive(PrometheusMetric(jobLastRunMetric.WithLabelValues(jobName1))))
			})

			It("does not returns a job_next_run metric job name 1", func() {
				Consistently(metrics).ShouldNot(Receive(PrometheusMetric(jobNextRunMetric.WithLabelValues(jobName1))))
			})

			It("does not returns a job_status metric for job name 1", func() {
				Consistently(metrics).ShouldNot(Receive(PrometheusMetric(jobStatusMetric.WithLabelValues(jobName1))))
			})

			It("does not returns a job_paused metric for job name 1", func() {
				Consistently(metrics).ShouldNot(Receive(PrometheusMetric(jobPausedMetric.WithLabelValues(jobName1))))
			})

			It("returns a jobs_total metric for job paused 1, store plugin 1, target plugin 1", func() {
				Eventually(metrics).Should(Receive(PrometheusMetric(jobsTotalMetric.WithLabelValues(strconv.FormatBool(jobPaused1), storePlugin1, targetPlugin1))))
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
