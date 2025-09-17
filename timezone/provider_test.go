package timezone

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestProvider_Vars(t *testing.T) {
	assert.Equal(t, time.UTC, UTC)
	assert.Equal(t, "Asia/Shanghai", PRC.String())
	assert.Equal(t, "Asia/Taipei", Taipei.String())
}

func TestMustLoadLocation(t *testing.T) {
	assert.Panics(t, func() {
		MustLoadLocation("invalid")
	})
	assert.Equal(t, "Asia/Shanghai", MustLoadLocation("Asia/Shanghai").String())
}

func TestProvider(t *testing.T) {
	assert.Equal(t, "Local", time.Local.String())

	p := NewProvider(PRC)

	ctx, err := p.Bootstrap(t.Context())
	assert.NoError(t, err)
	assert.Equal(t, "Asia/Shanghai", time.Local.String())

	l1, ok := FromContext(ctx)
	assert.True(t, ok)
	assert.Equal(t, "Asia/Shanghai", l1.String())

	ctx2, err := p.Terminate(ctx)
	assert.NoError(t, err)
	l2, ok := FromContext(ctx2)
	assert.True(t, ok)
	assert.Equal(t, "Asia/Shanghai", l2.String())
}
