# stackdriver-monitoring-simple-reporter

## Setup environment

`walkthrough cloud-shell-icon` Open Cloud shell

### Get current project id

```shell
gcloud config get-value project
```

If the project id is not your want, you can change it.

Replace `<PROJECT_ID>`.

```shell
gcloud config set project <PROJECT_ID>
```

### Enable needed Cloud API

We need the `Resource Manager` API to list projects that the GAE service account can access.

```shell
gcloud services enable monitoring.googleapis.com
gcloud services enable cloudresourcemanager.googleapis.com
```

## Create Google Cloud Storage Bucket(Option)

If you don't any GCS bucket, or you want create a new bucket. Execute below command.

Replace `<GCS_BUCKET_NAME>`.

```shell
gsutil mb gs://<GCS_BUCKET_NAME>
```

Quick Example:

```shell
gsutil mb "gs://${DEVSHELL_PROJECT_ID}-sdm-report"
```

## Setup application

### Configure

Edit the `config.yaml`.

```shell
cp config.yaml.example config.yaml
nano config.yaml
```

Replace `<GCS_BUCKET_NAME>`, `<EMAIL_ADDRESS_*>`. You can assign multi email adderss to `mailReceiver`.

```shell
timezone: 8
destination: <GCS_BUCKET_NAME>
mailReceiver: <EMAIL_ADDRESS_1>,<EMAIL_ADDRESS_2>
```

### Deploy application

```shell
gcloud app deploy
```

### Set cronjob

```shell
gcloud app deploy cron.yaml
```

## Configure Security

Deny all ingress traffic.

```shell
gcloud app firewall-rules update default --action deny
```

## Conclusion

Done!
