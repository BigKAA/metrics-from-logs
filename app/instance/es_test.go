// cSpell:disable
package instance

import (
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

func Test_executeEsCount(t *testing.T) {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	level, _ := logrus.ParseLevel("debug")
	logger.SetLevel(level)
	logs := logger.WithFields(logrus.Fields{
		"pid": strconv.Itoa(os.Getpid()),
	})

	if err := godotenv.Load("..\\..\\.env"); err != nil {
		t.Error("No .env file found")
		t.Fail()
		return
	}

	config := &Config{
		Confd:         getEnv("MFL_CONF_DIR", "etc\\mfl\\conf.d\\"),
		Loglevel:      getEnv("MFL_LOG_LEVEL", "debug"),
		Bindaddr:      getEnv("MFL_BIND_ADDR", "127.0.0.1:8080"),
		Context:       getEnv("MFL_CONTEXT", "/"),
		EsHost:        getEnv("MFL_ES_HOST", "10.206.189.67"),
		EsPort:        getEnv("MFL_ES_PORT", "80"),
		EsUser:        getEnv("MFL_ES_USER", "user"),
		EsPassword:    getEnv("MFL_ES_PASSWORD", "password"),
		K8sPod:        getEnv("MFL_K8S_POD", ""),
		K8sNamespace:  getEnv("MFL_K8S_NAMESPACE", ""),
		RedisServer:   getEnv("MFL_REDIS_SERVER", "127.0.0.1"),
		RedisPort:     getEnv("MFL_REDIS_PORT", "6379"),
		RedisPassword: getEnv("MFL_REDIS_PASSWORD", ""),
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
		Query:      q,
		Repeat:     "15",
		Metrictype: "counter",
		Index:      "oasi-stage-app-nginx-access-*",
	}

	instance := &Instance{
		logs:    logs,
		config:  config,
		metrics: nil,
	}

	es, _ := getEsClient(instance)

	count, err := executeEsCount(es, metric, logs)
	if err != nil {
		t.Error("executeEsCount error: ", err)
		t.Fail()
		return
	}
	logs.Debug("f: es - count: ", count)
	t.Log("success")
}

func Test_getEsClient(t *testing.T) {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	level, _ := logrus.ParseLevel("debug")
	logger.SetLevel(level)
	logs := logger.WithFields(logrus.Fields{
		"pid": strconv.Itoa(os.Getpid()),
	})

	if err := godotenv.Load("..\\..\\.env"); err != nil {
		t.Error("No .env file found")
		t.Fail()
		return
	}

	config := &Config{
		Confd:         getEnv("MFL_CONF_DIR", "etc\\mfl\\conf.d\\"),
		Loglevel:      getEnv("MFL_LOG_LEVEL", "debug"),
		Bindaddr:      getEnv("MFL_BIND_ADDR", "127.0.0.1:8080"),
		Context:       getEnv("MFL_CONTEXT", "/"),
		EsHost:        getEnv("MFL_ES_HOST", "127.0.0.1"),
		EsPort:        getEnv("MFL_ES_PORT", "9200"),
		EsUser:        getEnv("MFL_ES_USER", "user"),
		EsPassword:    getEnv("MFL_ES_PASSWORD", "password"),
		K8sPod:        getEnv("MFL_K8S_POD", ""),
		K8sNamespace:  getEnv("MFL_K8S_NAMESPACE", ""),
		RedisServer:   getEnv("MFL_REDIS_SERVER", "127.0.0.1"),
		RedisPort:     getEnv("MFL_REDIS_PORT", "6379"),
		RedisPassword: getEnv("MFL_REDIS_PASSWORD", ""),
	}

	instance := &Instance{
		logs:    logs,
		config:  config,
		metrics: nil,
	}

	_, err := getEsClient(instance)
	if err != nil {
		instance.logs.Error("f: Test_getEsClient - error getEsClient: ", err)
		t.Error("getEsClient error: ", err)
		t.Fail()
		return
	}

	t.Log("success")
}
