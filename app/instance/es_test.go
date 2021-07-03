// cSpell:disable
package instance

import (
	"testing"
	"time"

	"github.com/joho/godotenv"
)

func Test_executeEsCount(t *testing.T) {

	if err := godotenv.Load("..\\..\\.env"); err != nil {
		t.Error("No .env file found")
		t.Fail()
		return
	}

	config := GetConfig()

	logs := GetLogEntry(config)

	instance, err := GetInstance(config, logs)
	if err != nil {
		logs.Error("f: GetInstance - :", err)
		t.Error(err)
		t.Fail()
		return
	}

	lte := time.Now()
	gte := time.Now().Add(time.Duration(-20) * time.Second)

	query := "{\"query\":{\"bool\":{\"filter\":[{\"range\":{\"@timestamp\":{\"gte\":\"{{.Gte}}\",\"lte\":\"{{.Lte}}\",\"format\":\"strict_date_optional_time\"}}},{\"match_phrase\":{\"status\":200}}]}}}"

	q, err := parseQuery(query, logs, lte, gte)
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}

	metric := &RedisMetric{
		Metric:     "test_metric_1",
		Metrichelp: "help for metric",
		Metrictype: "counter",
		Query:      q,
		Repeat:     "15",
		Index:      "oasi-stage-app-nginx-access-*",
		Labels:     nil,
	}

	count, err := executeEsCount(metric, instance)
	if err != nil {
		t.Error("executeEsCount error: ", err)
		t.Fail()
		return
	}
	logs.Debug("f: es - count: ", count)
	t.Log("success")
}
