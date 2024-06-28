package models

import "time"

type Organization struct {
	ID        uint      `gorm:"primaryKey;column:id"`
	Name      string    `gorm:"column:name"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
	Active    bool      `gorm:"column:active"`
}

// TableName overrides the table name used by GORM for this model
func (Organization) TableName() string {
	return "organizations"
}
