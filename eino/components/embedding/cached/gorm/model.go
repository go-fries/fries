package gorm

import (
	"time"

	"gorm.io/datatypes"
)

var tableName = "embeddings_cache"

type Model struct {
	Key     string                        `gorm:"column:key;primaryKey;size:255;not null;comment:'Key'" json:"key"`
	Vector  datatypes.JSONType[[]float64] `gorm:"column:vector;type:json;not null;comment:'Vector'" json:"vector"`
	Expired time.Time                     `gorm:"column:expired;type:datetime;not null;comment:'Expired'" json:"expired"`
}

func (*Model) TableName() string {
	return tableName
}

func SetTableName(name string) {
	tableName = name
}
