package delivery

type Broadcaster interface {
	Broadcast(destination string, message []byte)
}
