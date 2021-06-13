package instance

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// VERSION версия программы
const VERSION = "0.01"

type Metric struct {
	Mertic         string `yaml:"metric"`         // Название метрики. Должно быть уникальным.
	Metrictype     string `yaml:"metrictype"`     // Тип метрики: counter, gauge, histogram, summary
	Esserver       string `yaml:"esserver"`       // es.any.com
	Esserverport   int    `yaml:"esserverport"`   // es port
	Esuserfrom     string `yaml:"esuserfrom"`     // одно из трёх значений: conf, file, env
	Esuser         string `yaml:"esuser"`         // Пользователь
	Espasswordfrom string `yaml:"espasswordfrom"` // одно из трёх значений: conf, file, env
	Espassword     string `yaml:"espassword"`     // пароль
	Query          string `yaml:"query"`          // Запрос к es
	Repeat         int    `yaml:"repeat"`         // Количество секунд, через сколько повторять запрос
}

// Config параметеры из конфигурационного файла программы.
type Config struct {
	Main struct {
		Confd    string `yaml:"confd"` // Директория с конфигурационными файлами с запросами к es
		Loglevel string `yaml:"loglevel"`
	} `yaml:"main"`
}

// Instance ...
type Instance struct {
	logs    *logrus.Logger
	config  *Config
	metrics []Metric
}

// NewInstance Создаёт приложение
func NewInstance() *Instance {
	var configFileName string
	showVersion := false

	flag.StringVar(&configFileName, "c", "conf.yaml", "Config file")
	flag.BoolVar(&showVersion, "v", false, "Show version")
	flag.Parse()

	if showVersion {
		fmt.Print("Version: ", VERSION, "\n")
		return nil
	}

	// Читаем основной конфигурационный файл.
	config, err := ReadConfigFile(configFileName)
	if err != nil {
		fmt.Print("Не могу загрузить информацию из конфигурационного файла. ", err)
	}

	logs := logrus.New()
	logs.SetFormatter(&logrus.JSONFormatter{})

	// Устанавливаем уровень важности сообщений, выводимых logrus
	level, err := logrus.ParseLevel(config.Main.Loglevel)
	if err != nil {
		logs.Warn("Не правильно определён loglevel в конфигурационном файле. Установлен уровень по умолчанию debug.")
		level, _ = logrus.ParseLevel("debug")
	}
	logs.SetLevel(level)

	// Проверяем наличие конфигурационной директории
	if config.Main.Confd == "" {
		logs.Error("Не определена директория для файлов с описанием Метрик.")
		return nil
	}
	ret, err := exists(config.Main.Confd)
	if !ret || err != nil {
		logs.Error("Директория " + config.Main.Confd + " не существует.")
		return nil
	}

	metrics, err := FillMetrics(config.Main.Confd, logs)
	if err != nil || metrics == nil {
		logs.Error("Неудалось сформировать массив метрик.")
		return nil
	}

	for _, m := range metrics {
		logs.Debug("{ Метрика: " + m.Mertic + ", запрос: " + m.Query + ", периодичность: " + strconv.Itoa(m.Repeat) + "}")
	}

	return &Instance{
		logs:    logs,
		config:  config,
		metrics: metrics,
	}
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

// ReadConfigFile Читаем конфигурацию из конфигурационного файла и формируем
// структуру Conf
func ReadConfigFile(fileName string) (*Config, error) {
	// Чтение конфигурационного файла
	confFile, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	var conf Config
	err = yaml.Unmarshal(confFile, &conf)
	if err != nil {
		return nil, err
	}

	return &conf, nil
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
