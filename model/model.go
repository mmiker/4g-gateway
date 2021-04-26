package model

import (
	"gorm.io/gorm"
)

type MQTTMsg struct {
	gorm.Model
	Topic string
	Msg   string
}
