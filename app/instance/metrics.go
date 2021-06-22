// cSpell:disable
package instance

import (
	"io/ioutil"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// FillMetrics Заполняем структуру информацией о метриках
func (i *Instance) FillMetrics() error {
	// Получаем список файлов в директории.
	files, _ := filepath.Glob(i.config.Confd + "*.yaml")

	for _, fileName := range files {
		confFile, err := ioutil.ReadFile(fileName)
		if err != nil {
			return err
		}
		var metric Metric
		err = yaml.Unmarshal(confFile, &metric)
		if err != nil {
			return err
		}
		if IsMetricExist(metric, i.metrics) {
			i.logs.Error("Не уникальная реплика " + metric.Mertic + " в файле " + fileName + ". Пропускаем.")
			continue
		}
		i.metrics = append(i.metrics, metric)
	}

	return nil
}

// IsMetricExist проверяет, есть ли метрика с таким именем в массиве
func IsMetricExist(metric Metric, metrics []Metric) bool {
	// Если массив пустой, то и метрика уникальная
	if len(metrics) == 0 {
		return false
	}
	for _, m := range metrics {
		if m.Mertic == metric.Mertic {
			return true
		}
	}
	return false
}
