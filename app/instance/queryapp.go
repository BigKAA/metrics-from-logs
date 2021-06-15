// cSpell:disable
package instance

import "time"

// CreateThreads создаёт набор threads, в которых порисходят запросы к серверу elasticsearch.
func (i *Instance) CreateThreads() error {
	for _, metric := range i.metrics {
		i.logs.Debug("Start metric: " + metric.Mertic)

		go func(me *Metric) {
			// Задержка перед стартом цикла
			time.Sleep(time.Duration(me.Delay))
			tick := time.NewTicker(time.Duration(me.Repeat) * time.Second)
			// Если вываливаемся из бесконечного цикла, обязательно останавливаем тикер
			defer tick.Stop()

			for {
				<-tick.C
				// Время выполнения запроса к elasticsearch не должно превышать времени tick
				queryResult, err := i.Query(me)
				if err != nil {
					i.logs.Error(err)
				}
				i.logs.Debug("Query result: " + queryResult)
			}
		}(&metric)
	}
	return nil
}

// Query выполняет запрос к elasticsearch. Если запрос не удался - возвращает ошибку.
func (i *Instance) Query(metric *Metric) (string, error) {
	i.logs.Debug("Metric: " + metric.Mertic + " tick")
	return "query result", nil
}
