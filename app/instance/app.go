// cSpell:disable
package instance

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
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
	UNDEF  role = "undefined"
)

// Start start app
func (i *Instance) Start() error {

	abort := make(chan bool)

	go i.masterElection(abort)

	httpServerExitDone := &sync.WaitGroup{}
	httpServerExitDone.Add(1)
	srv := i.doHttp(httpServerExitDone)

	for {
		select {
		case <-abort:
			break
		}
	}

	if err := srv.Shutdown(context.TODO()); err != nil {
		i.logs.Error(err) // failure/timeout shutting down the server gracefully
	}

	httpServerExitDone.Wait()

	return nil
}

// doHttp запус http сервера.
func (i *Instance) doHttp(wg *sync.WaitGroup) *http.Server {
	i.ConfigRouter()
	i.logs.Info("Starting http server Listen on: http://", i.config.Bindaddr, i.config.Context)

	srv := &http.Server{
		Addr:    i.config.Bindaddr,
		Handler: i.logRequestHandler(i.router),
	}

	go func() {
		defer wg.Done() // let main know we are done cleaning up

		// always returns error. ErrServerClosed on graceful close
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			// unexpected error. port in use?
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()

	// returning reference so caller can call Shutdown()
	return srv
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
func (i *Instance) masterElection(errorAbortChan chan<- bool) {
	for {
		conn := i.pool.Get()
		defer conn.Close()

		// id - значение в редисе для идентификации мастера. Присваивается masterKey
		var id string
		if i.config.K8sPod != "" {
			// если работаем в кубере, то id содержит имя пода
			id = i.config.K8sPod
		} else {
			id = strconv.Itoa(os.Getpid()) // pid на разных машинах может совпадать...   <----- ????
		}

		// Пытаемся присвоить masterKey значение id данного приложения.
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

		// Если значение присвоить удалось - то это мастер.

		if current == "OK" {
			if err := i.beMaster(); err != nil {
				close(errorAbortChan)
				break // совсем плохо с конфигурацией
			}
		} else {
			i.logs.Info("Start slave")

			i.role = SLAVE

			electionChan := time.NewTicker(time.Millisecond * masterExpire).C

			for {
				<-electionChan
				break
			}
		}
	}
}
