// cSpell:disable
package instance

import (
	"testing"
	"time"

	"github.com/joho/godotenv"
)

func Test_GetPMFromRedisMetric(t *testing.T) {
	if err := godotenv.Load("..\\..\\.env"); err != nil {
		t.Error("No .env file found: ", err)
		t.Fail()
		return
	}

	config := GetConfig()

	config.Confd = "..\\..\\" + config.Confd

	logs := GetLogEntry(config)

	i, err := GetInstance(config, logs)
	if err != nil {
		logs.Error("f: GetInstance - :", err)
	}

	metrics, err := FillMetrics(logs, config)
	if err != nil || metrics == nil {
		logs.Error("Неудалось сформировать массив метрик. ", err)
		t.Error("Неудалось сформировать массив метрик. ", err)
		t.Fail()
		return
	}
	metric := metrics[0]

	redisMetric := GetRedisMetricFromMetric(&metric)

	metric_key := "mfl_test" + ":" + redisMetric.Metric + ":count"
	pm := GetPMFromRedisMetric(&redisMetric)

	err = pm.UpdateInRedis(metric_key, 100, time.Duration(10)*time.Minute, i.Pool, i.Logs)
	if err != nil {
		t.Error("ERROR updatePrometheusMetric: ", err)
		t.Fail()
		return
	}
	t.Log("Success")
}
