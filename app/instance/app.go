package instance

// Start start app
func (i *Instance) Start() {
	i.logs.Debug("Start application")
}

// ConfigRouter конфигурирует роутер
// func (i *Instance) ConfigRouter() {
// 	i.router.HandleFunc("/", i.HandlerRoot())
// 	pods := s.router.PathPrefix("/pods").Subrouter()
// 	pods.HandleFunc("/", s.HandlerRoot())
// 	pods.HandleFunc("/{ns}", s.HandlerPods())
// }