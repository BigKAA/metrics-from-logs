// cSpell:disable
package instance

import (
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/sirupsen/logrus"
)

func Test_updatePrometheusMetric(t *testing.T) {

	prometheusLabels := [2]PrometheusLabels{
		{
			Name:  "test_name_1",
			Value: "test_value_1",
		},
		{
			Name:  "test_name_11",
			Value: "test_value_11",
		},
	}

	redisMetric := &RedisMetric{
		Metric:     "test_metric_1",
		Metrichelp: "help for metric",
		Metrictype: "counter",
		Query: "{ \"query\": { \"bool\": {  \"filter\": [ { " +
			" \"range\": { \"@timestamp\": { " +
			" \"gte\": \"{{.Gte}}\", " +
			" \"lte\": \"{{.Lte}}\", " +
			"\"format\": \"strict_date_optional_time\" " +
			" } } }, { " +
			"\"match_phrase\": {  \"status\": 200 } } ] } }",
		Index:  "oasi-stage-app-nginx-access-*",
		Repeat: "10",
		Labels: prometheusLabels[0:1],
	}

	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	level, _ := logrus.ParseLevel("debug")
	logger.SetLevel(level)
	logs := logger.WithFields(logrus.Fields{
		"pid": strconv.Itoa(os.Getpid()),
	})

	config := &Config{
		Confd:         "etc\\mfl\\conf.d\\",
		Loglevel:      "debug",
		Bindaddr:      "127.0.0.1:8080",
		Context:       "/",
		EsHost:        "127.0.0.1",
		EsPort:        "9200",
		EsUser:        "user",
		EsPassword:    "password",
		K8sPod:        "",
		K8sNamespace:  "",
		RedisServer:   "127.0.0.1",
		RedisPort:     "6379",
		RedisPassword: "",
	}

	pool := newRedisPool(config.RedisServer+":"+config.RedisPort, config.RedisPassword)
	//
	err := updatePrometheusMetric(pool, logs, 5111, redisMetric)
	if err != nil {
		t.Error("ERROR updatePrometheusMetric: ", err)
		t.Fail()
		return
	}
	t.Log("Success")
}

func Test_redisMagic(t *testing.T) {
	// Подготовка
	metric := &Metric{
		Mertic:     "test_metric_1",
		Mertichelp: "help for metric",
		Query: "{ \"query\": { \"bool\": {  \"filter\": [ { " +
			" \"range\": { \"@timestamp\": { " +
			" \"gte\": \"{{.Gte}}\", " +
			" \"lte\": \"{{.Lte}}\", " +
			"\"format\": \"strict_date_optional_time\" " +
			" } } }, { " +
			"\"match_phrase\": {  \"status\": 200 } } ] } }",
		Repeat:     15,
		Metrictype: "counter",
		Delay:      10,
	}

	// Переписать. Брать конфиг из .env или из env
	config := &Config{
		Confd:         "etc\\mfl\\conf.d\\",
		Loglevel:      "debug",
		Bindaddr:      "127.0.0.1:8080",
		Context:       "/",
		EsHost:        "127.0.0.1",
		EsPort:        "9200",
		EsUser:        "user",
		EsPassword:    "password",
		K8sPod:        "",
		K8sNamespace:  "",
		RedisServer:   "127.0.0.1",
		RedisPort:     "6379",
		RedisPassword: "",
	}

	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	level, _ := logrus.ParseLevel("debug")
	logger.SetLevel(level)
	logs := logger.WithFields(logrus.Fields{
		"pid": strconv.Itoa(os.Getpid()),
	})

	pool := newRedisPool(config.RedisServer+":"+config.RedisPort, config.RedisPassword)
	connSub := pool.Get()
	defer connSub.Close()
	conn := pool.Get()
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
	_, _, _, err := redisMagic(conn, logs)
	if err != nil {
		t.Fail()
	} else {
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

	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	level, _ := logrus.ParseLevel("debug")
	logger.SetLevel(level)
	logs := logger.WithFields(logrus.Fields{
		"pid": strconv.Itoa(os.Getpid()),
	})

	lte := time.Now()
	gte := time.Now().Add(time.Duration(-20) * time.Second)

	q, err := parseQuery(query, logs, lte, gte)
	if err != nil {
		t.Error(err)
		t.Fail()
	}
	t.Log("Passed query: ", q)
}
