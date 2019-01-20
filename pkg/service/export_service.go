package service

import (
	"context"
	"log"

	"google.golang.org/appengine/taskqueue"

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
