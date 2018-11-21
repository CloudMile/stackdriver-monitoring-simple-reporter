package metric_exporter

import (
	"context"
	"time"
)

type MetricExporter interface {
	ExportWeeklyMetrics(dateTime time.Time, projectID, metric, instanceName string, metricPoints []string)
	ExportWeeklyMetricsChart(startDate time.Time, projectID, metric, instanceName string, xValues []time.Time, yValues []float64)
	ExportWeeklyReport(projectID string, startDate time.Time)
	SendWeeklyReport(appCtx context.Context, projectID, mailReceiver string)
}
