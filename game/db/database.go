package db

import (
	"os"
	"sync"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	db   *gorm.DB
	once sync.Once
)

func Connect() {
	once.Do(func() {
		dsn := os.Getenv("POSTGRES_DSN")
		if dsn == "" {
			dsn = "host=postgres user=admin password=adminpassword dbname=battleship port=5432 sslmode=disable"
		}
		var d *gorm.DB
		var err error
		for i := 0; i < 30; i++ {
			d, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
			if err == nil {
				break
			}
			time.Sleep(time.Second)
		}
		if err != nil {
			panic(err)
		}
		db = d
	})
}

func GetDB() *gorm.DB {
	if db == nil {
		panic("database not initialized")
	}
	return db
}
