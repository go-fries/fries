package contract

type Retriable interface {
	Retries() int
}
