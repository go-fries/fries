package queue

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConsumerConfig_Normalize(t *testing.T) {
	tests := []struct {
		name string
		in   ConsumerConfig
		want ConsumerConfig
	}{
		{
			name: "default queue",
			in:   ConsumerConfig{Name: "worker-1"},
			want: ConsumerConfig{Queue: DefaultQueue, Name: "worker-1"},
		},
		{
			name: "keeps queue",
			in:   ConsumerConfig{Queue: "emails", Name: "worker-1"},
			want: ConsumerConfig{Queue: "emails", Name: "worker-1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.in.Normalize())
		})
	}
}
