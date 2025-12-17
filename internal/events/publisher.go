package events

type Publisher interface {
	Publish(channel string, message interface{}) error
}
