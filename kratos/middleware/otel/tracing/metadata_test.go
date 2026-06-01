package tracing

import (
	"context"
	"testing"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/metadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/propagation"
)

func TestMetadata_Inject(t *testing.T) {
	type args struct {
		appName string
		carrier propagation.TextMapCarrier
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "https://go-kratos.dev",
			args: args{"https://go-kratos.dev", propagation.HeaderCarrier{}},
			want: "https://go-kratos.dev",
		},
		{
			name: "https://github.com/go-kratos/kratos",
			args: args{"https://github.com/go-kratos/kratos", propagation.HeaderCarrier{"mode": []string{"test"}}},
			want: "https://github.com/go-kratos/kratos",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := kratos.New(kratos.Name(tt.args.appName))
			ctx := kratos.NewContext(t.Context(), a)
			m := new(Metadata)
			m.Inject(ctx, tt.args.carrier)
			assert.Equal(t, tt.want, tt.args.carrier.Get(serviceHeader))
		})
	}
}

func TestMetadata_Extract(t *testing.T) {
	type args struct {
		parent  context.Context
		carrier propagation.TextMapCarrier
	}
	tests := []struct {
		name  string
		args  args
		want  string
		crash bool
	}{
		{
			name: "https://go-kratos.dev",
			args: args{
				parent:  t.Context(),
				carrier: propagation.HeaderCarrier{"X-Md-Service-Name": []string{"https://go-kratos.dev"}},
			},
			want: "https://go-kratos.dev",
		},
		{
			name: "https://github.com/go-kratos/kratos",
			args: args{
				parent:  metadata.NewServerContext(t.Context(), metadata.Metadata{}),
				carrier: propagation.HeaderCarrier{"X-Md-Service-Name": []string{"https://github.com/go-kratos/kratos"}},
			},
			want: "https://github.com/go-kratos/kratos",
		},
		{
			name: "https://github.com/go-kratos/kratos",
			args: args{
				parent:  metadata.NewServerContext(t.Context(), metadata.Metadata{}),
				carrier: propagation.HeaderCarrier{"X-Md-Service-Name": nil},
			},
			crash: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := Metadata{}
			ctx := b.Extract(tt.args.parent, tt.args.carrier)
			md, ok := metadata.FromServerContext(ctx)
			if !ok {
				if tt.crash {
					return
				}
				require.True(t, ok)
			}
			assert.Equal(t, tt.want, md.Get(serviceHeader))
		})
	}
}

func TestFields(t *testing.T) {
	b := Metadata{}
	assert.Equal(t, []string{"x-md-service-name"}, b.Fields())
}
