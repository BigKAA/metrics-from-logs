// cSpell:disable
package instance

import (
	"errors"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// CreateThreads создаёт набор threads, в которых порисходят запросы к серверу elasticsearch.
func (i *Instance) CreateThreads() error {
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
			counterVec := promauto.NewCounterVec(
				prometheus.CounterOpts{
					Name: metric.Mertic,
					Help: metric.Mertichelp,
				},
				labels,
			)
			go i.ProcessCounter(metric, counterVec)
		} else {
			err = errors.New("Метрика: " + metric.Mertic + ", тип: " + metric.Metrictype + " не поддерживается.")
			i.logs.Error(err)
		}
	}
	return err
}

// ProcessCounter Создает метрики типа Counter
func (i *Instance) ProcessCounter(metric Metric, counterVec *prometheus.CounterVec) {
	// Задержка перед стартом цикла
	time.Sleep(time.Duration(metric.Delay))
	tick := time.NewTicker(time.Duration(metric.Repeat) * time.Second)

	// Если вываливаемся из бесконечного цикла, обязательно останавливаем тикер
	defer tick.Stop()

	for {
		<-tick.C
		// Время выполнения запроса к elasticsearch не должно превышать времени tick
		// Тут должен быть запрос к эластику <==============================================================<<<<<<<====<<<<<<<
		if i.config.K8sPod != "" {
			counterVec.WithLabelValues(i.config.K8sPod, i.config.K8sNamespace, i.config.EsHost).Inc()
		} else {
			counterVec.WithLabelValues(i.config.EsHost).Inc()
		}
		i.logs.Debug("Metric: " + metric.Mertic + " tick")
		// i.logs.Debug("Query result: " + queryResult)
	}

}
