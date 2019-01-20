package service

import (
	"context"
	"stackdriver-monitoring-simple-reporter/pkg/gcp"
)

/************************************************

Weekly Export Stuff

************************************************/

func (es *ExportService) ExportWeeklyStuff(projectID, metric, aligner, filter, instanceName string) {
	points, xValues, yValues := es.client.RetrieveMetricPoints(projectID, metric, aligner, filter)

	if len(points) == 0 {
		return
	}

	metricExporter := es.newMetricExporter()
	metricExporter.ExportWeeklyMetrics(es.client.StartTime.In(es.client.Location()), projectID, metric, instanceName, points)
	metricExporter.ExportWeeklyMetricsChart(es.client.StartTime.In(es.client.Location()), projectID, metric, instanceName, xValues, yValues, es.client.TotalHours)
}

/************************************************

Weekly Send Report

************************************************/

func (es *ExportService) ExportWeeklyReport(ctx context.Context) {
	metricExporter := es.newMetricExporter()

	projectIDs := gcp.GetProjects(ctx)

	for prjIdx := range projectIDs {
		projectID := projectIDs[prjIdx]
		metricExporter.ExportWeeklyReport(projectID, es.client.StartTime.In(es.client.Location()))
		metricExporter.SendWeeklyReport(ctx, projectID, es.conf.MailReceiver, es.client.StartTime.In(es.client.Location()))
	}
}
