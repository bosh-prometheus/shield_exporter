package collectors

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"github.com/starkandwayne/shield/api"
)

type ArchivesCollector struct {
	namespace                               string
	environment                             string
	backendName                             string
	archivesTotalMetric                     *prometheus.GaugeVec
	archivesScrapesTotalMetric              prometheus.Counter
	archivesScrapeErrorsTotalMetric         prometheus.Counter
	lastArchivesScrapeErrorMetric           prometheus.Gauge
	lastArchivesScrapeTimestampMetric       prometheus.Gauge
	lastArchivesScrapeDurationSecondsMetric prometheus.Gauge
}

func NewArchivesCollector(
	namespace string,
	environment string,
	backendName string,
) *ArchivesCollector {
	archivesTotalMetric := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "archives",
			Name:        "total",
			Help:        "Labeled total number of Shield Archives.",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
		[]string{"archive_status", "store_plugin", "target_plugin"},
	)

	archivesScrapesTotalMetric := prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace:   namespace,
			Subsystem:   "archives",
			Name:        "scrapes_total",
			Help:        "Total number of scrapes for Shield Archives.",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
	)

	archivesScrapeErrorsTotalMetric := prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace:   namespace,
			Subsystem:   "archives",
			Name:        "scrape_errors_total",
			Help:        "Total number of scrape errors of Shield Archives.",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
	)

	lastArchivesScrapeErrorMetric := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "",
			Name:        "last_archives_scrape_error",
			Help:        "Whether the last scrape of Archive metrics from Shield resulted in an error (1 for error, 0 for success).",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
	)

	lastArchivesScrapeTimestampMetric := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "",
			Name:        "last_archives_scrape_timestamp",
			Help:        "Number of seconds since 1970 since last scrape of Archive metrics from Shield.",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
	)

	lastArchivesScrapeDurationSecondsMetric := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "",
			Name:        "last_archives_scrape_duration_seconds",
			Help:        "Duration of the last scrape of Archive metrics from Shield.",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
	)

	return &ArchivesCollector{
		namespace:                               namespace,
		environment:                             environment,
		backendName:                             backendName,
		archivesTotalMetric:                     archivesTotalMetric,
		archivesScrapesTotalMetric:              archivesScrapesTotalMetric,
		archivesScrapeErrorsTotalMetric:         archivesScrapeErrorsTotalMetric,
		lastArchivesScrapeErrorMetric:           lastArchivesScrapeErrorMetric,
		lastArchivesScrapeTimestampMetric:       lastArchivesScrapeTimestampMetric,
		lastArchivesScrapeDurationSecondsMetric: lastArchivesScrapeDurationSecondsMetric,
	}
}

func (c ArchivesCollector) Collect(ch chan<- prometheus.Metric) {
	var begun = time.Now()

	errorMetric := float64(0)
	if err := c.reportTargetsMetrics(ch); err != nil {
		errorMetric = float64(1)
		c.archivesScrapeErrorsTotalMetric.Inc()
	}
	c.archivesScrapeErrorsTotalMetric.Collect(ch)

	c.archivesScrapesTotalMetric.Inc()
	c.archivesScrapesTotalMetric.Collect(ch)

	c.lastArchivesScrapeErrorMetric.Set(errorMetric)
	c.lastArchivesScrapeErrorMetric.Collect(ch)

	c.lastArchivesScrapeTimestampMetric.Set(float64(time.Now().Unix()))
	c.lastArchivesScrapeTimestampMetric.Collect(ch)

	c.lastArchivesScrapeDurationSecondsMetric.Set(time.Since(begun).Seconds())
	c.lastArchivesScrapeDurationSecondsMetric.Collect(ch)
}

func (c ArchivesCollector) Describe(ch chan<- *prometheus.Desc) {
	c.archivesTotalMetric.Describe(ch)
	c.archivesScrapesTotalMetric.Describe(ch)
	c.archivesScrapeErrorsTotalMetric.Describe(ch)
	c.lastArchivesScrapeErrorMetric.Describe(ch)
	c.lastArchivesScrapeTimestampMetric.Describe(ch)
	c.lastArchivesScrapeDurationSecondsMetric.Describe(ch)
}

func (c ArchivesCollector) reportTargetsMetrics(ch chan<- prometheus.Metric) error {
	c.archivesTotalMetric.Reset()

	archives, err := api.GetArchives(api.ArchiveFilter{})
	if err != nil {
		log.Errorf("Error while listing archives: %v", err)
		return err
	}

	for _, archive := range archives {
		c.archivesTotalMetric.WithLabelValues(archive.Status, archive.StorePlugin, archive.TargetPlugin).Inc()
	}

	c.archivesTotalMetric.Collect(ch)

	return nil
}
