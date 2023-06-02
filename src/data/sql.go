package data

import (
	"database/sql"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func NewDatabaseHandler() *DatabaseHandler {
	db, err := sql.Open("mysql", "user:password@tcp(db:3306)/classicmodels")
	if err != nil {
		panic(err)
	}
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	gormDB, err := gorm.Open(mysql.New(mysql.Config{
		Conn: db,
	}), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	return &DatabaseHandler{
		client: *gormDB,
	}

}

type DatabaseHandler struct {
	client gorm.DB
}

func (dh *DatabaseHandler) GetDBClient() *gorm.DB {
	return &dh.client
}

func (dh *DatabaseHandler) CloseConnection() {
	db, err := dh.client.DB()
	if err != nil {
		panic(err)
	}
	db.Close()
}
