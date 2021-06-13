package instance

import (
	"html/template"
	"net/http"

	"github.com/felixge/httpsnoop"
	"github.com/sirupsen/logrus"
)

// Start start app
func (i *Instance) Start() error {
	i.ConfigRouter()
	i.logs.Info("Starting API server Listen on: http://", i.config.Bindaddr)

	server := &http.Server{
		Addr:    i.config.Bindaddr,
		Handler: i.logRequestHandler(i.router),
		// ErrorLog: s.logger,
	}

	return server.ListenAndServe()
}

// ConfigRouter конфигурирует роутер
func (i *Instance) ConfigRouter() {
	i.router.HandleFunc("/", i.HandlerRoot())
	// pods := s.router.PathPrefix("/pods").Subrouter()
	// pods.HandleFunc("/", s.HandlerRoot())
	// pods.HandleFunc("/{ns}", s.HandlerPods())
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
