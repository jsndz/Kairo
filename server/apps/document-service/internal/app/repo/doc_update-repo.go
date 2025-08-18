package repos

import (
	"github.com/jsndz/kairo/apps/document-service/internal/app/model"
	"github.com/rs/zerolog/log"

	"gorm.io/gorm"
)

type DocUpdateRepository struct{
	db *gorm.DB
}


func NewDocUpdateRepository(db *gorm.DB) *DocUpdateRepository {
	return &DocUpdateRepository{db: db}
}

func (r *DocUpdateRepository) Create(doc *model.DocumentUpdate) (*model.DocumentUpdate ,error) {
	if err := r.db.Create(doc).Error; err != nil {
		log.Error().Err(err).Msg("Something went wrong in Create")
		return  nil,err
	}
	return  doc,nil
}

func (r *DocUpdateRepository) Update(ID uint32,data map[string]any) (*model.DocumentUpdate,error){
	var doc model.DocumentUpdate
	if err:= r.db.Model(&doc).Where("id = ?", ID).Updates(data).Error; err!=nil{
		return nil, err
	}
	r.db.First(&doc, ID)
	return &doc,nil
}

func (r *DocUpdateRepository) Delete(ID uint32) (error){
	var doc model.DocumentUpdate
	return  r.db.Delete(&doc,ID).Error
	
}

func (r *DocUpdateRepository) GetFromId(ID uint32) (*model.DocumentUpdate ,error){
	var doc model.DocumentUpdate
	err := r.db.First(&doc, ID).Error
	if err != nil {
		return nil, err
	}
	return &doc,nil
}