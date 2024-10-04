package publisher

type MessagePublisher interface {
	SendMessage(destination, source, message string) error
	Close()
}
