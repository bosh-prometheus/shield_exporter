# FAQ

### What metrics does this exporter report?

The Shield Prometheus Exporter gets information from a [Shield][shield] backend. The metrics that are being [reported][shield_exporter_metrics] include:

* Cummulative Archives information
* Cummulative Jobs information
* Cummulative Retention Policies information
* Cummulative Schedules information
* Internal Status information
* Cummulative Stores information
* Cummulative Targets information
* Cummulative Tasks information

### How can I enable only a particular collector?

The `filter.collectors` command flag allows you to filter what collectors will be enabled (if not set, all collectors will be enabled by default). Possible values are `Archives`, `Jobs`, `RetentionPolicies`, `Schedules`, `Status`, `Stores`, `Targets`, `Tasks` (or a combination of them).

### Can I target multiple Shield Backends with a single exporter instance?

No, this exporter only supports targetting a single [Shield][shield] backend. If you want to get metrics from several backends, you will need to use one exporter per backend.

### I have a question but I don't see it answered at this FAQ

We will be glad to address any questions not answered here. Please, just open a [new issue][issues].

[issues]: https://github.com/cloudfoundry-community/shield_exporter/issues
[shield]: https://github.com/starkandwayne/shield
[shield_exporter_metrics]: https://github.com/cloudfoundry-community/shield_exporter#metrics
