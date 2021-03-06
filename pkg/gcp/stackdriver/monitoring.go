package stackdriver

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"

	"google.golang.org/api/monitoring/v3"
)

const (
	PointCSVHeader  = "timestamp,datetime,value"
	InstanceNameKey = "instanceName"

	AggregationAlignmentPeriod      = "3600s"
	AggregationPerSeriesAlignerRate = "ALIGN_RATE"
	AggregationPerSeriesAlignerMean = "ALIGN_MEAN"

	HoursOfOneWeek = 24 * 7
)

/************************************************

Initialize and Configuraion

************************************************/
type MonitoringClient struct {
	TimeZone          int
	StartTime         time.Time
	EndTime           time.Time
	IntervalStartTime string
	IntervalEndTime   string
	TotalHours        int
	client            *http.Client
}

func (c *MonitoringClient) SetTimezone(timezone int) {
	c.TimeZone = timezone
}

// Previous week
func (c *MonitoringClient) SetWeekly() {
	local := c.Location()
	now := time.Now().In(local)
	weekStartDay := now.AddDate(0, 0, -(int)(now.Weekday()))

	c.EndTime = time.Date(weekStartDay.Year(), weekStartDay.Month(), weekStartDay.Day(), 0, 0, 0, 0, local).UTC()
	c.StartTime = c.EndTime.AddDate(0, 0, -7)

	c.IntervalEndTime = c.EndTime.Format("2006-01-02T15:04:05.000000000Z")
	c.IntervalStartTime = c.StartTime.Format("2006-01-02T15:04:05.000000000Z")

	log.Printf("IntervalEndTime  : %s", c.IntervalEndTime)
	log.Printf("IntervalStartTime: %s", c.IntervalStartTime)

	c.TotalHours = HoursOfOneWeek
}

// Previous month
func (c *MonitoringClient) SetMonthly() {
	local := c.Location()
	now := time.Now().In(local)

	c.EndTime = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, local).UTC()
	c.StartTime = c.EndTime.AddDate(0, 0, -c.EndTime.Day())

	c.IntervalEndTime = c.EndTime.Format("2006-01-02T15:04:05.000000000Z")
	c.IntervalStartTime = c.StartTime.Format("2006-01-02T15:04:05.000000000Z")

	log.Printf("IntervalEndTime  : %s", c.IntervalEndTime)
	log.Printf("IntervalStartTime: %s", c.IntervalStartTime)

	c.TotalHours = c.EndTime.AddDate(0, 0, -1).Day() * 24
}

func (c *MonitoringClient) Location() *time.Location {
	localSecondsEastOfUTC := int((time.Duration(c.TimeZone) * time.Hour).Seconds())
	return time.FixedZone("localtime", localSecondsEastOfUTC)
}

func (c *MonitoringClient) getCred(ctx context.Context) (cred *google.Credentials) {
	cred, err := google.FindDefaultCredentials(ctx, monitoring.MonitoringReadScope)
	if err != nil {
		log.Fatal("getCred: ", err.Error())
	}
	log.Printf("Project ID: %s", cred.ProjectID)

	return
}

func (c *MonitoringClient) getClient() (client *http.Client) {
	if c.client == nil {
		ctx := context.Background()
		cred := c.getCred(ctx)
		c.client = c.newClient(ctx, cred)
	}

	client = c.client

	return
}

func (c *MonitoringClient) newClient(ctx context.Context, cred *google.Credentials) (client *http.Client) {
	conf, err := google.JWTConfigFromJSON(cred.JSON, monitoring.MonitoringReadScope)
	if err != nil {
		log.Fatal("newClient:", err.Error())
	}

	client = conf.Client(ctx)

	return
}

func (c *MonitoringClient) SetContext(ctx context.Context) {
	client, err := google.DefaultClient(ctx, monitoring.MonitoringReadScope)
	if err != nil {
		log.Fatal("SetContext: ", err.Error())
	}

	c.client = client
}

/************************************************

Generate Metric Filter

************************************************/

func MakeInstanceFilter(metric, instanceName string) string {
	return fmt.Sprintf(`metric.type="%s" AND metric.labels.instance_name="%s"`, metric, instanceName)
}

// Only query instance used memory from agent
func MakeAgentMemoryFilter(metric, instanceName string) string {
	return fmt.Sprintf(`metric.type="%s" AND metadata.user_labels.name="%s" AND metric.labels.state="%s"`, metric, instanceName, "used")
}

/************************************************

Get GCE Instance Name in Project

************************************************/

func (c *MonitoringClient) GetInstanceNames(projectID, metric string) (instanceNames []string) {
	client := c.getClient()

	svc, err := monitoring.New(client)
	if err != nil {
		log.Fatal("GetInstanceNames: ", err.Error())
	}

	project := "projects/" + projectID

	projectsTimeSeriesListCall := svc.Projects.TimeSeries.List(project)
	projectsTimeSeriesListCall.View("HEADERS")
	projectsTimeSeriesListCall.Filter(`metric.type="` + metric + `"`)
	projectsTimeSeriesListCall.IntervalStartTime(c.IntervalStartTime)
	projectsTimeSeriesListCall.IntervalEndTime(c.IntervalEndTime)

	listResp, err := projectsTimeSeriesListCall.Do()
	if err != nil {
		log.Fatal("GetInstanceNames: ", err.Error())
	}

	instanceNames = make([]string, len(listResp.TimeSeries))
	for i := range listResp.TimeSeries {
		instanceNames[i] = listResp.TimeSeries[i].Metric.Labels["instance_name"]
	}

	return
}

/************************************************

Timeseries List

************************************************/

func (c *MonitoringClient) RetrieveMetricPoints(projectID, metric, aligner, filter string) (metricPoints []string, xValues []time.Time, yValues []float64) {
	client := c.getClient()

	svc, err := monitoring.New(client)
	if err != nil {
		log.Fatal("RetrieveMetricPoints: ", err.Error())
	}

	project := "projects/" + projectID

	projectsTimeSeriesListCall := svc.Projects.TimeSeries.List(project)
	projectsTimeSeriesListCall.Filter(filter)
	projectsTimeSeriesListCall.IntervalStartTime(c.IntervalStartTime)
	projectsTimeSeriesListCall.IntervalEndTime(c.IntervalEndTime)
	projectsTimeSeriesListCall.AggregationPerSeriesAligner(aligner)
	projectsTimeSeriesListCall.AggregationAlignmentPeriod(AggregationAlignmentPeriod)

	listResp, err := projectsTimeSeriesListCall.Do()
	if err != nil {
		log.Fatal("RetrieveMetricPoints projectsTimeSeriesListCall: ", err.Error())
	}

	// Only get the first timeseries
	if len(listResp.TimeSeries) > 0 {
		timeSeries := listResp.TimeSeries[0]

		if len(timeSeries.Points) > 0 {
			metricPoints = c.pointsToMetricPoints(timeSeries.Points)
			xValues, yValues = c.pointsToXY(timeSeries.Points)
			return
		}
	}

	return
}

/************************************************

Timeseries CSV point (timestamp,datetime,value)

************************************************/

func (c *MonitoringClient) pointsToMetricPoints(points []*monitoring.Point) (metricPoints []string) {
	metricPoints = make([]string, c.TotalHours)

	pointTime := c.StartTime
	var pointIdx = len(points) - 1
	var t time.Time
	for metricIdx := range metricPoints {
		pointTime = pointTime.Add(time.Hour)

		if pointIdx >= 0 {
			t, _ = time.Parse("2006-01-02T15:04:05Z", points[pointIdx].Interval.StartTime)

			if pointTime.Equal(t) {
				t = t.Add(time.Hour * (time.Duration)(c.TimeZone))
				metricPoints[metricIdx] = fmt.Sprintf("%d,%s,%f", t.Unix(), t.Format("2006-01-02 15:04:05"), *(points[pointIdx].Value.DoubleValue))

				pointIdx = pointIdx - 1

				continue
			}
		}

		t = pointTime.Add(time.Hour * (time.Duration)(c.TimeZone))
		metricPoints[metricIdx] = fmt.Sprintf("%d,%s,", t.Unix(), t.Format("2006-01-02 15:04:05"))
	}

	return
}

/************************************************

Timeseries Graph point (X, Y)

************************************************/

func (c *MonitoringClient) pointsToXY(points []*monitoring.Point) (xValues []time.Time, yValues []float64) {
	xValues = make([]time.Time, c.TotalHours)
	yValues = make([]float64, c.TotalHours)

	pointTime := c.StartTime
	var pointIdx = len(points) - 1
	var t time.Time
	for metricIdx := 0; metricIdx < c.TotalHours; metricIdx++ {
		pointTime = pointTime.Add(time.Hour)

		if pointIdx >= 0 {
			t, _ = time.Parse("2006-01-02T15:04:05Z", points[pointIdx].Interval.StartTime)

			if pointTime.Equal(t) {
				t = t.Add(time.Hour * (time.Duration)(c.TimeZone))

				xValues[metricIdx] = t
				yValues[metricIdx] = *(points[pointIdx].Value.DoubleValue)

				pointIdx = pointIdx - 1

				continue
			}
		}

		t = pointTime.Add(time.Hour * (time.Duration)(c.TimeZone))

		xValues[metricIdx] = t
		yValues[metricIdx] = 0
	}

	return
}
