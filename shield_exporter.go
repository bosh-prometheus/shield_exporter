package main

import (
	"net/http"
	"os"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"
	"github.com/starkandwayne/shield/api"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/bosh-prometheus/shield_exporter/collectors"
	"github.com/bosh-prometheus/shield_exporter/filters"
)

var (
	shieldBackendUrl = kingpin.Flag(
		"shield.backend_url", "Shield Backend URL ($SHIELD_EXPORTER_SHIELD_BACKEND_URL)",
	).Envar("SHIELD_EXPORTER_SHIELD_BACKEND_URL").Required().String()

	shieldUsername = kingpin.Flag(
		"shield.username", "Shield Username ($SHIELD_EXPORTER_SHIELD_USERNAME)",
	).Envar("SHIELD_EXPORTER_SHIELD_USERNAME").Required().String()

	shieldPassword = kingpin.Flag(
		"shield.password", "Shield Password ($SHIELD_EXPORTER_SHIELD_PASSWORD)",
	).Envar("SHIELD_EXPORTER_SHIELD_PASSWORD").Required().String()

	filterCollectors = kingpin.Flag(
		"filter.collectors", "Comma separated collectors to filter (Archives,Jobs,RetentionPolicies,Schedules,Status,Stores,Targets,Tasks) ($SHIELD_EXPORTER_FILTER_COLLECTORS)",
	).Envar("SHIELD_EXPORTER_FILTER_COLLECTORS").Default("").String()

	metricsNamespace = kingpin.Flag(
		"metrics.namespace", "Metrics Namespace ($SHIELD_EXPORTER_METRICS_NAMESPACE)",
	).Envar("SHIELD_EXPORTER_METRICS_NAMESPACE").Default("shield").String()

	metricsEnvironment = kingpin.Flag(
		"metrics.environment", "Environment label to be attached to metrics ($SHIELD_EXPORTER_METRICS_ENVIRONMENT)",
	).Envar("SHIELD_EXPORTER_METRICS_ENVIRONMENT").Required().String()

	listenAddress = kingpin.Flag(
		"web.listen-address", "Address to listen on for web interface and telemetry ($SHIELD_EXPORTER_WEB_LISTEN_ADDRESS)",
	).Envar("SHIELD_EXPORTER_WEB_LISTEN_ADDRESS").Default(":9179").String()

	metricsPath = kingpin.Flag(
		"web.telemetry-path", "Path under which to expose Prometheus metrics ($SHIELD_EXPORTER_WEB_TELEMETRY_PATH)",
	).Envar("SHIELD_EXPORTER_WEB_TELEMETRY_PATH").Default("/metrics").String()

	authUsername = kingpin.Flag(
		"web.auth.username", "Username for web interface basic auth ($SHIELD_EXPORTER_WEB_AUTH_USERNAME)",
	).Envar("SHIELD_EXPORTER_WEB_AUTH_USERNAME").String()

	authPassword = kingpin.Flag(
		"web.auth.password", "Password for web interface basic auth ($SHIELD_EXPORTER_WEB_AUTH_PASSWORD)",
	).Envar("SHIELD_EXPORTER_WEB_AUTH_PASSWORD").String()

	tlsCertFile = kingpin.Flag(
		"web.tls.cert_file", "Path to a file that contains the TLS certificate (PEM format). If the certificate is signed by a certificate authority, the file should be the concatenation of the server's certificate, any intermediates, and the CA's certificate ($SHIELD_EXPORTER_WEB_TLS_CERTFILE)",
	).Envar("SHIELD_EXPORTER_WEB_TLS_CERTFILE").ExistingFile()

	tlsKeyFile = kingpin.Flag(
		"web.tls.key_file", "Path to a file that contains the TLS private key (PEM format) ($SHIELD_EXPORTER_WEB_TLS_KEYFILE)",
	).Envar("SHIELD_EXPORTER_WEB_TLS_KEYFILE").ExistingFile()
)

func init() {
	prometheus.MustRegister(version.NewCollector(*metricsNamespace))
}

type basicAuthHandler struct {
	handler  http.HandlerFunc
	username string
	password string
}

func (h *basicAuthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	username, password, ok := r.BasicAuth()
	if !ok || username != h.username || password != h.password {
		log.Errorf("Invalid HTTP auth from `%s`", r.RemoteAddr)
		w.Header().Set("WWW-Authenticate", "Basic realm=\"metrics\"")
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}
	h.handler(w, r)
	return
}

func prometheusHandler() http.Handler {
	handler := prometheus.Handler()

	if *authUsername != "" && *authPassword != "" {
		handler = &basicAuthHandler{
			handler:  prometheus.Handler().ServeHTTP,
			username: *authUsername,
			password: *authPassword,
		}
	}

	return handler
}

func main() {
	log.AddFlags(kingpin.CommandLine)
	kingpin.Version(version.Print("shield_exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	log.Infoln("Starting shield_exporter", version.Info())
	log.Infoln("Build context", version.BuildContext())

	api.Cfg = &api.Config{
		Backend:  "default",
		Backends: map[string]string{},
		Aliases:  map[string]string{},
	}

	if err := api.Cfg.AddBackend(*shieldBackendUrl, "default"); err != nil {
		log.Errorf("Error adding Shield Backend: %s", err.Error())
		os.Exit(1)
	}

	authToken := api.BasicAuthToken(*shieldUsername, *shieldPassword)
	if err := api.Cfg.UpdateBackend("default", authToken); err != nil {
		log.Errorf("Error updating Shield Backend: %s", err.Error())
		os.Exit(1)
	}

	shieldStatus, err := api.GetStatus()
	if err != nil {
		log.Errorf("Error while getting Shield Status: %v", err.Error())
		os.Exit(1)
	}

	log.Infof("Collecting data from Shield `%s' version %s", shieldStatus.Name, shieldStatus.Version)

	var collectorsFilters []string
	if *filterCollectors != "" {
		collectorsFilters = strings.Split(*filterCollectors, ",")
	}
	collectorsFilter, err := filters.NewCollectorsFilter(collectorsFilters)
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	if collectorsFilter.Enabled(filters.ArchivesCollector) {
		archivesCollector := collectors.NewArchivesCollector(*metricsNamespace, *metricsEnvironment, shieldStatus.Name)
		prometheus.MustRegister(archivesCollector)
	}

	if collectorsFilter.Enabled(filters.JobsCollector) {
		jobsCollector := collectors.NewJobsCollector(*metricsNamespace, *metricsEnvironment, shieldStatus.Name)
		prometheus.MustRegister(jobsCollector)
	}

	if collectorsFilter.Enabled(filters.RetentionPoliciesCollector) {
		retentionPoliciesCollector := collectors.NewRetentionPoliciesCollector(*metricsNamespace, *metricsEnvironment, shieldStatus.Name)
		prometheus.MustRegister(retentionPoliciesCollector)
	}

	if collectorsFilter.Enabled(filters.SchedulesCollector) {
		schedulesCollector := collectors.NewSchedulesCollector(*metricsNamespace, *metricsEnvironment, shieldStatus.Name)
		prometheus.MustRegister(schedulesCollector)
	}

	if collectorsFilter.Enabled(filters.StatusCollector) {
		statusCollector := collectors.NewStatusCollector(*metricsNamespace, *metricsEnvironment, shieldStatus.Name)
		prometheus.MustRegister(statusCollector)
	}

	if collectorsFilter.Enabled(filters.StoresCollector) {
		storesCollector := collectors.NewStoresCollector(*metricsNamespace, *metricsEnvironment, shieldStatus.Name)
		prometheus.MustRegister(storesCollector)
	}

	if collectorsFilter.Enabled(filters.TargetsCollector) {
		targetsCollector := collectors.NewTargetsCollector(*metricsNamespace, *metricsEnvironment, shieldStatus.Name)
		prometheus.MustRegister(targetsCollector)
	}

	if collectorsFilter.Enabled(filters.TasksCollector) {
		tasksCollector := collectors.NewTasksCollector(*metricsNamespace, *metricsEnvironment, shieldStatus.Name)
		prometheus.MustRegister(tasksCollector)
	}

	handler := prometheusHandler()
	http.Handle(*metricsPath, handler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>Shield Exporter</title></head>
             <body>
             <h1>Shield Exporter</h1>
             <p><a href='` + *metricsPath + `'>Metrics</a></p>
             </body>
             </html>`))
	})

	if *tlsCertFile != "" && *tlsKeyFile != "" {
		log.Infoln("Listening TLS on", *listenAddress)
		log.Fatal(http.ListenAndServeTLS(*listenAddress, *tlsCertFile, *tlsKeyFile, nil))
	} else {
		log.Infoln("Listening on", *listenAddress)
		log.Fatal(http.ListenAndServe(*listenAddress, nil))
	}
}
