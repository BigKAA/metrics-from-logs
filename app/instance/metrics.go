// cSpell:disable
package instance

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// FillMetrics Заполняем структуру информацией о метриках
func FillMetrics(logs *logrus.Entry, config *Config) ([]Metric, error) {
	// Получаем список файлов в директории.
	files, _ := filepath.Glob(config.Confd + "\\*.yaml")

	var metrics []Metric

	for _, fileName := range files {
		confFile, err := ioutil.ReadFile(fileName)
		if err != nil {
			return nil, err
		}

		var metric Metric
		err = yaml.Unmarshal(confFile, &metric)
		if err != nil {
			return nil, err
		}
		if IsMetricExist(metric, metrics) {
			logs.Warnf("Не уникальная метрика %s в файле %s. Пропускаем. ", metric.Mertic, fileName)
			continue
		}
		// Сожмем метрику. Удалим лишние пролбелы.
		src := []byte(metric.Query)
		dst := bytes.NewBuffer([]byte{})
		if err = json.Compact(dst, src); err != nil {

		}

		metrics = append(metrics, metric)
	}

	return metrics, nil
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
