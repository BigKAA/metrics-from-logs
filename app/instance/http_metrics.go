// cSpell:disable
package instance

import (
	"fmt"
	"net/http"
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
		retString, _ := GetPMStringFromRedis(key, conn, logs)
		// добавляем строку в итговый результат.
		sb.WriteString(retString)
	}

	ret := sb.String()
	return ret
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
