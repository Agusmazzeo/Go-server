package database

import (
	"server/src/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func SetupDB(cfg *config.Config) (*gorm.DB, error) {
	// Setup GORM
	dsn := "host=" + cfg.Databases.SQL.Host + " user=" + cfg.Databases.SQL.Username + " password=" + cfg.Databases.SQL.Password + " dbname=" + cfg.Databases.SQL.Database + " port=" + cfg.Databases.SQL.Port + " sslmode=disable"
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}
	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, err
	}
	return db, nil
}
