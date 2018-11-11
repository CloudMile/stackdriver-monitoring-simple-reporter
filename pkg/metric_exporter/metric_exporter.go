package metric_exporter

import (
	"time"
)

type MetricExporter interface {
	ExportWeeklyMetrics(dateTime time.Time, projectID, metric, instanceName string, metricPoints []string)
	ExportWeeklyMetricsChart(startDate time.Time, projectID, metric, instanceName string, xValues []time.Time, yValues []float64)
}
