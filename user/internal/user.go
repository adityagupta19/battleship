package internal

import (
	"time"

	db "github.com/adityagupta19/battleship/user/db"
	"github.com/gofiber/fiber/v2/log"
	"gorm.io/gorm"
)

var DB *gorm.DB

type User struct {
	ID        uint      `gorm:"primaryKey"`
	Username  string    `gorm:"unique;not null"`
	Rating    int       `gorm:"default:1000"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

func init() {
	db.Connect()
	DB = db.GetDB()
	DB.AutoMigrate(&User{})
}

func RegisterUser(username string) (User, error) {
	newUser := User{Username: username}
	res := DB.Create(&newUser)

	if res.Error != nil {
		log.Warnf("Failed to create new user: %+v, error: %v", newUser, res.Error)
		return User{}, res.Error
	}
	return newUser, nil
}

func GetUser(id uint) (User, error) {
	var user User
	result := DB.First(&user, id)

	if result.Error != nil {
		return User{}, result.Error // Return error if user is not found
	}

	return user, nil
}
