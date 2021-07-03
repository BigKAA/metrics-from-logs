// cSpell:disable
package instance

import "strconv"

func GetRedisMetricFromMetric(metric *Metric) RedisMetric {
	return RedisMetric{
		Metric:     metric.Mertic,
		Metrichelp: metric.Mertichelp,
		Metrictype: metric.Metrictype,
		Query:      metric.Query,
		Index:      metric.Index,
		Repeat:     strconv.Itoa(metric.Repeat),
		Labels:     metric.Labels,
	}
}
