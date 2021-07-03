// cSpell:disable
package instance

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gomodule/redigo/redis"
	"github.com/sirupsen/logrus"
)

// HandlerRoot Обработка запроса /
func (i *Instance) HandlerMetrics() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn := i.Pool.Get()
		defer conn.Close()
		fmt.Fprint(w, createMetricsPage(i.Logs, conn))
	}
}

// createMetricsPage подготовка страницы с метриками
func createMetricsPage(logs *logrus.Entry, conn redis.Conn) string {
	var sb strings.Builder

	// получаем список Prometheus метрик из Redis
	keys, err := getMetricsFromRedis(logs, conn)
	if err != nil {
		logs.Error("f: createMetricsPage - getMetricsFromRedis error: ", err)
	}

	// Получем по каждому ключу строку с метрикой в форматие Prometheus
	for _, key := range keys {
		retString, _ := getMetricString(logs, key, conn)
		// добавляем строку в итговый результат.
		sb.WriteString(retString)
	}

	ret := sb.String()
	return ret
}

//
func getMetricString(logs *logrus.Entry, metricKey string, conn redis.Conn) (string, error) {
	// получаем метрику
	pm := PrometheusMetric{}
	mString, _ := redis.String(conn.Do("GET", metricKey))
	if mString != "" {
		b := []byte(mString)
		// Парсим json
		err := json.Unmarshal(b, &pm)
		if err != nil {
			logs.Error("f: getMetricString - json.Unmarshal error: ", err)
			return "", err
		}
	}
	var sb strings.Builder

	sb.WriteString("# HELP " + pm.Metric + " " + pm.Help + "\n")
	sb.WriteString("# TYPE " + pm.Metric + " " + pm.Type + "\n")

	labels := makeLabelsString(&pm)
	if labels != "" {
		sb.WriteString(pm.Metric + labels + " " + strconv.FormatInt(pm.Count, 10) + "\n")
	} else {
		sb.WriteString(pm.Metric + " " + strconv.FormatInt(pm.Count, 10) + "\n")
	}
	return sb.String(), nil
}

// makeLabelsString Формирует строку с labels, типа "{label="blah", label2="blah"}"
func makeLabelsString(pm *PrometheusMetric) string {
	ii := len(pm.Labels)
	if ii == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("{")

	for _, label := range pm.Labels {
		ii--
		sb.WriteString(label.Name + "=\"" + label.Value + "\"")
		if ii > 0 {
			sb.WriteString(",")
		}
	}
	sb.WriteString("}")
	return sb.String()
}

// getMetricsFromRedis получаем список метрик из Redis
func getMetricsFromRedis(logs *logrus.Entry, conn redis.Conn) ([]string, error) {
	var keys []string
	// we scan with our iter offset, starting at 0
	if arr, err := redis.Values(conn.Do("SCAN", 0, "MATCH", mfl_metric_prefix+":*:count")); err != nil {
		logs.Error("f: getMetricsFromRedis - redis.Values error: ", err)
		return nil, err
	} else {
		// now we get the iter and the keys from the multi-bulk reply
		// iter, _ := redis.Int(arr[0], nil)
		// logs.Debug("f: getMetricsFromRedis - redis keys", iter)
		keys, _ = redis.Strings(arr[1], nil)
	}

	logs.Debug("f: getMetricsFromRedis - redis keys", keys)

	return keys, nil
}
