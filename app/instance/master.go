// cSpell:disable
package instance

import (
	"errors"
	"strconv"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/sirupsen/logrus"
)

// beMaster Запускаем процедуры мастера.
func (i *Instance) beMaster() error {
	if i.role == MASTER {
		return nil
	}

	i.logs.Info("start master")

	i.role = MASTER

	// Читаем метрики из конфигурационных файлов
	err := i.FillMetrics()
	if err != nil || i.metrics == nil {
		i.logs.Error("Неудалось сформировать массив метрик. ", err)
		// если не удалось прочитать конфигурационный файл - это критическая ошибка.
		// можно выклюать программу.
		return err
	}

	if i.config.Loglevel == "debug" {
		for _, m := range i.metrics {
			i.logs.Debug("{ Метрика: " + m.Mertic + ", запрос: " + m.Query + ", периодичность: " + strconv.Itoa(m.Repeat) + "}")
		}
	}

	// Создаём канал с тикером.
	expireChan := time.NewTicker(time.Millisecond * masterExpire / 2).C
	closeThreads := make(chan bool)

	go i.doMaster(closeThreads)

	for {
		<-expireChan
		if i.expireMaster() {
			close(closeThreads) // завершить процессы обработки постановки в очередь.
			break
		}
	}

	return nil
}

// expireMaster Пытаемся остаться мастером :) Обновляем, если это возможно, мастер ключ.
func (i *Instance) expireMaster() bool {
	conn := i.pool.Get()
	defer conn.Close()

	if _, err := conn.Do("PEXPIRE", masterKey, masterExpire); err != nil {
		i.logs.Info("master expare")
		return true
	}
	return false
}

func (i *Instance) doMaster(abort <-chan bool) {
	// time.Sleep(4000000000)
	// i.logs.Debug("doMaster - TIK")
	var err error = nil

	for _, metric := range i.metrics {
		i.logs.Debug("Start metric: " + metric.Mertic)
		if metric.Metrictype == "counter" {
			go i.ProcessCounter(metric, abort)
		} else {
			err = errors.New("Метрика: " + metric.Mertic + ", тип: " + metric.Metrictype + " не поддерживается.")
			i.logs.Error(err)
		}
	}
}

// ProcessCounter Отсылает в очередь задания на запросы к elasticseaarch.
// metric - метрика которую необходимо обработать.
// labels - обязательные labels метрик.
// abort - если канал закрывается - необходимо завершить работу go программы. Обычно если проигрываем
// выборы мастера/
func (i *Instance) ProcessCounter(metric Metric, abort <-chan bool) {
	// Задержка перед стартом цикла
	time.Sleep(time.Duration(metric.Delay))
	tick := time.NewTicker(time.Duration(metric.Repeat) * time.Second)

	// Если вываливаемся из бесконечного цикла, обязательно останавливаем тикер
	defer tick.Stop()

	for {
		select {
		case <-tick.C:
			// Время выполнения запроса к elasticsearch не должно превышать времени tick
			// Отсылаем запрос на выполнение в очередь.
			i.envelopeSend(&metric)
		case <-abort: // если канал закрылся. Т.е. требуется завершить работу программы.
			return
		}
	}
}

func (i *Instance) envelopeSend(metric *Metric) error {
	conn := i.pool.Get()
	defer conn.Close()

	return send(conn, i.logs, metric)
}

// send Ставит в очередь на выполнение метрики. Посылает уведомление о постановке в канал.
func send(conn redis.Conn, logs *logrus.Entry, metric *Metric) error {

	ti := time.Now().Add(time.Duration(metric.Delay)).Unix()
	key := mfl_metric_prefix + ":" + metric.Mertic + ":" + strconv.Itoa(int(ti))
	redisMetric := RedisMetric{
		Metric:     metric.Mertic,
		Metrichelp: metric.Mertichelp,
		Metrictype: metric.Metrictype,
		Query:      metric.Query,
		Index:      metric.Index,
		Repeat:     strconv.Itoa(metric.Repeat),
	}

	logs.Debug("f: send - Metric: " + metric.Mertic + ", key: " + key)

	// Формируем hash
	_, err := conn.Do("HSET", redis.Args{}.Add(key).AddFlat(redisMetric)...)
	if err != nil {
		logs.Error("f: send - Redis HSET error: ", err)
		return err
	}

	_, err = conn.Do("EXPIRE", key, metric.Repeat*2)
	if err != nil {
		logs.Error("f: send - Redis EXPIRE error: ", err)
		return err
	}

	// Добавляем hash в очередь.
	_, err = conn.Do("LPUSH", mfl_list, key)
	if err != nil {
		logs.Error("f: send - Redis LPUSH error: ", err)
		return err
	}

	// Посылаем уведомление в канал, о добавлениии hash в очередь.
	_, err = conn.Do("PUBLISH", mfl_query, key)
	if err != nil {
		logs.Error("f: send - Redis PUBLISH error: ", err)
		return err
	}

	return nil
}
