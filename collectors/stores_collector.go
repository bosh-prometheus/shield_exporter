package collectors

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"github.com/starkandwayne/shield/api"
)

type StoresCollector struct {
	namespace                             string
	environment                           string
	backendName                           string
	storesTotalMetric                     *prometheus.GaugeVec
	storesScrapesTotalMetric              prometheus.Counter
	storesScrapeErrorsTotalMetric         prometheus.Counter
	lastStoresScrapeErrorMetric           prometheus.Gauge
	lastStoresScrapeTimestampMetric       prometheus.Gauge
	lastStoresScrapeDurationSecondsMetric prometheus.Gauge
}

func NewStoresCollector(
	namespace string,
	environment string,
	backendName string,
) *StoresCollector {
	storesTotalMetric := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "stores",
			Name:        "total",
			Help:        "Labeled total number of Shield Stores.",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
		[]string{"store_plugin"},
	)

	storesScrapesTotalMetric := prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace:   namespace,
			Subsystem:   "stores",
			Name:        "scrapes_total",
			Help:        "Total number of scrapes for Shield Stores.",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
	)

	storesScrapeErrorsTotalMetric := prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace:   namespace,
			Subsystem:   "stores",
			Name:        "scrape_errors_total",
			Help:        "Total number of scrape errors of Shield Stores.",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
	)

	lastStoresScrapeErrorMetric := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "",
			Name:        "last_stores_scrape_error",
			Help:        "Whether the last scrape of Store metrics from Shield resulted in an error (1 for error, 0 for success).",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
	)

	lastStoresScrapeTimestampMetric := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "",
			Name:        "last_stores_scrape_timestamp",
			Help:        "Number of seconds since 1970 since last scrape of Store metrics from Shield.",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
	)

	lastStoresScrapeDurationSecondsMetric := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   "",
			Name:        "last_stores_scrape_duration_seconds",
			Help:        "Duration of the last scrape of Store metrics from Shield.",
			ConstLabels: prometheus.Labels{"environment": environment, "backend_name": backendName},
		},
	)

	return &StoresCollector{
		namespace:                             namespace,
		environment:                           environment,
		backendName:                           backendName,
		storesTotalMetric:                     storesTotalMetric,
		storesScrapesTotalMetric:              storesScrapesTotalMetric,
		storesScrapeErrorsTotalMetric:         storesScrapeErrorsTotalMetric,
		lastStoresScrapeErrorMetric:           lastStoresScrapeErrorMetric,
		lastStoresScrapeTimestampMetric:       lastStoresScrapeTimestampMetric,
		lastStoresScrapeDurationSecondsMetric: lastStoresScrapeDurationSecondsMetric,
	}
}

func (c StoresCollector) Collect(ch chan<- prometheus.Metric) {
	var begun = time.Now()

	errorMetric := float64(0)
	if err := c.reportStoresMetrics(ch); err != nil {
		errorMetric = float64(1)
		c.storesScrapeErrorsTotalMetric.Inc()
	}
	c.storesScrapeErrorsTotalMetric.Collect(ch)

	c.storesScrapesTotalMetric.Inc()
	c.storesScrapesTotalMetric.Collect(ch)

	c.lastStoresScrapeErrorMetric.Set(errorMetric)
	c.lastStoresScrapeErrorMetric.Collect(ch)

	c.lastStoresScrapeTimestampMetric.Set(float64(time.Now().Unix()))
	c.lastStoresScrapeTimestampMetric.Collect(ch)

	c.lastStoresScrapeDurationSecondsMetric.Set(time.Since(begun).Seconds())
	c.lastStoresScrapeDurationSecondsMetric.Collect(ch)
}

func (c StoresCollector) Describe(ch chan<- *prometheus.Desc) {
	c.storesTotalMetric.Describe(ch)
	c.storesScrapesTotalMetric.Describe(ch)
	c.storesScrapeErrorsTotalMetric.Describe(ch)
	c.lastStoresScrapeErrorMetric.Describe(ch)
	c.lastStoresScrapeTimestampMetric.Describe(ch)
	c.lastStoresScrapeDurationSecondsMetric.Describe(ch)
}

func (c StoresCollector) reportStoresMetrics(ch chan<- prometheus.Metric) error {
	c.storesTotalMetric.Reset()

	stores, err := api.GetStores(api.StoreFilter{})
	if err != nil {
		log.Errorf("Error while listing stores: %v", err)
		return err
	}

	for _, store := range stores {
		c.storesTotalMetric.WithLabelValues(store.Plugin).Inc()
	}

	c.storesTotalMetric.Collect(ch)

	return nil
}
