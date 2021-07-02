// cSpell:disable
package instance

import (
	"bytes"
	"encoding/json"
	"html/template"
	"strconv"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/sirupsen/logrus"
)

// worker Основной процесс обработки метрик.
// Подписывается на канал в Redis.
func (i *Instance) worker() {
	c := i.pool.Get()
	defer c.Close()
	psc := redis.PubSubConn{c}
	psc.Subscribe(mfl_query)

	for c.Err() == nil {
		switch v := psc.Receive().(type) {
		case redis.Message:
			i.logs.Debug("f: worker - Message: " + v.Channel + " " + string(v.Data))
			// Для работы требуется новый коннект к Redis
			go i.envelopePocessRecievedMetric()
		case error:
			i.logs.Error("Error channel " + mfl_query)
		}
	}
}

// processRecievedMetric Забирает метрику из очереди.
// Выполняет запрос к elasticsearch.
func (i *Instance) envelopePocessRecievedMetric() {
	conn := i.pool.Get()
	defer conn.Close()

	rMetric, _ := processQuery(conn, i.logs)

	// Запрос в elasticsearch
	count, err := i.es(&rMetric)
	if err != nil {
		i.logs.Error("f: envelopePocessRecievedMetric - ", err)
		return
	}

	// формируем метрику в redis
	// с именем:  mfl_metric_prefix:название_метрики:count
	// Время устаревания метрики: Metric.Repeat * 2
	err = updatePrometheusMetric(i, count, &rMetric)
	if err != nil {
		i.logs.Error("f: es - error updateMetric: ", err)
		return
	}
}

// updateMetric формируем () метрику в redis
// с именем:  mfl_metric_prefix:название_метрики:count
// Время устаревания метрики: Metric.Repeat * 2
func updatePrometheusMetric(i *Instance, count int64, rMetric *RedisMetric) error {
	conn := i.pool.Get()
	defer conn.Close()
	// Формируем имя метрики
	metric_key := mfl_metric_prefix + ":" + rMetric.Metric + ":count"

	promMetric := PrometheusMetric{
		Metric: rMetric.Metric,
		Labels: rMetric.Labels,
		Type:   rMetric.Metrictype,
		Help:   rMetric.Metrichelp,
		Count:  0,
	}

	// Получаем старое значение метрики
	pmString, _ := redis.String(conn.Do("GET", metric_key))
	// Если метрики нет было, то используем значение по умолчанию promMetric.
	// Если была - то заполняем структуру promMetric тем, что было в Redis
	if pmString != "" {
		pmBytes := []byte(pmString)

		err := json.Unmarshal(pmBytes, &promMetric)
		if err != nil {
			i.logs.Error("f: updatePrometheusMetric - json.Unmarshal error: ", err)
			return err
		}
	}

	// Добавляем новое значение к существующему.
	promMetric.Count += count

	b, err := json.Marshal(&promMetric)
	if err != nil {
		i.logs.Error("f: updatePrometheusMetric - json.Marshal error: ", err)
		return err
	}

	// Записываем метрику в Redis
	_, err = conn.Do("SET", metric_key, string(b))
	if err != nil {
		i.logs.Error("f: updatePrometheusMetric - Redis SET error: ", err)
		return err
	}

	_, err = conn.Do("EXPIRE", metric_key, expire_prom_metric.Seconds())
	if err != nil {
		i.logs.Error("f: send - Redis EXPIRE error: ", err)
		return err
	}
	return nil
}

// processQuery Суммирующая функция обработки
func processQuery(conn redis.Conn, logs *logrus.Entry) (RedisMetric, error) {

	rMetric, lteUnix, gteUnix, err := redisMagic(conn, logs)
	if err != nil {
		logs.Debug("f: processQuery - redisMagic error: ", err)
		return RedisMetric{}, err
	}

	// Подготовка запроса.
	// Запрос приходит в виде шаблона.
	query, err := parseQuery(rMetric.Query, logs, time.Unix(lteUnix, 0), time.Unix(gteUnix, 0))
	if err != nil {
		logs.Debug("f: processQuery - parseQueryError: ", err)
		return RedisMetric{}, err
	}

	logs.Debug("f: processQuery - parsed query: ", query)

	rMetric.Query = query

	return rMetric, nil
}

// redisMagic забираем из очереди в Redis метрику и делаем все необходимое для предварительной
// обработки.
// На выходе RedisMetric. Веременные ограничения для запроса в Unix формате и ошибка.
func redisMagic(conn redis.Conn, logs *logrus.Entry) (RedisMetric, int64, int64, error) {
	// Читаем из очереди метрику для обработки.

	metric_key, err := conn.Do("LPOP", mfl_list)
	if err != nil {
		logs.Error("f: redisMagic - Redis LPOP error: ", err)
		return RedisMetric{}, 0, 0, err
	}

	// Читаем метрику
	mString, err := redis.String(conn.Do("GET", metric_key))
	if err != nil {
		logs.Error("f: redisMagic - Redis HGETALL error: ", err)
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
		logs.Error("f: redisMagic - Redis HGETALL error: ", err)
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
