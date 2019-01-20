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

	"stackdriver-monitoring-simple-reporter/pkg/gcp/stackdriver"
	"stackdriver-monitoring-simple-reporter/pkg/utils"

	"cloud.google.com/go/storage"

	"github.com/wcharczuk/go-chart"
	"github.com/wcharczuk/go-chart/drawing"
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

func getValueFormat(metric string) chart.ValueFormatter {
	if "compute.googleapis.com/instance/cpu/usage_time" == metric {
		return utils.CPUValueFormatter
	}
	return utils.MemoryValueFormatter
}

/************************************************

Weekly Report(CSV)

************************************************/

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

/************************************************

Monthly Report(CSV)

************************************************/

//
// <destination>/
// └── <project_id>
//     └── 2018
//         └── monthly
//             └── 2018-10
//                 ├── 2018-10[instance_name][cpu_usage_time].csv
//  							 └── 2018-10[instance_name][memory_bytes_used].csv
//
func (g *GCSExporter) ExportMonthlyMetrics(startDate time.Time, projectID, metric, instanceName string, metricPoints []string) {
	monthStr := fmt.Sprintf("%d-%02d", startDate.Year(), startDate.Month())
	folder := fmt.Sprintf("%s/%d/monthly/%s", projectID, startDate.Year(), monthStr)

	title := strings.Replace(metric, "compute.googleapis.com/instance/", "", -1)
	title = strings.Replace(title, "agent.googleapis.com/", "", -1)
	title = strings.Replace(title, "/", "_", -1)

	output := fmt.Sprintf("%s/%s[%s][%s].csv", folder, monthStr, instanceName, title)

	g.saveTimeSeriesToCSV(output, metricPoints)
}

/************************************************

Report Helper(PNG)

************************************************/

func (g *GCSExporter) saveTimeSeriesToPNG(filename string, graph chart.Chart) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	bh := client.Bucket(g.BucketName)
	obj := bh.Object(filename)
	w := obj.NewWriter(ctx)

	defer w.Close()

	err = graph.Render(chart.PNG, w)
	if err != nil {
		log.Fatalf("Failed to export metrics: %v", err)
	}
}

/************************************************

Weekly Report(PNG)

************************************************/

func (g *GCSExporter) ExportWeeklyMetricsChart(startDate time.Time, projectID, metric, instanceName string, xValues []time.Time, yValues []float64, totalHour int) {

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
			Ticks: generateWeeklyTicks(xValues, totalHour),
		},
		YAxis: chart.YAxis{
			Name:      "Value",
			NameStyle: chart.StyleShow(),
			Style: chart.Style{
				Show:                true,
				FontSize:            8.0,
				Font:                utils.GetFont(),
				TextHorizontalAlign: chart.TextHorizontalAlignRight,
			},
			ValueFormatter: getValueFormat(metric),
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
				Style: chart.Style{
					Show:        true,
					StrokeColor: drawing.ColorBlue,
					FillColor:   drawing.ColorBlue.WithAlpha(64),
				},
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

func generateWeeklyTicks(xValues []time.Time, totalHour int) chart.Ticks {
	ticks := make([]chart.Tick, 0)
	ticks = append(ticks, chart.Tick{
		Value: float64(xValues[0].UnixNano()),
		Label: xValues[0].Format(chart.DefaultDateFormat),
	})
	for i := 23; i < totalHour; i += 24 {
		ticks = append(ticks, chart.Tick{
			Value: util.Time.ToFloat64((xValues[i])),
			Label: xValues[i].Format(chart.DefaultDateFormat),
		})
	}
	return ticks
}

/************************************************

Month Report(PNG)

************************************************/

func (g *GCSExporter) ExportMonthlyMetricsChart(startDate time.Time, projectID, metric, instanceName string, xValues []time.Time, yValues []float64, totalHour int) {

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
			Ticks: generateMonthlyTicks(xValues, totalHour),
		},
		YAxis: chart.YAxis{
			Name:      "Value",
			NameStyle: chart.StyleShow(),
			Style: chart.Style{
				Show:                true,
				FontSize:            8.0,
				Font:                utils.GetFont(),
				TextHorizontalAlign: chart.TextHorizontalAlignRight,
			},
			ValueFormatter: getValueFormat(metric),
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
				Style: chart.Style{
					Show:        true,
					StrokeColor: drawing.ColorBlue,
					FillColor:   drawing.ColorBlue.WithAlpha(64),
				},
			},
		},
	}

	monthStr := fmt.Sprintf("%d-%02d", startDate.Year(), startDate.Month())
	folder := fmt.Sprintf("%s/%d/monthly/%s", projectID, startDate.Year(), monthStr)

	title := strings.Replace(metric, "compute.googleapis.com/instance/", "", -1)
	title = strings.Replace(title, "agent.googleapis.com/", "", -1)
	title = strings.Replace(title, "/", "_", -1)

	output := fmt.Sprintf("%s/%s[%s][%s].png", folder, monthStr, instanceName, title)

	g.saveTimeSeriesToPNG(output, graph)
}

func generateMonthlyTicks(xValues []time.Time, totalHour int) chart.Ticks {
	ticks := make([]chart.Tick, 0)
	ticks = append(ticks, chart.Tick{
		Value: float64(xValues[0].UnixNano()),
		Label: xValues[0].Format("02"),
	})
	for i := 23; i < totalHour; i += 24 {
		ticks = append(ticks, chart.Tick{
			Value: util.Time.ToFloat64((xValues[i])),
			Label: xValues[i].Format("02"),
		})
	}
	return ticks
}

/************************************************

Report Helper(PDF)

************************************************/

type ImageReader struct {
	Path   string
	Reader *storage.Reader
}

func (ir ImageReader) ImageTitle() string {
	r, _ := regexp.Compile(`\[(\w|-)+\]\[(\w|-)+\]`)
	return r.FindString(ir.Path)
}

/************************************************

Weekly Report(PDF)

************************************************/

func (g *GCSExporter) ExportWeeklyReport(projectID string, startDate time.Time) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	bh := client.Bucket(g.BucketName)

	basePath := basePathOfWeeklyReportStuff(projectID, "weekly", startDate)
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
	pdf.CellFormat(0, 50, weeklyReportTitle(projectID, "weekly", startDate), "", 1, "C", false, 0, "")

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
	g.ReportName = weeklyReportName(projectID, "weekly", startDate)
	g.ReportPath = fmt.Sprintf("%s/%s", basePath, g.ReportName)
	obj := bh.Object(g.ReportPath)
	w := obj.NewWriter(ctx)

	defer w.Close()

	err = pdf.Output(w)
	if err != nil {
		log.Fatalf("Failed to export metrics: %v", err)
	}
}

func basePathOfWeeklyReportStuff(projectID, reportType string, startDate time.Time) string {
	endDate := startDate.AddDate(0, 0, 7)
	durationStr := fmt.Sprintf("%d-%02d%02d-%02d%02d", startDate.Year(), startDate.Month(), startDate.Day(), endDate.Month(), endDate.Day())
	folder := fmt.Sprintf("%s/%d/%s/%s", projectID, startDate.Year(), reportType, durationStr)

	return folder
}

func weeklyReportTitle(projectID, reportType string, startDate time.Time) string {
	endDate := startDate.AddDate(0, 0, 7)
	title := fmt.Sprintf("Metrics Weekly Report %s - %s", startDate.Format("2006/01/02"), endDate.Format("2006/01/02"))

	return title
}

func weeklyReportName(projectID, reportType string, startDate time.Time) string {
	endDate := startDate.AddDate(0, 0, 7)
	durationStr := fmt.Sprintf("%d-%02d%02d-%02d%02d", startDate.Year(), startDate.Month(), startDate.Day(), endDate.Month(), endDate.Day())
	return fmt.Sprintf("%s-%s-report-%s.pdf", durationStr, reportType, projectID)
}

/************************************************

Monthly Report(PDF)

************************************************/

func (g *GCSExporter) ExportMonthlyReport(projectID string, startDate time.Time) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	bh := client.Bucket(g.BucketName)

	basePath := basePathOfMonthlyReportStuff(projectID, "monthly", startDate)
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
	pdf.CellFormat(0, 50, monthlyReportTitle(projectID, "monthly", startDate), "", 1, "C", false, 0, "")

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
	g.ReportName = monthlyReportName(projectID, "monthly", startDate)
	g.ReportPath = fmt.Sprintf("%s/%s", basePath, g.ReportName)
	obj := bh.Object(g.ReportPath)
	w := obj.NewWriter(ctx)

	defer w.Close()

	err = pdf.Output(w)
	if err != nil {
		log.Fatalf("Failed to export metrics: %v", err)
	}
}

func basePathOfMonthlyReportStuff(projectID, reportType string, startDate time.Time) string {
	durationStr := fmt.Sprintf("%d-%02d", startDate.Year(), startDate.Month())
	folder := fmt.Sprintf("%s/%d/%s/%s", projectID, startDate.Year(), reportType, durationStr)

	return folder
}

func monthlyReportTitle(projectID, reportType string, startDate time.Time) string {
	title := fmt.Sprintf("Metrics Monthly Report %s", startDate.Format("2006/01"))

	return title
}

func monthlyReportName(projectID, reportType string, startDate time.Time) string {
	durationStr := fmt.Sprintf("%d-%02d", startDate.Year(), startDate.Month())
	return fmt.Sprintf("%s-%s-report-%s.pdf", durationStr, reportType, projectID)
}

/************************************************

Report Helper(Mail)

************************************************/

func sender() string {
	return fmt.Sprintf("GCP Report System<noreply@%s.appspotmail.com>", os.Getenv("GOOGLE_CLOUD_PROJECT"))
}

/************************************************

Weekly Report(Mail)

************************************************/

func (g *GCSExporter) SendWeeklyReport(appCtx context.Context, projectID, mailReceiver string, startDate time.Time) {
	log.Printf("SendWeeklyReport ReportName: %s", g.ReportName)
	log.Printf("SendWeeklyReport ReportPath: %s", g.ReportPath)

	if g.ReportPath == "" {
		return
	}

	attach := g.getAttachment()

	mailReceiver = strings.Replace(mailReceiver, " ", "", -1)
	mailReceivers := strings.Split(mailReceiver, ",")

	subject := weeklyReportSubject(projectID, startDate)

	msg := &mail.Message{
		Sender:      sender(),
		To:          mailReceivers,
		Subject:     subject,
		Body:        "You got report.",
		Attachments: []mail.Attachment{attach},
	}
	if err := mail.Send(appCtx, msg); err != nil {
		log.Printf("Sender: %s", msg.Sender)
		log.Printf("To: %s", mailReceiver)
		log.Fatalf("Couldn't send email: %v", err)
	} else {
		log.Printf("%s Report mail sent!", subject)
	}
}

func weeklyReportSubject(projectID string, startDate time.Time) string {
	endDate := startDate.AddDate(0, 0, 7)
	title := fmt.Sprintf("Metrics Weekly Report %s - %s: %s", startDate.Format("2006/01/02"), endDate.Format("2006/01/02"), projectID)

	return title
}

/************************************************

Monthly Report(Mail)

************************************************/

func (g *GCSExporter) SendMonthlyReport(appCtx context.Context, projectID, mailReceiver string, startDate time.Time) {
	log.Printf("SendMonthlyReport ReportName: %s", g.ReportName)
	log.Printf("SendMonthlyReport ReportPath: %s", g.ReportPath)

	if g.ReportPath == "" {
		return
	}

	attach := g.getAttachment()

	mailReceiver = strings.Replace(mailReceiver, " ", "", -1)
	mailReceivers := strings.Split(mailReceiver, ",")

	subject := monthlyReportSubject(projectID, startDate)

	msg := &mail.Message{
		Sender:      sender(),
		To:          mailReceivers,
		Subject:     subject,
		Body:        "You got report.",
		Attachments: []mail.Attachment{attach},
	}
	if err := mail.Send(appCtx, msg); err != nil {
		log.Printf("Sender: %s", msg.Sender)
		log.Printf("To: %s", mailReceiver)
		log.Fatalf("Couldn't send email: %v", err)
	} else {
		log.Printf("%s Report mail sent!", subject)
	}
}

func monthlyReportSubject(projectID string, startDate time.Time) string {
	title := fmt.Sprintf("Metrics Monthly Report %s: %s", startDate.Format("2006/01"), projectID)

	return title
}

/************************************************

Mail Attachment

************************************************/

func (g *GCSExporter) getAttachment() mail.Attachment {
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

	return mail.Attachment{
		Name: g.ReportName,
		Data: attachData,
	}
}
