package db

import (
	"fmt"
	"log"

	"github.com/jsndz/kairo/pkg/types"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)


func InitDB(config types.DBConfig) (*gorm.DB,error){

	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=%s",
		config.Host, config.User, config.Password, config.DBName, config.Port, config.TimeZone,
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Coudn't run postgres")
	}
	return db,nil

}