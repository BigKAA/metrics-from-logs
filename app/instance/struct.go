// cSpell:disable
package instance

import (
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

const (
	// VERSION версия программы
	VERSION            = "0.01"
	masterKey          = "mfl_master_key"              // имя ключа для выбора мастера
	masterExpire       = 5000                          // Время устаревания токена в ms (5 с)
	mfl_query          = "mfl_query"                   // имя канала для посылки события
	mfl_list           = "mfl_list"                    // имя листа (очереди) в которую записывает задания мастер.
	mfl_metric_prefix  = "mfl_metric"                  // с чего начинается имя ключа метрики
	expire_prom_metric = time.Duration(24) * time.Hour // время удаление метрики прометея из редиса
)

type role string

type Metric struct {
	Mertic     string             `yaml:"metric"` // Название метрики. Должно быть уникальным.
	Mertichelp string             `yaml:"metrichelp"`
	Metrictype string             `yaml:"metrictype"` // Тип метрики: counter, gauge, histogram, summary
	Query      string             `yaml:"query"`      // Запрос к es
	Index      string             `yaml:"index"`      // Имя индекса (шаблона), в котором происходит поиск
	Repeat     int                `yaml:"repeat"`     // Количество секунд, через сколько повторять запрос
	Delay      int64              `yaml:"delay"`      // Количество секунд, задержка после старта программы перед началом цикла опроса.
	Labels     []PrometheusLabels `redis:"labels"`
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
	Logs    *logrus.Entry
	Config  *Config
	Metrics []Metric
	router  *mux.Router
	Pool    *redis.Pool
	role    role
}

type RedisMetric struct {
	Metric     string             `json:"metric"` // Название метрики. Должно быть уникальным.
	Metrichelp string             `json:"metrichelp"`
	Metrictype string             `json:"metrictype"` // Тип метрики: counter, gauge, histogram, summary
	Query      string             `json:"query"`      // Запрос к es
	Index      string             `json:"index"`      // es index or index pattern
	Repeat     string             `json:"repeat"`     // Периодичность повторения запросов.
	Labels     []PrometheusLabels `json:"labels"`
}

type PrometheusLabels struct {
	Name  string
	Value string
}

// Структура в Redis для формировния метрики в формате Prometheus
type PrometheusMetric struct {
	Metric    string             `redis:"metric"`
	Help      string             `redis:"help"`
	Type      string             `redis:"type"`
	Count     int64              `redis:"count"`
	Timestamp int64              `redis:"timestamp"`
	Labels    []PrometheusLabels `redis:"labels"`
}

const (
	SLAVE  role = "slave"
	MASTER role = "master"
	UNDEF  role = "undefined"
)
