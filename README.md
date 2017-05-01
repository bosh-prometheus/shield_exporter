# Shield Prometheus Exporter [![Build Status](https://travis-ci.org/cloudfoundry-community/shield_exporter.png)](https://travis-ci.org/cloudfoundry-community/shield_exporter)

A [Prometheus][prometheus] exporter for [Shield][shield]. Please refer to the [FAQ][faq] for general questions about this exporter.

## Architecture overview

![](https://cdn.rawgit.com/cloudfoundry-community/shield_exporter/master/architecture/architecture.svg)

## Installation

### Binaries

Download the already existing [binaries][binaries] for your platform:

```bash
$ ./shield_exporter <flags>
```

### From source

Using the standard `go install` (you must have [Go][golang] already installed in your local machine):

```bash
$ go install github.com/cloudfoundry-community/shield_exporter
$ shield_exporter <flags>
```

### Cloud Foundry

The exporter can be deployed to an already existing [Cloud Foundry][cloudfoundry] environment:

```bash
$ git clone https://github.com/cloudfoundry-community/shield_exporter.git
$ cd shield_exporter
```

Modify the included [application manifest file][manifest] to include your Shield properties. Then you can push the exporter to your Cloud Foundry environment:

```bash
$ cf push
```

### BOSH

This exporter can be deployed using the [Prometheus BOSH Release][prometheus-boshrelease].

## Usage

### Flags

| Flag / Environment Variable | Required | Default | Description |
| --------------------------- | -------- | ------- | ----------- |
| `shield.backend_url`<br />`SHIELD_EXPORTER_SHIELD_BACKEND_URL` | Yes | | Shield Backend URL |
| `shield.username`<br />`SHIELD_EXPORTER_SHIELD_USERNAME` | Yes | | Shield Username |
| `shield.password`<br />`SHIELD_EXPORTER_SHIELD_PASSWORD` | Yes | | Shield Password |
| `filter.collectors`<br />`SHIELD_EXPORTER_FILTER_COLLECTORS` | No | | Comma separated collectors to filter. If not set, all collectors will be enabled (`Archives`, `Jobs`, `RetentionPolicies`, `Schedules`, `Status`, `Stores`, `Targets`, `Tasks`) |
| `metrics.namespace`<br />`SHIELD_EXPORTER_METRICS_NAMESPACE` | No | `shield` | Metrics Namespace |
| `metrics.environment`<br />`SHIELD_EXPORTER_METRICS_ENVIRONMENT` | No | | Environment label to be attached to metrics |
| `web.listen-address`<br />`SHIELD_EXPORTER_WEB_LISTEN_ADDRESS` | No | `:9179` | Address to listen on for web interface and telemetry |
| `web.telemetry-path`<br />`SHIELD_EXPORTER_WEB_TELEMETRY_PATH` | No | `/metrics` | Path under which to expose Prometheus metrics |
| `web.auth.username`<br />`SHIELD_EXPORTER_WEB_AUTH_USERNAME` | No | | Username for web interface basic auth |
| `web.auth.pasword`<br />`SHIELD_EXPORTER_WEB_AUTH_PASSWORD` | No | | Password for web interface basic auth |
| `web.tls.cert_file`<br />`SHIELD_EXPORTER_WEB_TLS_CERTFILE` | No | | Path to a file that contains the TLS certificate (PEM format). If the certificate is signed by a certificate authority, the file should be the concatenation of the server's certificate, any intermediates, and the CA's certificate |
| `web.tls.key_file`<br />`SHIELD_EXPORTER_WEB_TLS_KEYFILE` | No | | Path to a file that contains the TLS private key (PEM format) |

### Metrics

The exporter returns the following `Archives` metrics:

| Metric | Description | Labels |
| ------ | ----------- | ------ |
| *metrics.namespace*_archives_total | Labeled total number of Shield Archives | `environment`, `backend_name`, `archive_status`, `store_plugin`, `target_plugin` |
| *metrics.namespace*_archives_scrapes_total | Total number of scrapes for Shield Archives | `environment`, `backend_name` |
| *metrics.namespace*_archives_scrape_errors_total | Total number of scrape errors of Shield Archives | `environment`, `backend_name` |
| *metrics.namespace*_last_archives_scrape_error | Whether the last scrape of Archive metrics from Shield resulted in an error (`1` for error, `0` for success) | `environment`, `backend_name` |
| *metrics.namespace*_last_archives_scrape_timestamp | Number of seconds since 1970 since last scrape of Archive metrics from Shield | `environment`, `backend_name` |
| *metrics.namespace*_last_archives_scrape_duration_seconds | Duration of the last scrape of Archive metrics from Shield | `environment`, `backend_name` |

The exporter returns the following `Jobs` metrics:

| Metric | Description | Labels |
| ------ | ----------- | ------ |
| *metrics.namespace*_jobs_total | Labeled total number of Shield Jobs | `environment`, `backend_name`, `job_paused`, `store_plugin`, `target_plugin` |
| *metrics.namespace*_jobs_scrapes_total | Total number of scrapes for Shield Jobs | `environment`, `backend_name` |
| *metrics.namespace*_jobs_scrape_errors_total | Total number of scrape errors of Shield Jobs | `environment`, `backend_name` |
| *metrics.namespace*_last_jobs_scrape_error | Whether the last scrape of Job metrics from Shield resulted in an error (`1` for error, `0` for success) | `environment`, `backend_name` |
| *metrics.namespace*_last_jobs_scrape_timestamp | Number of seconds since 1970 since last scrape of Job metrics from Shield | `environment`, `backend_name` |
| *metrics.namespace*_last_jobs_scrape_duration_seconds | Duration of the last scrape of Job metrics from Shield | `environment`, `backend_name` |

The exporter returns the following `Retention Policies` metrics:

| Metric | Description | Labels |
| ------ | ----------- | ------ |
| *metrics.namespace*_retention_policies_total | Total number of Shield Retention Policies | `environment`, `backend_name` |
| *metrics.namespace*_retention_policies_scrapes_total | Total number of scrapes for Shield Retention Policies | `environment`, `backend_name` |
| *metrics.namespace*_retention_policies_scrape_errors_total | Total number of scrape errors of Shield Retention Policies | `environment`, `backend_name` |
| *metrics.namespace*_last_retention_policies_scrape_error | Whether the last scrape of Retention Policies metrics from Shield resulted in an error (`1` for error, `0` for success) | `environment`, `backend_name` |
| *metrics.namespace*_last_retention_policies_scrape_timestamp | Number of seconds since 1970 since last scrape of Retention Policies metrics from Shield | `environment`, `backend_name` |
| *metrics.namespace*_last_retention_policies_scrape_duration_seconds | Duration of the last scrape of Retention Policies metrics from Shield | `environment`, `backend_name` |

The exporter returns the following `Schedules` metrics:

| Metric | Description | Labels |
| ------ | ----------- | ------ |
| *metrics.namespace*_schedules_total | Total number of Shield Schedules | `environment`, `backend_name` |
| *metrics.namespace*_schedules_scrapes_total | Total number of scrapes for Shield Schedules | `environment`, `backend_name` |
| *metrics.namespace*_schedules_scrape_errors_total | Total number of scrape errors of Shield Schedules | `environment`, `backend_name` |
| *metrics.namespace*_last_schedules_scrape_error | Whether the last scrape of Schedule metrics from Shield resulted in an error (`1` for error, `0` for success) | `environment`, `backend_name` |
| *metrics.namespace*_last_schedules_scrape_timestamp | Number of seconds since 1970 since last scrape of Schedule metrics from Shield | `environment`, `backend_name` |
| *metrics.namespace*_last_schedules_scrape_duration_seconds | Duration of the last scrape of Schedule metrics from Shield | `environment`, `backend_name` |

The exporter returns the following `Status` metrics:

| Metric | Description | Labels |
| ------ | ----------- | ------ |
| *metrics.namespace*_status_pending_tasks_total | Total number of Shield pending Tasks | `environment`, `backend_name` |
| *metrics.namespace*_status_running_tasks_total | Total number of Shield running Tasks | `environment`, `backend_name` |
| *metrics.namespace*_status_schedule_queue_total | Total number of Shield Tasks in the supervisor scheduler queue | `environment`, `backend_name` |
| *metrics.namespace*_status_run_queue_total | Total number of Shield Tasks in the supervisor run queue | `environment`, `backend_name` |
| *metrics.namespace*_status_scrapes_total | Total number of scrapes for Shield Status | `environment`, `backend_name` |
| *metrics.namespace*_status_scrape_errors_total | Total number of scrape errors of Shield Status | `environment`, `backend_name` |
| *metrics.namespace*_last_status_scrape_error | Whether the last scrape of Status metrics from Shield resulted in an error (`1` for error, `0` for success) |`environment`, `backend_name` |
| *metrics.namespace*_last_status_scrape_timestamp | Number of seconds since 1970 since last scrape of Status metrics from Shield | `environment`, `backend_name` |
| *metrics.namespace*_last_status_scrape_duration_seconds | Duration of the last scrape of Status metrics from Shield | `environment`, `backend_name` |

The exporter returns the following `Stores` metrics:

| Metric | Description | Labels |
| ------ | ----------- | ------ |
| *metrics.namespace*_stores_total | Labeled total number of Shield Stores | `environment`, `backend_name`, `store_plugin` |
| *metrics.namespace*_stores_scrapes_total | Total number of scrapes for Shield Stores | `environment`, `backend_name` |
| *metrics.namespace*_stores_scrape_errors_total | Total number of scrape errors of Shield Stores | `environment`, `backend_name` |
| *metrics.namespace*_last_stores_scrape_error | Whether the last scrape of Store metrics from Shield resulted in an error (`1` for error, `0` for success) |`environment`, `backend_name` |
| *metrics.namespace*_last_stores_scrape_timestamp | Number of seconds since 1970 since last scrape of Store metrics from Shield | `environment`, `backend_name` |
| *metrics.namespace*_last_stores_scrape_duration_seconds | Duration of the last scrape of Store metrics from Shield | `environment`, `backend_name` |

The exporter returns the following `Targets` metrics:

| Metric | Description | Labels |
| ------ | ----------- | ------ |
| *metrics.namespace*_targets_total | Labeled total number of Shield Targets | `environment`, `backend_name`, `target_plugin` |
| *metrics.namespace*_targets_scrapes_total | Total number of scrapes for Shield Targets | `environment`, `backend_name` |
| *metrics.namespace*_targets_scrape_errors_total | Total number of scrape errors of Shield Targets | `environment`, `backend_name` |
| *metrics.namespace*_last_targets_scrape_error | Whether the last scrape of Target metrics from Shield resulted in an error (`1` for error, `0` for success) |`environment`, `backend_name` |
| *metrics.namespace*_last_targets_scrape_timestamp | Number of seconds since 1970 since last scrape of Target metrics from Shield | `environment`, `backend_name` |
| *metrics.namespace*_last_targets_scrape_duration_seconds | Duration of the last scrape of Target metrics from Shield | `environment`, `backend_name` |

The exporter returns the following `Tasks` metrics:

| Metric | Description | Labels |
| ------ | ----------- | ------ |
| *metrics.namespace*_tasks_total | Labeled total number of Shield Tasks | `environment`, `backend_name`, `task_operation`, `task_status` |
| *metrics.namespace*_tasks_duration_seconds | Labeled summary of Shield Task durations in seconds | `environment`, `backend_name`,  `task_operation`, `task_status` |
| *metrics.namespace*_tasks_scrapes_total | Total number of scrapes for Shield Tasks | `environment`, `backend_name` |
| *metrics.namespace*_tasks_scrape_errors_total | Total number of scrape errors of Shield Tasks | `environment`, `backend_name` |
| *metrics.namespace*_last_tasks_scrape_error | Whether the last scrape of Task metrics from Shield resulted in an error (`1` for error, `0` for success) |`environment`, `backend_name` |
| *metrics.namespace*_last_tasks_scrape_timestamp | Number of seconds since 1970 since last scrape of Task metrics from Shield | `environment`, `backend_name` |
| *metrics.namespace*_last_tasks_scrape_duration_seconds | Duration of the last scrape of Task metrics from Shield | `environment`, `backend_name` |

## Contributing

Refer to the [contributing guidelines][contributing].

## License

Apache License 2.0, see [LICENSE][license].

[binaries]: https://github.com/cloudfoundry-community/shield_exporter/releases
[cloudfoundry]: https://www.cloudfoundry.org/
[contributing]: https://github.com/cloudfoundry-community/shield_exporter/blob/master/CONTRIBUTING.md
[faq]: https://github.com/cloudfoundry-community/shield_exporter/blob/master/FAQ.md
[golang]: https://golang.org/
[license]: https://github.com/cloudfoundry-community/shield_exporter/blob/master/LICENSE
[manifest]: https://github.com/cloudfoundry-community/shield_exporter/blob/master/manifest.yml
[prometheus]: https://prometheus.io/
[prometheus-boshrelease]: https://github.com/cloudfoundry-community/prometheus-boshrelease
[shield]: https://github.com/starkandwayne/shield
