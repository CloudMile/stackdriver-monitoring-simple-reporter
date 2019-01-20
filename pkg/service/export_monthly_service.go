package service

import (
	"context"
	"stackdriver-monitoring-simple-reporter/pkg/gcp"
)

/************************************************

Monthly Export Stuff

************************************************/

func (es *ExportService) ExportMonthlyStuff(projectID, metric, aligner, filter, instanceName string) {
	points, xValues, yValues := es.client.RetrieveMetricPoints(projectID, metric, aligner, filter)

	if len(points) == 0 {
		return
	}

	metricExporter := es.newMetricExporter()
	metricExporter.ExportMonthlyMetrics(es.client.StartTime.In(es.client.Location()), projectID, metric, instanceName, points)
	metricExporter.ExportMonthlyMetricsChart(es.client.StartTime.In(es.client.Location()), projectID, metric, instanceName, xValues, yValues, es.client.TotalHours)
}

/************************************************

Monthly Send Report

************************************************/

func (es *ExportService) ExportMonthlyReport(ctx context.Context) {
	metricExporter := es.newMetricExporter()

	projectIDs := gcp.GetProjects(ctx)

	for prjIdx := range projectIDs {
		projectID := projectIDs[prjIdx]
		metricExporter.ExportMonthlyReport(projectID, es.client.StartTime.In(es.client.Location()))
		metricExporter.SendMonthlyReport(ctx, projectID, es.conf.MailReceiver, es.client.StartTime.In(es.client.Location()))
	}
}
