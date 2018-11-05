package main

import (
	"fmt"
	"google.golang.org/appengine"
	"net/http"
)

func main() {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/cron/weekly-report", weeklyJobHandler)
	http.HandleFunc("/cron/monthly-report", monthlyJobHandler)
	http.HandleFunc("/export", exportMetricPointsHandler)

	appengine.Main()
}

// Index
func indexHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "")
}

func weeklyJobHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	// TODO: set time ranges
	exportService := service.NewExportService(ctx)
	exportService.Do(ctx)

	fmt.Fprint(w, "Weekly Job Done")
}

func monthlyJobHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	// TODO: set time ranges
	exportService := service.NewExportService(ctx)
	exportService.Do(ctx)

	fmt.Fprint(w, "Monthly Job Done")
}

// Export Metric Points to CSV
func exportMetricPointsHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("%v, %v, %v, %v, %v, %v",
		r.FormValue("projectID"),
		r.FormValue("metric"),
		r.FormValue("aligner"),
		r.FormValue("filter"),
		r.FormValue("instanceName"),
		strings.Split(r.FormValue("attendNames"), "|"),
	)

	ctx := appengine.NewContext(r)
	exportService := service.NewExportService(ctx)

	attendNamesStr := r.FormValue("attendNames")
	if attendNamesStr == "" {
		exportService.Export(
			r.FormValue("projectID"),
			r.FormValue("metric"),
			r.FormValue("aligner"),
			r.FormValue("filter"),
			r.FormValue("instanceName"),
		)
	} else {
		exportService.Export(
			r.FormValue("projectID"),
			r.FormValue("metric"),
			r.FormValue("aligner"),
			r.FormValue("filter"),
			r.FormValue("instanceName"),
			strings.Split(attendNamesStr, "|")...,
		)
	}

	fmt.Fprint(w, "Done")
}
