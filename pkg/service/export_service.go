package service

import (
	"context"
	"google.golang.org/appengine/taskqueue"
	"log"

	"stackdriver-monitoring-simple-reporter/pkg/gcp"
	"stackdriver-monitoring-simple-reporter/pkg/gcp/stackdriver"
	"stackdriver-monitoring-simple-reporter/pkg/metric_exporter"
	"stackdriver-monitoring-simple-reporter/pkg/utils"
)

var monitoringMetrics = []string{
	"compute.googleapis.com/instance/cpu/usage_time",
}

// sampled every 60 seconds
//
// * buffered
// * cached
// * free
// * used
//
var monitoringAgentMetrics = []string{
	"agent.googleapis.com/memory/bytes_used",
}

type ExportService struct {
	conf   utils.Conf
	client stackdriver.MonitoringClient
}

func NewExportService(ctx context.Context) *ExportService {
	var es = ExportService{}
	return es.init(ctx)
}

func (es *ExportService) newMetricExporter() metric_exporter.MetricExporter {
	return metric_exporter.NewGCSExporter(es.conf)
}

func (es *ExportService) init(ctx context.Context) *ExportService {
	es.conf.LoadConfig()

	es.client = stackdriver.MonitoringClient{}
	es.client.SetTimezone(es.conf.Timezone)
	es.client.SetContext(ctx)

	return es
}

func (es *ExportService) SetWeekly() {
	es.client.SetWeekly()
}

func (es *ExportService) SetMonthly() {
	es.client.SetMonthly()
}

func (es *ExportService) Do(ctx context.Context) {
	projectIDs := gcp.GetProjects(ctx)

	for prjIdx := range projectIDs {
		projectID := projectIDs[prjIdx]

		log.Printf("Query metrics in project ID: %s", projectID)

		// Common instance metrics
		es.exportInstanceCommonMetrics(ctx, projectID)

		// Agent metrics
		es.exportInstanceAgentMetrics(ctx, projectID)
	}
}

func (es *ExportService) exportInstanceCommonMetrics(ctx context.Context, projectID string) {
	for mIdx := range monitoringMetrics {
		metric := monitoringMetrics[mIdx]

		log.Printf("es.client.GetInstanceNames")
		instanceNames := es.client.GetInstanceNames(projectID, metric)

		for instIdx := range instanceNames {
			instanceName := instanceNames[instIdx]

			filter := stackdriver.MakeInstanceFilter(metric, instanceName)

			t := taskqueue.NewPOSTTask(
				"/export",
				map[string][]string{
					"projectID":         {projectID},
					"metric":            {metric},
					"aligner":           {stackdriver.AggregationPerSeriesAlignerRate},
					"filter":            {filter},
					"instanceName":      {instanceName},
					"intervalStartTime": {es.client.IntervalStartTime},
					"intervalEndTime":   {es.client.IntervalEndTime},
				},
			)
			if _, err := taskqueue.Add(ctx, t, ""); err != nil {
				log.Fatal(err.Error())
			}
		}
	}
}

func (es *ExportService) exportInstanceAgentMetrics(ctx context.Context, projectID string) {
	// We use the common metric to get the instance name, we can't query with agent metric
	instanceNames := es.client.GetInstanceNames(projectID, monitoringMetrics[0])

	for mIdx := range monitoringAgentMetrics {
		metric := monitoringAgentMetrics[mIdx]

		for instIdx := range instanceNames {
			instanceName := instanceNames[instIdx]

			// Currently only support instance memory
			filter := stackdriver.MakeAgentMemoryFilter(metric, instanceName)

			t := taskqueue.NewPOSTTask(
				"/export",
				map[string][]string{
					"projectID":         {projectID},
					"metric":            {metric},
					"aligner":           {stackdriver.AggregationPerSeriesAlignerMean},
					"filter":            {filter},
					"instanceName":      {instanceName},
					"intervalStartTime": {es.client.IntervalStartTime},
					"intervalEndTime":   {es.client.IntervalEndTime},
				},
			)
			if _, err := taskqueue.Add(ctx, t, ""); err != nil {
				log.Fatal(err.Error())
			}
		}
	}
}

func (es *ExportService) ExportWeeklyStuff(projectID, metric, aligner, filter, instanceName string) {
	es.ExportWeeklyMetrics(projectID, metric, aligner, filter, instanceName)
	es.ExportWeeklyMetricsGraph(projectID, metric, aligner, filter, instanceName)
}

func (es *ExportService) ExportWeeklyMetrics(projectID, metric, aligner, filter, instanceName string) {
	points := es.client.RetrieveMetricPoints(projectID, metric, aligner, filter)

	metricExporter := es.newMetricExporter()
	metricExporter.ExportWeeklyMetrics(es.client.StartTime.In(es.client.Location()), projectID, metric, instanceName, points)
}

func (es *ExportService) ExportWeeklyMetricsGraph(projectID, metric, aligner, filter, instanceName string) {
	xValues, yValues := es.client.RetrieveMetricPointsXY(projectID, metric, aligner, filter)

	metricExporter := es.newMetricExporter()
	metricExporter.ExportWeeklyMetricsChart(es.client.StartTime.In(es.client.Location()), projectID, metric, instanceName, xValues, yValues)
}

func (es *ExportService) ExportWeeklyReport(ctx context.Context) {
	metricExporter := es.newMetricExporter()

	projectIDs := gcp.GetProjects(ctx)

	for prjIdx := range projectIDs {
		projectID := projectIDs[prjIdx]
		metricExporter.ExportWeeklyReport(projectID, es.client.StartTime.In(es.client.Location()))
		metricExporter.SendWeeklyReport(ctx, projectID, es.conf.MailReceiver)
	}
}
