package models

import "time"

type Roles struct {
	ID          uint          `gorm:"primaryKey;column:id"`
	Name        string        `gorm:"column:name"`
	Active      bool          `gorm:"column:active"`
	CreatedAt   time.Time     `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time     `gorm:"column:updated_at;autoUpdateTime"`
	Permissions []Permissions `gorm:"many2many:role_permissions;"`
}

func (Roles) TableName() string {
	return "roles"
}

type RolePermissions struct {
	RoleID       uint `gorm:"primaryKey;column:role_id"`
	PermissionID uint `gorm:"primaryKey;column:permission_id"`
}

func (RolePermissions) TableName() string {
	return "role_permissions"
}
