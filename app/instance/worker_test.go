// cSpell:disable
package instance

import (
	"strconv"
	"testing"
	"time"

	"github.com/gomodule/redigo/redis"
)

func Test_updatePrometheusMetric(t *testing.T) {
	config := getConfig()

	logs := getLogEntry(config)

	i, err := getInstance(config, logs)
	if err != nil {
		logs.Error("f: getInstance - :", err)
	}

	metrics, err := FillMetrics(logs, config)
	if err != nil || metrics == nil {
		logs.Error("Неудалось сформировать массив метрик. ", err)
		t.Error("Неудалось сформировать массив метрик. ", err)
		t.Fail()
		return
	}
	metric := metrics[0]

	redisMetric := &RedisMetric{
		Metric:     metric.Mertic,
		Metrichelp: metric.Mertichelp,
		Metrictype: metric.Metrictype,
		Query:      metric.Query,
		Index:      metric.Index,
		Repeat:     strconv.Itoa(metric.Repeat),
		Labels:     metric.Labels,
	}

	err = updatePrometheusMetric(i.pool, i.logs, 5111, redisMetric)
	if err != nil {
		t.Error("ERROR updatePrometheusMetric: ", err)
		t.Fail()
		return
	}
	t.Log("Success")
}

func Test_redisMagic(t *testing.T) {

	config := getConfig()

	logs := getLogEntry(config)

	instance, err := getInstance(config, logs)
	if err != nil {
		logs.Error("f: getInstance - :", err)
	}

	metrics, err := FillMetrics(logs, config)
	if err != nil || metrics == nil {
		logs.Error("Неудалось сформировать массив метрик. ", err)
		t.Error("Неудалось сформировать массив метрик. ", err)
		t.Fail()
		return
	}
	metric := &metrics[0]

	connSub := instance.pool.Get()
	defer connSub.Close()
	conn := instance.pool.Get()
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

	config := getConfig()

	logs := getLogEntry(config)

	lte := time.Now()
	gte := time.Now().Add(time.Duration(-20) * time.Second)

	q, err := parseQuery(query, logs, lte, gte)
	if err != nil {
		t.Error(err)
		t.Fail()
	}
	t.Log("Passed query: ", q)
}
