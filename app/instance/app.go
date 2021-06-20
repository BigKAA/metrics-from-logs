// cSpell:disable
package instance

import (
	"html/template"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/felixge/httpsnoop"
	"github.com/sirupsen/logrus"
)

const (
	masterKey    = "master_key"
	masterExpire = 10000 // Время уствревания токена в ms (10 с)
)

type role string

const (
	SLAVE  role = "slave"
	MASTER role = "master"
)

// Start start app
func (i *Instance) Start() error {
	isMasterChannel := make(chan bool)

	go i.masterElection(isMasterChannel)

	i.ConfigRouter()
	i.logs.Info("Starting API server Listen on: http://", i.config.Bindaddr, i.config.Context)

	server := &http.Server{
		Addr:    i.config.Bindaddr,
		Handler: i.logRequestHandler(i.router),
	}

	return server.ListenAndServe()
}

// ConfigRouter конфигурирует роутер
func (i *Instance) ConfigRouter() {
	h := i.router.PathPrefix(i.config.Context).Subrouter()
	h.HandleFunc("/", i.HandlerRoot())
}

// HandlerRoot Обработка запроса /
func (i *Instance) HandlerRoot() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		templ, _ := template.ParseFiles("templates/index.html")
		index := "index"
		templ.Execute(w, index)
	}
}

// logRequestHandler логирует запросы
func (i *Instance) logRequestHandler(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := httpsnoop.CaptureMetrics(h, w, r)
		i.logs.WithFields(logrus.Fields{
			"method":         r.Method,
			"remote_address": r.RemoteAddr,
			"request_uri":    r.RequestURI,
			"user_agent":     r.UserAgent(),
			"status":         m.Code,
			"bytes":          m.Written,
			"duration":       m.Duration,
		}).Info("request")
	}
}

// masterElection Выборы мастера
func (i *Instance) masterElection(is chan bool) {
	conn := i.pool.Get()
	defer conn.Close()

	var id string
	if i.config.K8sPod != "" {
		id = i.config.K8sPod
	} else {
		id = strconv.Itoa(os.Getpid()) // pid на разных машинах может совпадать...   <----- ????
	}

	current, err := conn.Do("SET", masterKey, id, "NX", "PX", masterExpire)
	if err != nil {
		i.logs.Error("Redis connection error: ", err)
		return
	}

	if i.config.K8sPod != "" {
		i.logs.Info("Выборы мастера - pod: ", id, " результат: ", current)
	} else {
		i.logs.Info("Выборы мастера - pid: ", id, " результат: ", current)
	}

	if current == "OK" {
		i.beMaster(is)
	} else {
		i.beSlave(is)
	}
}

func (i *Instance) beMaster(is chan bool) {

	if i.role == MASTER {
		return
	}

	i.logs.Info("start master")

	i.role = MASTER

	expireChan := time.NewTicker(time.Millisecond * masterExpire / 2).C

	for {
		<-expireChan
		i.expireMaster()
	}
}

func (i *Instance) expireMaster() {
	conn := i.pool.Get()
	defer conn.Close()

	if _, err := conn.Do("PEXPIRE", masterKey, masterExpire); err != nil {
		i.logs.Error("EXPIRE MASTER ERROR: ", err)
	}
}

func (i *Instance) beSlave(is chan bool) {
	if i.role == SLAVE {
		return
	}

	i.logs.Info("Start slave")
	i.role = SLAVE

	electionChan := time.NewTicker(time.Millisecond * masterExpire).C

	for {
		<-electionChan
		i.masterElection(is)
	}
}
