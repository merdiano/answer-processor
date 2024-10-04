package delivery

import "net/http"

type Handler interface {
	HandleConnections(w http.ResponseWriter, r *http.Request)
	HandleMessages()
	Shutdown()
	Broadcaster
}
