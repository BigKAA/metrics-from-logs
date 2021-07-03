// cSpell:disable
package prometheusmetric

import (
	"encoding/json"
	"time"

	"github.com/BigKAA/metrics-from-logs/app/instance"
	"github.com/gomodule/redigo/redis"
	"github.com/sirupsen/logrus"
)

type PrometheusLabels struct {
	Name  string
	Value string
}

// Структура в Redis для формировния метрики в формате Prometheus
type PrometheusMetric struct {
	Metric    string             `redis:"metric"`
	Help      string             `redis:"help"`
	Type      string             `redis:"type"`
	Count     int64              `redis:"count"`
	Timestamp int64              `redis:"timestamp"`
	Labels    []PrometheusLabels `redis:"labels"`
}

// GetPMFromRedisMetric заполняет поля PrometheusMetric из RedisMetric
func (pm *PrometheusMetric) GetPMFromRedisMetric(metric instance.RedisMetric) {
	pm.Metric = metric.Metric
	pm.Help = metric.Metrichelp
	pm.Type = metric.Metrictype
	pm.Count = 0
	pm.Timestamp = 0
	for _, label := range metric.Labels {
		pm.Labels = append(pm.Labels, PrometheusLabels{
			label.Name,
			label.Value,
		})
	}
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
			logs.Error("f: updatePrometheusMetric - json.Unmarshal error: ", err)
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
		logs.Error("f: updatePrometheusMetric - json.Marshal error: ", err)
		return err
	}

	// Записываем метрику в Redis
	_, err = conn.Do("SET", metric_key, string(b))
	if err != nil {
		logs.Error("f: updatePrometheusMetric - Redis SET error: ", err)
		return err
	}

	_, err = conn.Do("EXPIRE", metric_key, expire_prom_metric.Seconds())
	if err != nil {
		logs.Error("f: send - Redis EXPIRE error: ", err)
		return err
	}
	return nil
}
