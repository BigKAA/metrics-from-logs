// cSpell:disable
package instance

import (
	"testing"
	"time"

	"github.com/gomodule/redigo/redis"
)

func Test_updatePrometheusMetric(t *testing.T) {
	config := GetConfig()

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

	pm := GetPMFromRedisMetric(&redisMetric)

	metric_key := mfl_metric_prefix + ":" + redisMetric.Metric + ":count"
	err = pm.UpdateInRedis(metric_key, 100, expire_prom_metric, i.Pool, i.Logs)
	if err != nil {
		t.Error("ERROR updatePrometheusMetric: ", err)
		t.Fail()
		return
	}
	t.Log("Success")
}

func Test_redisMagic(t *testing.T) {

	config := GetConfig()

	logs := GetLogEntry(config)

	instance, err := GetInstance(config, logs)
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
	metric := &metrics[0]

	connSub := instance.Pool.Get()
	defer connSub.Close()
	conn := instance.Pool.Get()
	defer conn.Close()

	// записываем данные в редис
	send(conn, logs, metric)

	psc := redis.PubSubConn{connSub}
	psc.Subscribe(mfl_query)
	time.Sleep(time.Duration(5) * time.Second)
	switch v := psc.Receive().(type) {
	case redis.Message:
		logs.Debug("Message: " + v.Channel + " " + string(v.Data))
	case error:
		logs.Error("Error channel " + mfl_query)
	}

	// Читаем данные из редис
	rMetric, _, _, err := redisMagic(conn, logs)
	if err != nil {
		t.Error(err)
		t.Fail()
	} else {
		logs.Debug("rMetricQuery: ", rMetric.Query)
		t.Log("success")
	}
}

func Test_parseQuery(t *testing.T) {
	query := "{ \"query\": { \"bool\": {  \"filter\": [ { " +
		" \"range\": { \"@timestamp\": { " +
		" \"gte\": \"{{.Gte}}\", " +
		" \"lte\": \"{{.Lte}}\", " +
		"\"format\": \"strict_date_optional_time\" " +
		" } } }, { " +
		"\"match_phrase\": {  \"status\": 200 } } ] } }"

	config := GetConfig()

	logs := GetLogEntry(config)

	lte := time.Now()
	gte := time.Now().Add(time.Duration(-20) * time.Second)

	q, err := parseQuery(query, logs, lte, gte)
	if err != nil {
		t.Error(err)
		t.Fail()
	}
	t.Log("Passed query: ", q)
}
