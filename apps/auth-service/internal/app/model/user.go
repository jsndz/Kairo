package model

import (
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)
type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

type User struct {
	ID       uint   `gorm:"primaryKey"`
	Name string `gorm:"unique;not null"`
	Email 	 string `gorm:"unique;not null"`
	Password string `gorm:"not null"`
	Role string `gorm:"type:enum('user','admin');default:'user'"`
}
	

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}


func CheckPassword(hashedPassword, inputPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(inputPassword))
	return err == nil
}