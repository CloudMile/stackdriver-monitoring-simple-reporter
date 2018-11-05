package metric_exporter

import (
	"time"
)

type MetricExporter interface {
	Export(dateTime time.Time, projectID, metric, instanceName string, metricPoints []string, attendNames ...string)
}
