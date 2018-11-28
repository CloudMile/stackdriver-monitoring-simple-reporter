package metric_exporter

import (
	"context"
	"time"
)

type MetricExporter interface {
	ExportWeeklyMetrics(dateTime time.Time, projectID, metric, instanceName string, metricPoints []string)
	ExportWeeklyMetricsChart(startDate time.Time, projectID, metric, instanceName string, xValues []time.Time, yValues []float64, totalHour int)
	ExportWeeklyReport(projectID string, startDate time.Time)
	SendWeeklyReport(appCtx context.Context, projectID, mailReceiver string, startDate time.Time)

	ExportMonthlyMetrics(dateTime time.Time, projectID, metric, instanceName string, metricPoints []string)
	ExportMonthlyMetricsChart(startDate time.Time, projectID, metric, instanceName string, xValues []time.Time, yValues []float64, totalHour int)
	ExportMonthlyReport(projectID string, startDate time.Time)
	SendMonthlyReport(appCtx context.Context, projectID, mailReceiver string, startDate time.Time)
}
