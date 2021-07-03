// cSpell:disable
package instance

import (
	"context"
	"html/template"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/felixge/httpsnoop"
	"github.com/sirupsen/logrus"
)

// Start start app
func (i *Instance) Start() error {

	abort := make(chan bool)

	go i.masterElection(abort)

	// Основной процесс обработки метрик
	go i.worker()

	// запускаем http сервер отдельно go программой.
	httpServerExitDone := &sync.WaitGroup{}
	httpServerExitDone.Add(1)
	srv := i.doHttp(httpServerExitDone)

	// если go программа закроет канал, значит надо завершать приложение.
	<-abort

	if err := srv.Shutdown(context.TODO()); err != nil {
		i.Logs.Error(err) // failure/timeout shutting down the server gracefully
	}

	httpServerExitDone.Wait()

	return nil
}

// doHttp запус http сервера.
func (i *Instance) doHttp(wg *sync.WaitGroup) *http.Server {
	i.ConfigRouter()
	i.Logs.Info("Starting http server Listen on: http://", i.Config.Bindaddr, i.Config.Context)

	srv := &http.Server{
		Addr:    i.Config.Bindaddr,
		Handler: i.logRequestHandler(i.router),
	}

	go func() {
		defer wg.Done() // let main know we are done cleaning up

		// always returns error. ErrServerClosed on graceful close
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			// unexpected error. port in use?
			i.Logs.Error("ListenAndServe(): ", err)
		}
	}()

	// returning reference so caller can call Shutdown()
	return srv
}

// ConfigRouter конфигурирует роутер
func (i *Instance) ConfigRouter() {
	h := i.router.PathPrefix(i.Config.Context).Subrouter()
	h.HandleFunc("/", i.HandlerRoot())
	h.HandleFunc("/metrics/", i.HandlerMetrics())
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
		i.Logs.WithFields(logrus.Fields{
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
func (i *Instance) masterElection(abort chan bool) {
	for {
		conn := i.Pool.Get()
		defer conn.Close()

		// id - значение в редисе для идентификации мастера. Присваивается masterKey
		var id string
		if i.Config.K8sPod != "" {
			// если работаем в кубере, то id содержит имя пода
			id = i.Config.K8sPod
		} else {
			id = strconv.Itoa(os.Getpid()) // pid на разных машинах может совпадать...   <----- ????
		}

		// Пытаемся присвоить masterKey значение id данного приложения.
		current, err := conn.Do("SET", masterKey, id, "NX", "PX", masterExpire)
		if err != nil {
			i.Logs.Error("Redis connection error: ", err)
			return
		}

		if i.Config.K8sPod != "" {
			i.Logs.Info("Выборы мастера - pod: ", id, " результат: ", current)
		} else {
			i.Logs.Info("Выборы мастера - pid: ", id, " результат: ", current)
		}

		// Если значение присвоить удалось - то это мастер.

		if current == "OK" {
			if err := i.beMaster(); err != nil {
				// совсем плохо с конфигурацией
				// сигнализируем другим go программам о завершении приложения.
				close(abort)
				break
			}
		} else {
			i.Logs.Info("Start slave")

			i.role = SLAVE

			electionChan := time.NewTicker(time.Millisecond * masterExpire).C

			for {
				<-electionChan
				break
			}
		}
	}
}
