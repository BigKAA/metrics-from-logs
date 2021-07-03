// cSpell:disable
package prometheusmetric

import (
	"log"
	"strconv"
	"testing"
	"time"

	"github.com/BigKAA/metrics-from-logs/app/instance"
	"github.com/joho/godotenv"
)

func Test_GetPMFromRedisMetric(t *testing.T) {
	if err := godotenv.Load("D:\\Projects\\go\\metrics-from-logs\\.env"); err != nil {
		log.Print("No .env file found")
	}

	config := instance.GetConfig()

	logs := instance.GetLogEntry(config)

	i, err := instance.GetInstance(config, logs)
	if err != nil {
		logs.Error("f: GetInstance - :", err)
	}

	metrics, err := instance.FillMetrics(logs, config)
	if err != nil || metrics == nil {
		logs.Error("Неудалось сформировать массив метрик. ", err)
		t.Error("Неудалось сформировать массив метрик. ", err)
		t.Fail()
		return
	}
	metric := metrics[0]

	redisMetric := instance.RedisMetric{
		Metric:     metric.Mertic,
		Metrichelp: metric.Mertichelp,
		Metrictype: metric.Metrictype,
		Query:      metric.Query,
		Index:      metric.Index,
		Repeat:     strconv.Itoa(metric.Repeat),
		Labels:     metric.Labels,
	}

	metric_key := "mfl_test" + ":" + redisMetric.Metric + ":count"
	pm := PrometheusMetric{}

	pm.GetPMFromRedisMetric(redisMetric)

	err = pm.UpdateInRedis(metric_key, 100, time.Duration(600), // 10 минут
		i.Pool, i.Logs)
	if err != nil {
		t.Error("ERROR updatePrometheusMetric: ", err)
		t.Fail()
		return
	}
	t.Log("Success")
}
