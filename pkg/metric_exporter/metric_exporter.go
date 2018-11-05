package metric_exporter

import (
	"time"
)

type MetricExporter interface {
	ExportWeeklyMetrics(dateTime time.Time, projectID, metric, instanceName string, metricPoints []string)
}
