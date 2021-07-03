// cSpell:disable
package instance

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/sirupsen/logrus"
)

// GetPMStringFromRedis читает метрику из Redis и формирует строку в формате Prometheus
func GetPMStringFromRedis(metricKey string,
	conn redis.Conn, logs *logrus.Entry) (string, error) {
	pm := PrometheusMetric{}
	mString, _ := redis.String(conn.Do("GET", metricKey))
	if mString != "" {
		b := []byte(mString)
		// Парсим json
		err := json.Unmarshal(b, &pm)
		if err != nil {
			logs.Error("f: GetPMStringFromRedis - json.Unmarshal error: ", err)
			return "", err
		}
	}
	var sb strings.Builder

	sb.WriteString("# HELP " + pm.Metric + " " + pm.Help + "\n")
	sb.WriteString("# TYPE " + pm.Metric + " " + pm.Type + "\n")

	labels := pm.makeLabelsString()
	if labels != "" {
		sb.WriteString(pm.Metric + labels + " " + strconv.FormatInt(pm.Count, 10))
	} else {
		sb.WriteString(pm.Metric + " " + strconv.FormatInt(pm.Count, 10))
	}
	// Если есть, добавляем timestamp
	if pm.Timestamp != 0 {
		sb.WriteString(" " + strconv.FormatInt(pm.Timestamp, 10))
	}
	sb.WriteString("\n")
	return sb.String(), nil
}

// makeLabelsString Формирует строку метрики.
func (pm *PrometheusMetric) makeLabelsString() string {
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

// GetPMFromRedisMetric заполняет поля PrometheusMetric из RedisMetric
func GetPMFromRedisMetric(metric *RedisMetric) PrometheusMetric {
	pm := PrometheusMetric{}
	pm.Metric = metric.Metric
	pm.Help = metric.Metrichelp
	pm.Type = metric.Metrictype
	pm.Count = 0
	pm.Timestamp = 0
	pm.Labels = metric.Labels

	return pm
}

// UpdateInRedis Обновляет, если надо - создаёт PrometheusMetric в Redis
func (pm *PrometheusMetric) UpdateInRedis(metric_key string, count int64,
	expire_prom_metric time.Duration, pool *redis.Pool, logs *logrus.Entry) error {
	conn := pool.Get()
	defer conn.Close()

	// Получаем старое значение метрики
	pmString, _ := redis.String(conn.Do("GET", metric_key))
	// Если метрики нет было, то используем значение по умолчанию pm.
	// Если была - то заполняем структуру pm тем, что было в Redis
	if pmString != "" {
		pmBytes := []byte(pmString)

		err := json.Unmarshal(pmBytes, pm)
		if err != nil {
			logs.Error("f: UpdateInRedis - json.Unmarshal error: ", err)
			return err
		}
	}

	// Добавляем новое значение к существующему.
	pm.Count += count

	// Устанавливаем timestamp последнего изменения
	// The timestamp is an int64 (milliseconds since epoch, i.e. 1970-01-01 00:00:00 UTC,
	// excluding leap seconds), represented as required by Go's ParseInt() function.
	// https://prometheus.io/docs/instrumenting/exposition_formats/#comments-help-text-and-type-information
	pm.Timestamp = time.Now().UnixNano() / int64(time.Millisecond)

	b, err := json.Marshal(pm)
	if err != nil {
		logs.Error("f: UpdateInRedis - json.Marshal error: ", err)
		return err
	}

	// Записываем метрику в Redis
	_, err = conn.Do("SET", metric_key, string(b))
	if err != nil {
		logs.Error("f: UpdateInRedis - Redis SET error: ", err)
		return err
	}

	_, err = conn.Do("EXPIRE", metric_key, expire_prom_metric.Seconds())
	if err != nil {
		logs.Error("f: UpdateInRedis - Redis EXPIRE error: ", err)
		return err
	}
	return nil
}
