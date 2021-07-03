// cSpell:disable
package instance

import (
	"bytes"
	"encoding/json"
	"errors"
	"html/template"
	"math/rand"
	"strconv"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/sirupsen/logrus"
)

// worker Основной процесс обработки метрик.
// Подписывается на канал в Redis.
func (i *Instance) worker() {
	c := i.Pool.Get()
	defer c.Close()
	psc := redis.PubSubConn{c}
	psc.Subscribe(mfl_query)

	for c.Err() == nil {
		switch v := psc.Receive().(type) {
		case redis.Message:
			i.Logs.Debug("f: worker - Message: " + v.Channel + " " + string(v.Data))
			// Небольшая рандомная задержка в пределах 2-х секунд
			n := rand.Intn(2000)
			time.Sleep(time.Duration(n) * time.Millisecond)

			go i.envelopePocessRecievedMetric()
		case error:
			i.Logs.Error("Error channel " + mfl_query)
		}
	}
}

// processRecievedMetric Забирает метрику из очереди.
// Выполняет запрос к elasticsearch.
func (i *Instance) envelopePocessRecievedMetric() {
	conn := i.Pool.Get()
	defer conn.Close()

	rMetric, num, err := processQuery(conn, i.Logs)
	if num == 1 {
		// это не ошибка. Пустая очередь.
		i.Logs.Debug("f: envelopePocessRecievedMetric - Empty query. ", err)
		return
	}
	if err != nil {
		i.Logs.Error("f: envelopePocessRecievedMetric - processQuery error: ", err)
		return
	}

	// Запрос в elasticsearch
	count, err := i.es(&rMetric)
	if err != nil {
		if err.Error() == "redisMagic empty query" {
			// Это не ошибка
			i.Logs.Debug("f: envelopePocessRecievedMetric - ", err)
		} else {
			i.Logs.Error("f: envelopePocessRecievedMetric - ", err)
		}
		return
	}

	// формируем метрику в redis
	// с именем:  mfl_metric_prefix:название_метрики:count
	pm := GetPMFromRedisMetric(&rMetric)

	metric_key := mfl_metric_prefix + ":" + rMetric.Metric + ":count"
	err = pm.UpdateInRedis(metric_key, count, expire_prom_metric, i.Pool, i.Logs)
	if err != nil {
		i.Logs.Error("f: es - error updateMetric: ", err)
		return
	}
}

// processQuery Суммирующая функция обработки
func processQuery(conn redis.Conn, logs *logrus.Entry) (RedisMetric, int64, error) {

	rMetric, lteUnix, gteUnix, err := redisMagic(conn, logs)
	if lteUnix == 1 {
		// это не  ошибка - это фича. Просто не было заданий в очереди.
		logs.Debug("f: processQuery - redisMagic empty query: ", errors.New("redisMagic empty query"))
		return RedisMetric{}, 1, err
	}
	if err != nil {
		logs.Error("f: processQuery - redisMagic error: ", err)
		return RedisMetric{}, 0, err
	}

	// Подготовка запроса.
	// Запрос приходит в виде шаблона.
	query, err := parseQuery(rMetric.Query, logs, time.Unix(lteUnix, 0), time.Unix(gteUnix, 0))
	if err != nil {
		logs.Debug("f: processQuery - parseQueryError: ", err)
		return RedisMetric{}, 0, err
	}

	logs.Debug("f: processQuery - parsed query: ", query)

	rMetric.Query = query

	return rMetric, 0, nil
}

// redisMagic забираем из очереди в Redis метрику и делаем все необходимое для предварительной
// обработки.
// На выходе RedisMetric. Веременные ограничения для запроса в Unix формате и ошибка.
func redisMagic(conn redis.Conn, logs *logrus.Entry) (RedisMetric, int64, int64, error) {
	// Читаем из очереди метрику для обработки.

	metric_key, err := conn.Do("LPOP", mfl_list)
	if err != nil {
		logs.Error("f: redisMagic - Redis LPOP: ", err)
		return RedisMetric{}, 0, 0, err
	}
	if metric_key == nil {
		logs.Debug("f: redisMagic - empty query. ", err)
		return RedisMetric{}, 1, 0, err
	}

	// Читаем метрику
	mString, err := redis.String(conn.Do("GET", metric_key))
	if err != nil {
		logs.Error("f: redisMagic - Redis GET error: ", err)
		return RedisMetric{}, 0, 0, err
	}
	b := []byte(mString)
	rMetric := RedisMetric{}
	// Парсим json
	err = json.Unmarshal(b, &rMetric)
	if err != nil {
		logs.Error("f: redisMagic - json.Unmarshal error: ", err)
		return RedisMetric{}, 0, 0, err
	}

	// Удаляем метрику
	_, err = conn.Do("DEL", metric_key)
	if err != nil {
		logs.Error("f: redisMagic - Redis DEL error: ", err)
	}

	// Временные метки выполнения запроса
	var gteUnix, lteUnix int64

	// Читаем время предыдущего выполнения запроса
	// Время в UNIX секундах
	metricTimeKey := mfl_metric_prefix + ":" + rMetric.Metric + ":time"

	// Проверяем наличие записи в Redis
	ok, err := redis.Bool(conn.Do("EXISTS", metricTimeKey))
	if err != nil {
		logs.Error("f: redisMagic - Redis EXIST error: ", err)
		return rMetric, 0, 0, err
	}

	if ok {
		gteUnix, err = redis.Int64(conn.Do("GETDEL", metricTimeKey))
		if err != nil {
			logs.Error("f: redisMagic - Redis GETDEL error: ", err)
		}
	} else {
		// такой метрики не было. Делаем в первый раз.
		// !!! Не контролируем ошибку !!! Надо исправить.
		repeat, _ := strconv.ParseInt(rMetric.Repeat, 10, 64)
		gteUnix = time.Now().Add(time.Duration(-repeat) * time.Second).Unix()
	}

	lteUnix = time.Now().Unix()

	// устанавливаем время предыдущего запроса
	_, err = conn.Do("SET", metricTimeKey, lteUnix)
	if err != nil {
		logs.Error("f: redisMagic - Redis SET metric time error: ", err)
	}

	_, err = conn.Do("EXPIRE", metricTimeKey, expire_prom_metric.Seconds())
	if err != nil {
		logs.Error("f: redisMagic - Redis EXPIRE error: ", err)
	}

	return rMetric, lteUnix, gteUnix, nil
}

// parseQuery Парсим шаблон запроса. Подставляем временные ограничения.
// Возвращаем готовый запрос.
// Входящее время в local timezone. Elasticsearch работсает с UTC
func parseQuery(query string, logs *logrus.Entry, lte time.Time, gte time.Time) (string, error) {
	type TimeLine struct {
		Lte string
		Gte string
	}

	utc, _ := time.LoadLocation("UTC")

	timeLine := TimeLine{
		Lte: lte.In(utc).Format("2006-01-02T15:04:05Z"),
		Gte: gte.In(utc).Format("2006-01-02T15:04:05Z"),
	}

	templ, err := template.New("es").Parse(query)
	if err != nil {
		logs.Error("f: parseQuery - error Parse: ", err)
		return "", err
	}
	var buf bytes.Buffer
	err = templ.Execute(&buf, timeLine)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
