package models

import "time"

type Permissions struct {
	ID        uint      `gorm:"primaryKey;column:id"`
	Name      string    `gorm:"column:name"`
	Active    bool      `gorm:"column:active"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (Permissions) TableName() string {
	return "permissions"
}
