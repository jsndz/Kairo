package handler

import "gorm.io/gorm"

type DocHandler struct{
}

func NewDocHandler(db *gorm.DB) * DocHandler {
	return &DocHandler{
	}
}