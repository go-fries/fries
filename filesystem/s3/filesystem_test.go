package s3

import (
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
)

var ctx = context.Background()

func TestStorage(t *testing.T) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-west-2"))
	assert.NoError(t, err)
	svc := s3.NewFromConfig(cfg)

	storage := New(svc, "bucket")

	storage.Get(ctx, "path")
}
