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

const (
	DataRangeWeekly  = "weekly"
	DataRangeMonthly = "monthly"
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

/************************************************

Initialize and Configuraion

************************************************/

type ExportService struct {
	conf      utils.Conf
	client    stackdriver.MonitoringClient
	DataRange string
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
	es.DataRange = DataRangeWeekly
}

func (es *ExportService) SetMonthly() {
	es.client.SetMonthly()
	es.DataRange = DataRangeMonthly
}

func (es *ExportService) SetDataRange(dataRange string) {
	switch dataRange {
	case DataRangeMonthly:
		es.client.SetMonthly()
		es.DataRange = DataRangeMonthly
	default:
		es.client.SetWeekly()
		es.DataRange = DataRangeWeekly
	}
}

/************************************************

Process

************************************************/

func (es *ExportService) Do(ctx context.Context) {
	projectIDs := gcp.GetProjects(ctx)

	for prjIdx := range projectIDs {
		projectID := projectIDs[prjIdx]

		log.Printf("Query metrics in project ID: %s", projectID)

		// GCP metrics
		es.exportInstanceGCPMetrics(ctx, projectID)

		// Agent metrics
		es.exportInstanceAgentMetrics(ctx, projectID)
	}
}

/************************************************

Export GCP and Agent Metrics

************************************************/

func (es *ExportService) exportInstanceGCPMetrics(ctx context.Context, projectID string) {
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
					"dataRange":         {es.DataRange},
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
					"dataRange":         {es.DataRange},
				},
			)
			if _, err := taskqueue.Add(ctx, t, ""); err != nil {
				log.Fatal(err.Error())
			}
		}
	}
}

/************************************************

Export Stuff

************************************************/

func (es *ExportService) ExportStuff(projectID, metric, aligner, filter, instanceName string) {
	switch es.DataRange {
	case DataRangeMonthly:
		es.ExportMonthlyStuff(
			projectID,
			metric,
			aligner,
			filter,
			instanceName,
		)
	default:
		es.ExportWeeklyStuff(
			projectID,
			metric,
			aligner,
			filter,
			instanceName,
		)
	}
}

/************************************************

Weekly Export Stuff

************************************************/

func (es *ExportService) ExportWeeklyStuff(projectID, metric, aligner, filter, instanceName string) {
	es.ExportWeeklyMetrics(projectID, metric, aligner, filter, instanceName)
	es.ExportWeeklyMetricsGraph(projectID, metric, aligner, filter, instanceName)
}

func (es *ExportService) ExportWeeklyMetrics(projectID, metric, aligner, filter, instanceName string) {
	points := es.client.RetrieveMetricPoints(projectID, metric, aligner, filter)

	if len(points) == 0 {
		return
	}
	metricExporter := es.newMetricExporter()
	metricExporter.ExportWeeklyMetrics(es.client.StartTime.In(es.client.Location()), projectID, metric, instanceName, points)
}

func (es *ExportService) ExportWeeklyMetricsGraph(projectID, metric, aligner, filter, instanceName string) {
	xValues, yValues := es.client.RetrieveMetricPointsXY(projectID, metric, aligner, filter)

	if len(xValues) == 0 {
		return
	}

	metricExporter := es.newMetricExporter()
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
		metricExporter.SendWeeklyReport(ctx, projectID, es.conf.MailReceiver)
	}
}

/************************************************

Monthly Export Stuff

************************************************/

func (es *ExportService) ExportMonthlyStuff(projectID, metric, aligner, filter, instanceName string) {
	es.ExportMonthlyMetrics(projectID, metric, aligner, filter, instanceName)
	es.ExportMonthlyMetricsGraph(projectID, metric, aligner, filter, instanceName)
}

func (es *ExportService) ExportMonthlyMetrics(projectID, metric, aligner, filter, instanceName string) {
	points := es.client.RetrieveMetricPoints(projectID, metric, aligner, filter)

	if len(points) == 0 {
		return
	}

	metricExporter := es.newMetricExporter()
	metricExporter.ExportMonthlyMetrics(es.client.StartTime.In(es.client.Location()), projectID, metric, instanceName, points)
}

func (es *ExportService) ExportMonthlyMetricsGraph(projectID, metric, aligner, filter, instanceName string) {
	xValues, yValues := es.client.RetrieveMetricPointsXY(projectID, metric, aligner, filter)

	if len(xValues) == 0 {
		return
	}

	metricExporter := es.newMetricExporter()
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
		metricExporter.SendMonthlyReport(ctx, projectID, es.conf.MailReceiver)
	}
}
