package instance

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// VERSION версия программы
const VERSION = "0.01"

type Metric struct {
	Mertic       string `yaml:"metric"`       // Название метрики. Должно быть уникальным.
	Metrictype   string `yaml:"metrictype"`   // Тип метрики: counter, gauge, histogram, summary
	Esserver     string `yaml:"esserver"`     // es.any.com
	Esserverport int    `yaml:"esserverport"` // es port
	Esuser       string `yaml:"esuser"`       // Пользователь
	Espassword   string `yaml:"espassword"`   // пароль
	Query        string `yaml:"query"`        // Запрос к es
	Repeat       int    `yaml:"repeat"`       // Количество секунд, через сколько повторять запрос
}

// Config параметеры из конфигурационного файла программы.
type Config struct {
	Confd    string // Директория с конфигурационными файлами с запросами к es
	Loglevel string
	Bindaddr string
}

// Instance ...
type Instance struct {
	logs    *logrus.Logger
	config  *Config
	metrics []Metric
	router  *mux.Router
}

// NewInstance Создаёт приложение
func NewInstance() *Instance {
	showVersion := false

	flag.BoolVar(&showVersion, "v", false, "Show version")
	flag.Parse()

	if showVersion {
		fmt.Print("Version: ", VERSION, "\n")
		return nil
	}

	logs := logrus.New()
	logs.SetFormatter(&logrus.JSONFormatter{})

	config := &Config{
		Confd:    getEnv("MFL_CONF_DIR", "etc\\mfl\\conf.d\\"),
		Loglevel: getEnv("MFL_LOG_LEVEL", "debug"),
		Bindaddr: getEnv("MFL_BIND_ADDR", "127.0.0.1:8080"),
	}

	// Устанавливаем уровень важности сообщений, выводимых logrus
	level, err := logrus.ParseLevel(config.Loglevel)
	if err != nil {
		logs.Warn("Не правильно определён loglevel в конфигурационном файле. Установлен уровень по умолчанию debug.")
		level, _ = logrus.ParseLevel("debug")
	}
	logs.SetLevel(level)

	ret, err := exists(config.Confd)
	if !ret || err != nil {
		logs.Error("Директория " + config.Confd + " не существует.")
		return nil
	}

	metrics, err := FillMetrics(config.Confd, logs)
	if err != nil || metrics == nil {
		logs.Error("Неудалось сформировать массив метрик.")
		return nil
	}

	for _, m := range metrics {
		logs.Debug("{ Метрика: " + m.Mertic + ", запрос: " + m.Query + ", периодичность: " + strconv.Itoa(m.Repeat) + "}")
	}

	instance := &Instance{
		logs:    logs,
		config:  config,
		metrics: metrics,
		router:  mux.NewRouter(),
	}

	return instance
}

// FillMetrics Заполняем структуру информацией о метриках
func FillMetrics(dirPath string, logs *logrus.Logger) ([]Metric, error) {
	files, _ := filepath.Glob(dirPath + "*.yaml")
	var ret []Metric
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
		if IsMetricExist(metric, ret) {
			logs.Error("Не уникальная реплика " + metric.Mertic + " в файле " + fileName)
			continue
		}
		ret = append(ret, metric)
	}
	return ret, nil
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

// exists returns whether the given file or directory exists
func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// Simple helper function to read an environment or return a default value
func getEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return defaultVal
}
