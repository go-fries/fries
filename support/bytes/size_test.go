package bytes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBytes_Humanize(t *testing.T) {
	tests := []struct {
		name string
		b    Bytes
		want string
	}{
		{
			name: "test B",
			b:    Bytes(1000),
			want: "1000.00B",
		},
		{
			name: "test KB",
			b:    Bytes(1024),
			want: "1.00KB",
		},
		{
			name: "test MB",
			b:    Bytes(1024 * 1024),
			want: "1.00MB",
		},
		{
			name: "test GB",
			b:    Bytes(1024 * 1024 * 1024),
			want: "1.00GB",
		},
		{
			name: "test TB",
			b:    Bytes(1024 * 1024 * 1024 * 1024),
			want: "1.00TB",
		},
		{
			name: "test PB",
			b:    Bytes(1024 * 1024 * 1024 * 1024 * 1024),
			want: "1.00PB",
		},
		{
			name: "test EB",
			b:    Bytes(1024 * 1024 * 1024 * 1024 * 1024 * 1024),
			want: "1.00EB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.b.Humanize())
			assert.Equal(t, tt.want, tt.b.Humanize(2))
		})
	}
}

func TestBytes_HumanizeLen(t *testing.T) {
	assert.Equal(t, "1.00KB", Bytes(1024).Humanize())
	assert.Equal(t, "1KB", Bytes(1024).Humanize(0))
	assert.Equal(t, "1.0KB", Bytes(1024).Humanize(1))
	assert.Equal(t, "1.00KB", Bytes(1024).Humanize(2))
	assert.Equal(t, "1.000KB", Bytes(1024).Humanize(3))
}
