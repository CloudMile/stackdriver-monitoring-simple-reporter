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

	"github.com/wcharczuk/go-chart"
	"github.com/wcharczuk/go-chart/util"
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

func MemoryValueFormatter(v interface{}) string {
	typed, _ := v.(float64)
	return fmt.Sprintf(chart.DefaultFloatFormat, typed/1024/1024)
}

func (g GCSExporter) ExportWeeklyMetricsChart(startDate time.Time, projectID, metric, instanceName string, xValues []time.Time, yValues []float64) {

	graph := chart.Chart{
		Background: chart.Style{
			Padding: chart.Box{
				Top:    10,
				Left:   10,
				Right:  50,
				Bottom: 10,
			},
		},
		Width: 1096,
		XAxis: chart.XAxis{
			Name:      "DateTime (1 hour interval)",
			NameStyle: chart.StyleShow(),
			Style:     chart.StyleShow(),
			GridMajorStyle: chart.Style{
				Show:        true,
				StrokeColor: chart.ColorAlternateGray,
				StrokeWidth: 1.0,
			},
			GridMinorStyle: chart.Style{
				Show:        true,
				StrokeColor: chart.ColorAlternateGray,
				StrokeWidth: 1.0,
			},
			Ticks: generateTicks(xValues),
		},
		YAxis: chart.YAxis{
			Name:      "Value",
			NameStyle: chart.StyleShow(),
			Style:     chart.StyleShow(),
			// ValueFormatter: MemoryValueFormatter,
			GridMajorStyle: chart.Style{
				Show:            true,
				StrokeColor:     chart.ColorAlternateGray,
				StrokeDashArray: []float64{5.0, 5.0},
				StrokeWidth:     1.0,
			},
		},
		Series: []chart.Series{
			chart.TimeSeries{
				XValues: xValues,
				YValues: yValues,
			},
		},
	}

	endDate := startDate.AddDate(0, 0, 7)
	weekStr := fmt.Sprintf("%d-%02d%02d-%02d%02d", startDate.Year(), startDate.Month(), startDate.Day(), endDate.Month(), endDate.Day())
	folder := fmt.Sprintf("%s/%d/weekly/%s", projectID, startDate.Year(), weekStr)

	title := strings.Replace(metric, "compute.googleapis.com/instance/", "", -1)
	title = strings.Replace(title, "agent.googleapis.com/", "", -1)
	title = strings.Replace(title, "/", "_", -1)

	output := fmt.Sprintf("%s/%s-%s[%s][%s].png", folder, startDate.Format("2006-0102"), endDate.Format("0102"), instanceName, title)

	g.saveTimeSeriesToPNG(output, graph)
}

func (g GCSExporter) saveTimeSeriesToPNG(filename string, graph chart.Chart) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	bh := client.Bucket(g.BucketName)
	obj := bh.Object(filename)
	w := obj.NewWriter(ctx)

	graph.Render(chart.PNG, w)

	log.Printf("%v", graph.Series[0].GetStyle().Show)

	if err := w.Close(); err != nil {
		log.Fatalf("Failed to export metrics: %v", err)
	}
}

func generateTicks(xValues []time.Time) chart.Ticks {
	ticks := make([]chart.Tick, 0)
	day7 := 24 * 7
	ticks = append(ticks, chart.Tick{
		Value: float64(xValues[0].UnixNano()),
		Label: xValues[0].Format(chart.DefaultDateFormat),
	})
	for i := 23; i < day7; i += 24 {
		ticks = append(ticks, chart.Tick{
			Value: util.Time.ToFloat64((xValues[i])),
			Label: xValues[i].Format(chart.DefaultDateFormat),
		})
	}
	return ticks
}
