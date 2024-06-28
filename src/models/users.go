package models

import "time"

type User struct {
	ID             uint      `gorm:"primaryKey;column:id"`
	Name           string    `gorm:"column:name"`
	Password       string    `gorm:"column:password"`
	Email          string    `gorm:"column:email"`
	OrganizationID uint      `gorm:"column:organization_id"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt      time.Time `gorm:"column:updated_at;autoUpdateTime"`
	Active         bool      `gorm:"column:active"`
}

func (User) TableName() string {
	return "users"
}
