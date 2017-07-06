FROM        quay.io/prometheus/busybox:latest
MAINTAINER  Ferran Rodenas <frodenas@gmail.com>

COPY shield_exporter /bin/shield_exporter

ENTRYPOINT ["/bin/shield_exporter"]
EXPOSE     9179