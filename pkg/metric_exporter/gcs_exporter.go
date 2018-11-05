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

func (g GCSExporter) Export(dateTime time.Time, projectID, metric, instanceName string, metricPoints []string, attendNames ...string) {
	folder := fmt.Sprintf("%s/%d/%2d/%2d/%s", projectID, dateTime.Year(), dateTime.Month(), dateTime.Day(), instanceName)

	title := strings.Replace(metric, "compute.googleapis.com/instance/", "", -1)
	title = strings.Replace(title, "agent.googleapis.com/", "", -1)
	title = strings.Replace(title, "/", "_", -1)

	var output string
	if len(attendNames) == 0 {
		output = fmt.Sprintf("%s/%s[%s][%s].csv", folder, dateTime.Format("2006-01-02"), instanceName, title)
	} else {
		output = fmt.Sprintf("%s/%s[%s][%s][%s].csv", folder, dateTime.Format("2006-01-02"), instanceName, title, strings.Join(attendNames, "-"))
	}

	g.saveTimeSeriesToCSV(output, metricPoints)
}
