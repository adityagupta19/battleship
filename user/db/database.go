package db

import (
	"sync"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	db   *gorm.DB
	once sync.Once
)

func Connect() {
	once.Do(func() {
		dsn := "host=battleship-postgres-1 user=admin password=adminpassword dbname=battleship port=5432 sslmode=disable TimeZone=Asia/Shanghai"
		d, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			panic(err)
		}
		db = d
	})
}

func GetDB() *gorm.DB {
	if db == nil {
		panic("Database is not initialized. Call Connect() first.")
	}
	return db
}
