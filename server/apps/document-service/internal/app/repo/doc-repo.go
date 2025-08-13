package repos

import (
	"github.com/jsndz/kairo/apps/document-service/internal/app/model"
	"github.com/rs/zerolog/log"

	"gorm.io/gorm"
)

type DocRepository struct{
	db *gorm.DB
}


func NewDocRepository(db *gorm.DB) *DocRepository {
	return &DocRepository{db: db}
}

func (r *DocRepository) Create(doc *model.Document) (*model.Document ,error) {
	if err := r.db.Create(doc).Error; err != nil {
		log.Error().Err(err).Msg("Something went wrong in Create")
		return  nil,err
	}
	return  doc,nil
}

func (r *DocRepository) GetAll(user_id uint32) (*[]model.Document, error) {
    var docs []model.Document

	err := r.db.Where(&docs, "user_id = ?", user_id).Find(&docs).Error
    if err != nil {
        return nil, err 
    }
    return &docs, nil
}

func (r *DocRepository) Update(ID string,data map[string]any) (*model.Document,error){
	var doc model.Document
	if err:= r.db.Model(&doc).Where("id = ?", ID).Updates(data).Error; err!=nil{
		return nil, err
	}
	r.db.First(&doc, ID)
	return &doc,nil
}

func (r *DocRepository) Delete(ID uint32) (error){
	var doc model.Document
	return  r.db.Delete(&doc,ID).Error
	
}

func (r *DocRepository) GetFromId(ID uint32) (*model.Document ,error){
	var doc model.Document
	err := r.db.First(&doc, ID).Error
	if err != nil {
		return nil, err
	}
	return &doc,nil
}