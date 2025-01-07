package spt

type Proxy[T any] struct {
	value T
}

func NewProxy[T any](value T) *Proxy[T] {
	return &Proxy[T]{
		value: value,
	}
}

func (p *Proxy[T]) Tap(callbacks ...func(T)) *Proxy[T] {
	for _, callback := range callbacks {
		if callback != nil {
			callback(p.value)
		}
	}
	return p
}

func (p *Proxy[T]) With(callbacks ...func(T) T) *Proxy[T] {
	for _, callback := range callbacks {
		if callback != nil {
			p.value = callback(p.value)
		}
	}
	return p
}

func (p *Proxy[T]) When(condition bool, callbacks ...func(T) T) *Proxy[T] {
	if condition {
		return p.With(callbacks...)
	}
	return p
}

func (p *Proxy[T]) Unless(condition bool, callbacks ...func(T) T) *Proxy[T] {
	return p.When(!condition, callbacks...)
}

func (p *Proxy[T]) Transform(callback func(T) T) *Proxy[T] {
	if callback != nil {
		p.value = callback(p.value)
	}
	return p
}

func (p *Proxy[T]) Value() T {
	return p.value
}
