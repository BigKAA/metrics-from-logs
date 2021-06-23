// cSpell:disable
package instance

import (
	"errors"
	"strconv"
	"time"
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
	var labels []string

	if i.config.K8sPod != "" {
		labels = []string{"pod", "namespace", "es_host"} // названия labels в метриках
	} else {
		labels = []string{"es_host"}
	}

	for _, metric := range i.metrics {
		i.logs.Debug("Start metric: " + metric.Mertic)
		if metric.Metrictype == "counter" {
			go i.ProcessCounter(metric, labels, abort)
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
func (i *Instance) ProcessCounter(metric Metric, labels []string, abort <-chan bool) {
	// Задержка перед стартом цикла
	time.Sleep(time.Duration(metric.Delay))
	tick := time.NewTicker(time.Duration(metric.Repeat) * time.Second)

	// Если вываливаемся из бесконечного цикла, обязательно останавливаем тикер
	defer tick.Stop()

	for {
		select {
		case <-tick.C:
			// Время выполнения запроса к elasticsearch не должно превышать времени tick
			// Тут должен быть запрос к эластику <==============================================================<<<<<<<====<<<<<<<
			// i.logs.Debug("Query result: " + queryResult)

			// добавление|тик метрики в Redis
			if i.config.K8sPod != "" {
				i.logs.Debug("Metric: "+metric.Mertic+" tick ", i.config.K8sPod, i.config.K8sNamespace, i.config.EsHost)
				//
			} else {
				i.logs.Debug("Metric: "+metric.Mertic+" tick ", i.config.EsHost)
				//
			}
		case <-abort: // если канал закрылся. Т.е. требуется завершить работу программы.
			return
		}
	}
}
