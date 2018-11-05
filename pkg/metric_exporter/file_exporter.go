package metric_exporter

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"stackdriver-monitoring-simple-reporter/pkg/gcp/stackdriver"
	"stackdriver-monitoring-simple-reporter/pkg/utils"
)

type FileExporter struct {
	Dir string
}

func NewFileExporter(c utils.Conf) MetricExporter {
	exporter := FileExporter{}
	exporter.Dir = c.Destination

	return exporter
}

func (f FileExporter) saveTimeSeriesToCSV(filename string, metricPoints []string) {
	log.Printf("Points len: %d", len(metricPoints))

	f.saveToFile(filename, stackdriver.PointCSVHeader, strings.Join(metricPoints, "\n"))
}

func (f FileExporter) saveToFile(filename, header, content string) {
	file, err := os.Create(filename)
	if err != nil {
		log.Fatal("Cannot create file", err)
	}
	defer file.Close()

	fmt.Fprintf(file, "%s\n", header)
	fmt.Fprintf(file, content)
}

func (f FileExporter) Export(dateTime time.Time, projectID, metric, instanceName string, metricPoints []string, attendNames ...string) {
	folder := fmt.Sprintf("%s/%s/%d/%2d/%2d/%s", f.Dir, projectID, dateTime.Year(), dateTime.Month(), dateTime.Day(), instanceName)
	os.MkdirAll(folder, os.ModePerm)

	title := strings.Replace(metric, "compute.googleapis.com/instance/", "", -1)
	title = strings.Replace(title, "/", "_", -1)

	var output string
	if len(attendNames) == 0 {
		output = fmt.Sprintf("%s/%s[%s][%s].csv", folder, dateTime.Format("2006-01-02"), instanceName, title)
	} else {
		output = fmt.Sprintf("%s/%s[%s][%s][%s].csv", folder, dateTime.Format("2006-01-02"), instanceName, title, strings.Join(attendNames, "-"))
	}

	f.saveTimeSeriesToCSV(output, metricPoints)
}
