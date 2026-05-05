package errors

import "errors"

var multipleErrors = "multiple errors"

type Group struct {
	errors []error
}

func NewGroup() *Group {
	return &Group{
		errors: make([]error, 0),
	}
}

func (g *Group) Add(errs ...error) *Group {
	for _, err := range errs {
		if err == nil {
			continue
		}

		g.errors = append(g.errors, err)
	}

	return g
}

func (g *Group) Error() string {
	switch len(g.errors) {
	case 0:
		return ""
	case 1:
		return g.errors[0].Error()
	default:
		return multipleErrors
	}
}

func (g *Group) Errors() []error {
	return g.errors
}

func (g *Group) Len() int {
	return len(g.errors)
}

func (g *Group) Has(err error) bool {
	for _, e := range g.errors {
		if errors.Is(e, err) {
			return true
		}
	}

	return false
}

func (g *Group) First() error {
	if len(g.errors) == 0 {
		return nil
	}

	return g.errors[0]
}

func (g *Group) IsNil() bool {
	return len(g.errors) == 0
}
