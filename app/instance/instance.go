// cSpell:disable
package instance

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// NewInstance Создаёт приложение
func NewInstance() *Instance {
	showVersion := false

	flag.BoolVar(&showVersion, "v", false, "Show version")
	flag.Parse()

	if showVersion {
		fmt.Print("Version: ", VERSION, "\n")
		return nil
	}

	config := GetConfig()

	logs := GetLogEntry(config)

	instance, err := GetInstance(config, logs)
	if err != nil {
		logs.Error("f: GetInstance - :", err)
	}

	return instance
}

func GetInstance(config *Config, logs *logrus.Entry) (*Instance, error) {
	// Проверяем наличие директории с конф файлами метрик
	ret, err := exists(config.Confd)
	if !ret || err != nil {
		return nil, errors.New("Директория " + config.Confd + " не существует.")
	}

	instance := &Instance{
		Logs:    logs,
		Config:  config,
		Metrics: nil,
		router:  mux.NewRouter(),
		Pool:    newRedisPool(config.RedisServer+":"+config.RedisPort, config.RedisPassword),
		role:    UNDEF,
	}

	return instance, nil
}

func GetLogEntry(config *Config) *logrus.Entry {
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

	return logs
}

func GetConfig() *Config {
	return &Config{
		Confd:         getEnv("MFL_CONF_DIR", "/etc/mfl/conf.d/"),
		Loglevel:      getEnv("MFL_LOG_LEVEL", "info"),
		Bindaddr:      getEnv("MFL_BIND_ADDR", "0.0.0.0:8080"),
		Context:       getEnv("MFL_CONTEXT", "/"),
		EsHost:        getEnv("MFL_ES_HOST", "http://127.0.0.1"),
		EsPort:        getEnv("MFL_ES_PORT", "9200"),
		EsUser:        getEnv("MFL_ES_USER", "user"),
		EsPassword:    getEnv("MFL_ES_PASSWORD", "password"),
		K8sPod:        getEnv("MFL_K8S_POD", ""),
		K8sNamespace:  getEnv("MFL_K8S_NAMESPACE", ""),
		RedisServer:   getEnv("MFL_REDIS_SERVER", "127.0.0.1"),
		RedisPort:     getEnv("MFL_REDIS_PORT", "6379"),
		RedisPassword: getEnv("MFL_REDIS_PASSWORD", ""),
	}
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
