package capability

type Asyncable interface {
	Async() bool
}

type AsyncFeature struct{}

func (*AsyncFeature) Async() bool {
	return true
}
