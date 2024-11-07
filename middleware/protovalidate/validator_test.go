package protovalidate

import (
	"context"
	"testing"

	"github.com/bufbuild/protovalidate-go"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"

	pb "github.com/go-kratos-ecosystem/components/v2/internal/genproto/tests/middleware/protovalidate/v1"
)

var next = func(context.Context, any) (reply any, err error) {
	return "reply", nil
}

func defaultValidator(t *testing.T) *protovalidate.Validator {
	validator, err := protovalidate.New(
		protovalidate.WithFailFast(true),
	)
	assert.NoError(t, err)
	return validator
}

func defaultWant(t *testing.T, reply any, err error) {
	assert.Equal(t, "reply", reply)
	assert.NoError(t, err)
}

func TestValidator(t *testing.T) {
	tests := []struct {
		name      string
		validator func(t *testing.T) *protovalidate.Validator
		req       proto.Message
		want      func(t *testing.T, reply any, err error)
	}{
		{
			name:      "",
			validator: defaultValidator,
			req: &pb.Person{
				Id:    1000,
				Email: "test@example.com",
				Name:  "test",
			},
			want: defaultWant,
		},
		{
			name:      "",
			validator: defaultValidator,
			req: &pb.Person{
				Id:    1000,
				Email: "test@example.com",
				Name:  "test",
			}, want: defaultWant,
		},
		{
			name:      "",
			validator: defaultValidator,
			req: &pb.Person{
				Id:    1000,
				Email: "test@example.com",
				Name:  "test",
			}, want: defaultWant,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reply, err := Server(
				Validator(tt.validator(t)),
				Handler(defaultHandler),
			)(next)(context.Background(), tt.req)
			tt.want(t, reply, err)
		})
	}
}
