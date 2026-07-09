package capability

type Retriable interface {
	Retries() int
}
