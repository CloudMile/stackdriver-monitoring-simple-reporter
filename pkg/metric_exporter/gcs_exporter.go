package metric_exporter

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
	"google.golang.org/api/iterator"
	"google.golang.org/appengine/mail"

	"cloud.google.com/go/storage"
	"stackdriver-monitoring-simple-reporter/pkg/gcp/stackdriver"
	"stackdriver-monitoring-simple-reporter/pkg/utils"

	"github.com/wcharczuk/go-chart"
	"github.com/wcharczuk/go-chart/util"
)

type GCSExporter struct {
	BucketName string
	ReportName string
	ReportPath string
}

func NewGCSExporter(c utils.Conf) MetricExporter {
	exporter := &GCSExporter{}
	exporter.BucketName = c.Destination

	return exporter
}

func (g *GCSExporter) saveTimeSeriesToCSV(filename string, metricPoints []string) {
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
func (g *GCSExporter) ExportWeeklyMetrics(startDate time.Time, projectID, metric, instanceName string, metricPoints []string) {
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

func (g *GCSExporter) ExportWeeklyMetricsChart(startDate time.Time, projectID, metric, instanceName string, xValues []time.Time, yValues []float64) {

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

func (g *GCSExporter) saveTimeSeriesToPNG(filename string, graph chart.Chart) {
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

type ImageReader struct {
	Path   string
	Reader *storage.Reader
}

func (ir ImageReader) ImageTitle() string {
	r, _ := regexp.Compile(`\[(\w|-)+\]\[(\w|-)+\]`)
	return r.FindString(ir.Path)
}

func (g *GCSExporter) ExportWeeklyReport(projectID string, startDate time.Time) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	bh := client.Bucket(g.BucketName)

	basePath := basePathOfReportStuff(projectID, "weekly", startDate)
	log.Printf("basePath: %s", basePath)

	readers := make([]ImageReader, 0)

	q := &storage.Query{Prefix: fmt.Sprintf("%s/", basePath), Delimiter: "/"}
	it := bh.Objects(ctx, q)
	for {
		objAttrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatalf("Failed to list files: %v", err)
		}

		if strings.HasSuffix(objAttrs.Name, ".png") {
			log.Printf("%s", objAttrs.Name)

			png := bh.Object(objAttrs.Name)
			reader, err := png.NewReader(ctx)
			if err != nil {
				log.Fatalf("Failed to generate report: %v", err)
			}
			defer reader.Close()

			readers = append(readers, ImageReader{
				Path:   objAttrs.Name,
				Reader: reader,
			})
		}
	}

	// Generate report
	pdf := gofpdf.New("P", "mm", "A4", "")

	// Cover
	pdf.AddPage()
	pdf.SetFont("Times", "B", 24)
	pdf.CellFormat(0, 50, pdfTitle(projectID, "weekly", startDate), "", 1, "C", false, 0, "")

	// Pages
	pdf.SetFont("Times", "B", 16)

	readersLen := len(readers)

	// No output
	if readersLen == 0 {
		g.ReportName = ""
		g.ReportPath = ""
		return
	}

	for i := 0; i < readersLen; i += 2 {
		pdf.AddPage()

		cpuReader := readers[i]
		memReader := readers[i+1]

		_ = pdf.RegisterImageOptionsReader(cpuReader.Path, gofpdf.ImageOptions{ImageType: "png", ReadDpi: true}, cpuReader.Reader)
		_ = pdf.RegisterImageOptionsReader(memReader.Path, gofpdf.ImageOptions{ImageType: "png", ReadDpi: true}, memReader.Reader)

		pdf.CellFormat(0, 50, cpuReader.ImageTitle(), "", 1, "C", false, 0, "")
		pdf.Image(cpuReader.Path, 0, 0, -128, 0, true, "png", 0, "")

		pdf.CellFormat(0, 50, memReader.ImageTitle(), "", 1, "C", false, 0, "")
		pdf.Image(memReader.Path, 0, 0, -128, 0, true, "png", 0, "")
	}

	// Upload report
	g.ReportName = reportName(projectID, "weekly", startDate)
	g.ReportPath = fmt.Sprintf("%s/%s", basePath, g.ReportName)
	obj := bh.Object(g.ReportPath)
	w := obj.NewWriter(ctx)

	pdf.Output(w)

	if err := w.Close(); err != nil {
		log.Fatalf("Failed to export metrics: %v", err)
	}
}

func basePathOfReportStuff(projectID, reportType string, startDate time.Time) string {
	endDate := startDate.AddDate(0, 0, 7)
	durationStr := fmt.Sprintf("%d-%02d%02d-%02d%02d", startDate.Year(), startDate.Month(), startDate.Day(), endDate.Month(), endDate.Day())
	folder := fmt.Sprintf("%s/%d/%s/%s", projectID, startDate.Year(), reportType, durationStr)

	return folder
}

func pdfTitle(projectID, reportType string, startDate time.Time) string {
	endDate := startDate.AddDate(0, 0, 7)
	title := fmt.Sprintf("Metrics Weekly Report %s - %s", startDate.Format("2006/01/02"), endDate.Format("2006/01/02"))

	return title
}

func reportName(projectID, reportType string, startDate time.Time) string {
	endDate := startDate.AddDate(0, 0, 7)
	durationStr := fmt.Sprintf("%d-%02d%02d-%02d%02d", startDate.Year(), startDate.Month(), startDate.Day(), endDate.Month(), endDate.Day())
	return fmt.Sprintf("%s-%s-report-%s.pdf", durationStr, reportType, projectID)
}

////////////////

func (g *GCSExporter) SendWeeklyReport(appCtx context.Context, projectID, mailReceiver string) {
	log.Printf("SendWeeklyReport ReportName: %s", g.ReportName)
	log.Printf("SendWeeklyReport ReportPath: %s", g.ReportPath)

	if g.ReportPath == "" {
		return
	}

	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	bh := client.Bucket(g.BucketName)
	obj := bh.Object(g.ReportPath)
	r, err := obj.NewReader(ctx)
	if err != nil {
		log.Fatalf("Couldn't create reader: %v", err)
	}

	attachData, err := ioutil.ReadAll(r)
	if err != nil {
		log.Fatalf("Couldn't read report: %v", err)
	}

	attach := mail.Attachment{
		Name: g.ReportName,
		Data: attachData,
	}

	msg := &mail.Message{
		Sender:      sender(),
		To:          []string{mailReceiver},
		Subject:     subject(projectID),
		Body:        "You got report.",
		Attachments: []mail.Attachment{attach},
	}
	if err := mail.Send(appCtx, msg); err != nil {
		log.Printf("Sender: %s", msg.Sender)
		log.Printf("To: %s", mailReceiver)
		log.Fatalf("Couldn't send email: %v", err)
	} else {
		log.Printf("Report mail sent!")
	}
}

func sender() string {
	return fmt.Sprintf("Weekly report <noreply@%s.appspotmail.com>", os.Getenv("GOOGLE_CLOUD_PROJECT"))
}

func subject(projectID string) string {
	return fmt.Sprintf("Weekly report: %s", projectID)
}
