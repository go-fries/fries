package gorm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestModel(t *testing.T) {
	assert.Equal(t, tableName, (&Model{}).TableName())

	SetTableName("test_embeddings_cache")
	assert.Equal(t, "test_embeddings_cache", (&Model{}).TableName())
}
