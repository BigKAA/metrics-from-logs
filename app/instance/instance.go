// cSpell:disable
package instance

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// VERSION версия программы
const VERSION = "0.01"

type Metric struct {
	Mertic     string `yaml:"metric"` // Название метрики. Должно быть уникальным.
	Mertichelp string `yaml:"metrichelp"`
	Metrictype string `yaml:"metrictype"` // Тип метрики: counter, gauge, histogram, summary
	Query      string `yaml:"query"`      // Запрос к es
	Repeat     int    `yaml:"repeat"`     // Количество секунд, через сколько повторять запрос
	Delay      int    `yaml:"delay"`      // Количество секунд, задержка после старта программы перед началом цикла опроса.
}

// Config параметеры из конфигурационного файла программы.
type Config struct {
	Confd         string // Директория с конфигурационными файлами с запросами к es
	Loglevel      string
	Bindaddr      string
	Context       string
	EsHost        string
	EsPort        string
	EsUser        string
	EsPassword    string
	K8sPod        string
	K8sNamespace  string
	RedisServer   string
	RedisPort     string
	RedisPassword string
}

// Instance ...
type Instance struct {
	logs    *logrus.Entry
	config  *Config
	metrics []Metric
	router  *mux.Router
	pool    *redis.Pool
	role    role
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

	config := &Config{
		Confd:         getEnv("MFL_CONF_DIR", "etc\\mfl\\conf.d\\"),
		Loglevel:      getEnv("MFL_LOG_LEVEL", "debug"),
		Bindaddr:      getEnv("MFL_BIND_ADDR", "127.0.0.1:8080"),
		Context:       getEnv("MFL_CONTEXT", "/"),
		EsHost:        getEnv("MFL_ES_HOST", "127.0.0.1"),
		EsPort:        getEnv("MFL_ES_PORT", "9200"),
		EsUser:        getEnv("MFL_ES_USER", "user"),
		EsPassword:    getEnv("MFL_ES_PASSWORD", "password"),
		K8sPod:        getEnv("MFL_K8S_POD", ""),
		K8sNamespace:  getEnv("MFL_K8S_NAMESPACE", ""),
		RedisServer:   getEnv("MFL_REDIS_SERVER", "127.0.0.1"),
		RedisPort:     getEnv("MFL_REDIS_PORT", "6379"),
		RedisPassword: getEnv("MFL_REDIS_PASSWORD", ""),
	}

	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	// Устанавливаем уровень важности сообщений, выводимых logrus
	level, err := logrus.ParseLevel(config.Loglevel)
	if err != nil {
		logger.Warn("Не правильно определён loglevel в конфигурационном файле. Установлен уровень по умолчанию debug.")
		level, _ = logrus.ParseLevel("debug")
	}
	logger.SetLevel(level)

	// Подставляем в логи дополнительные поля для k8s и для простого приложения.
	var logs *logrus.Entry
	if config.K8sPod != "" {
		logs = logger.WithFields(logrus.Fields{
			"k8s_pod":       config.K8sPod,
			"k8s_namespace": config.K8sNamespace,
		})
	} else {
		logs = logger.WithFields(logrus.Fields{
			"pid": strconv.Itoa(os.Getpid()),
		})
	}

	// Проверяем наличие директории с конф файлами метрик
	ret, err := exists(config.Confd)
	if !ret || err != nil {
		logs.Error("Директория " + config.Confd + " не существует.")
		return nil
	}

	// Читаем метрики из конфигурационных файлов
	metrics, err := FillMetrics(config.Confd, logs)
	if err != nil || metrics == nil {
		logs.Error("Неудалось сформировать массив метрик.")
		return nil
	}

	if config.Loglevel == "debug" {
		for _, m := range metrics {
			logs.Debug("{ Метрика: " + m.Mertic + ", запрос: " + m.Query + ", периодичность: " + strconv.Itoa(m.Repeat) + "}")
		}
	}

	instance := &Instance{
		logs:    logs,
		config:  config,
		metrics: metrics,
		router:  mux.NewRouter(),
		pool:    newRedisPool(config.RedisServer+":"+config.RedisPort, config.RedisPassword),
	}

	return instance
}

// FillMetrics Заполняем структуру информацией о метриках
func FillMetrics(dirPath string, logs *logrus.Entry) ([]Metric, error) {
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

// newRedisPool Создаёт новый pool подключений к Redis
func newRedisPool(addr string, password string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		// Dial or DialContext must be set. When both are set, DialContext takes precedence over Dial.
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", addr)
			if err != nil {
				return nil, err
			}
			if password != "" {
				if _, err := c.Do("AUTH", password); err != nil {
					c.Close()
					return nil, err
				}
			}
			// if _, err := c.Do("SELECT", db); err != nil {
			// 	c.Close()
			// 	return nil, err
			// }
			return c, nil
		},
	}
}
