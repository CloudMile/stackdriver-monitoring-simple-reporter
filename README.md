# stackdriver-monitoring-simple-reporter

A GAE service to send GCE Instance CPU/Memory Usage report weekly/monthly.

Using go version 1.11 or above.

This is the Alpha version.

## Installation

Clone this project

```shell
git clone git@github.com:CloudMile/stackdriver-monitoring-simple-reporter.git
cd stackdriver-monitoring-exporter
```

Edit config

```shell
cp config.yaml.example config.yaml
vi config.yaml
```

Check the current GCP project

```shell
gcloud config list
```

Enable needed API

```shell
gcloud services enable monitoring.googleapis.com
gcloud services enable cloudresourcemanager.googleapis.com
```

Deploy project

```shell
gcloud app deploy
gcloud app deploy cron.yaml
```

## Support Metrics

* compute.googleapis.com/instance/cpu/usage_time
* agent.googleapis.com/memory/bytes_used

Documents:
* [GCP Metrics List](https://cloud.google.com/monitoring/api/metrics_gcp)
* [Agent Metrics List](https://cloud.google.com/monitoring/api/metrics_agent#agent-memory)


## Export

Weekly Metrics path format

```shell
<destination>/
└── <project_id>
    └── 2018
        └── weekly
            └── 2018-1028-1104
                ├── 2018-1028-1104[instance_name][cpu_usage_time].csv
 								└── 2018-1028-1104[instance_name][memory_bytes_used].csv
```

Monthly Metrics path format

```shell
<destination>/
└── <project_id>
    └── 2018
        └── monthly
            └── 2018-10
                ├── 2018-10[instance_name][cpu_usage_time].csv
 								└── 2018-10[instance_name][memory_bytes_used].csv
```
