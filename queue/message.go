package queue

type Message struct {
	ID         string
	Queue      string
	Meta       map[string][]string
	Subject    string
	Data       []byte
	MaxRetries int
	Retries    int
}
