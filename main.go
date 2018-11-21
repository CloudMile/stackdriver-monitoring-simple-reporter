package main

import (
	"fmt"
	"google.golang.org/appengine"
	"log"
	"net/http"
	"stackdriver-monitoring-simple-reporter/pkg/service"
)

func main() {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/cron/weekly-report-stuff", weeklyStuffJobHandler)
	http.HandleFunc("/cron/weekly-report", weeklyReportJobHandler)
	http.HandleFunc("/cron/monthly-report-stuff", monthlyStuffJobHandler)
	http.HandleFunc("/cron/monthly-report", monthlyReportJobHandler)
	http.HandleFunc("/export", exportMetricPointsHandler)

	appengine.Main()
}

// Index
func indexHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "")
}

/************************************************

Export Metric Points to CSV and PNG

************************************************/

func exportMetricPointsHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("%v, %v, %v, %v, %v, %v, %v",
		r.FormValue("projectID"),
		r.FormValue("metric"),
		r.FormValue("aligner"),
		r.FormValue("filter"),
		r.FormValue("instanceName"),
	)

	ctx := appengine.NewContext(r)
	exportService := service.NewExportService(ctx)

	dataRange := r.FormValue("dataRange")
	exportService.SetDataRange(dataRange)

	exportService.ExportStuff(
		r.FormValue("projectID"),
		r.FormValue("metric"),
		r.FormValue("aligner"),
		r.FormValue("filter"),
		r.FormValue("instanceName"),
	)

	fmt.Fprint(w, "Done")
}

/************************************************

Weekly

************************************************/

func weeklyStuffJobHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	exportService := service.NewExportService(ctx)
	exportService.SetWeekly()
	exportService.Do(ctx)

	fmt.Fprint(w, "Weekly Job Done")
}

func weeklyReportJobHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	exportService := service.NewExportService(ctx)
	exportService.SetWeekly()

	exportService.ExportWeeklyReport(ctx)

	fmt.Fprint(w, "Done")
}

/************************************************

Monthly

************************************************/

func monthlyStuffJobHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	exportService := service.NewExportService(ctx)
	exportService.SetMonthly()
	exportService.Do(ctx)

	fmt.Fprint(w, "Monthly Job Done")
}

func monthlyReportJobHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	exportService := service.NewExportService(ctx)
	exportService.SetMonthly()

	exportService.ExportWeeklyReport(ctx)

	fmt.Fprint(w, "Done")
}
