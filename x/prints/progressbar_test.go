package prints

import (
	"testing"

	"github.com/cheggaaa/pb/v3"
	"github.com/stretchr/testify/assert"
)

func TestNewProgressBar(t *testing.T) {
	assert.Equal(t, pb.Full, Full)
	assert.Equal(t, pb.Simple, Simple)
	assert.Equal(t, pb.Default, Default)

	p1 := NewProgressBar(100)
	for i := 1; i <= 100; i++ {
		p1.Increment()
	}
	p1.Finish()

	p2 := NewProgressBar(100, WithTemplate(Full))
	for i := 1; i <= 100; i++ {
		p2.Increment()
	}
	p2.Finish()
}

func TestWithProgressBar(*testing.T) {
	WithProgressBar(100, func(p *ProgressBar) {
		for i := 1; i <= 100; i++ {
			p.Increment()
		}
	})
}
