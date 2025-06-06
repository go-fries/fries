package gorm

import (
	"time"

	"gorm.io/datatypes"
)

type Model struct {
	Key     string                        `gorm:"column:key;primaryKey" json:"key"`
	Vector  datatypes.JSONType[[]float64] `gorm:"column:vector;type:json" json:"vector"`
	Expired time.Time                     `gorm:"column:expired;type:datetime" json:"expired"`
}

func (Model) TableName() string {
	return "embeddings_cache"
}
