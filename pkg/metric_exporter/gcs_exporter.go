package metric_exporter

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"stackdriver-monitoring-simple-reporter/pkg/gcp/stackdriver"
	"stackdriver-monitoring-simple-reporter/pkg/utils"
)

type GCSExporter struct {
	BucketName string
}

func NewGCSExporter(c utils.Conf) MetricExporter {
	exporter := GCSExporter{}
	exporter.BucketName = c.Destination

	return exporter
}

func (g GCSExporter) saveTimeSeriesToCSV(filename string, metricPoints []string) {
	log.Printf("Points len: %d", len(metricPoints))

	content := fmt.Sprintf("%s\n%s", stackdriver.PointCSVHeader, strings.Join(metricPoints, "\n"))
	r := strings.NewReader(content)

	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	bh := client.Bucket(g.BucketName)
	obj := bh.Object(filename)
	w := obj.NewWriter(ctx)
	if _, err := io.Copy(w, r); err != nil {
		log.Fatalf("Failed to export metrics: %v", err)
	}
	if err := w.Close(); err != nil {
		log.Fatalf("Failed to export metrics: %v", err)
	}
}

//
// <destination>/
// └── <project_id>
//     └── 2018
//         └── weekly
//             └── 2018-1028-1104
//                 ├── 2018-1028-1104[instance_name][cpu_usage_time].csv
//  							 └── 2018-1028-1104[instance_name][memory_bytes_used].csv
//
func (g GCSExporter) ExportWeeklyMetrics(startDate time.Time, projectID, metric, instanceName string, metricPoints []string) {
	endDate := startDate.AddDate(0, 0, 7)
	weekStr := fmt.Sprintf("%d-%02d%02d-%02d%02d", startDate.Year(), startDate.Month(), startDate.Day(), endDate.Month(), endDate.Day())
	folder := fmt.Sprintf("%s/%d/weekly/%s", projectID, startDate.Year(), weekStr)

	title := strings.Replace(metric, "compute.googleapis.com/instance/", "", -1)
	title = strings.Replace(title, "agent.googleapis.com/", "", -1)
	title = strings.Replace(title, "/", "_", -1)

	output := fmt.Sprintf("%s/%s-%s[%s][%s].csv", folder, startDate.Format("2006-0102"), endDate.Format("0102"), instanceName, title)

	g.saveTimeSeriesToCSV(output, metricPoints)
}
