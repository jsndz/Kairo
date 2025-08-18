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
func (r *DocRepository) GetDB() *gorm.DB {
    return r.db
}

func (r *DocRepository) Create(doc *model.Document) (*model.Document ,error) {
	if err := r.db.Create(doc).Error; err != nil {
		log.Error().Err(err).Msg("Something went wrong in Create")
		return  nil,err
	}
	return  doc,nil
}


func (r *DocRepository) GetAll(userID uint32) (*[]model.Document, error) {
    var docs []model.Document
    if err := r.db.
        Select("id", "title", "user_id", "created_at", "updated_at").
        Where("user_id = ?", userID).
        Find(&docs).Error; err != nil {
        return nil, err
    }

    return &docs, nil
}


func (r *DocRepository) Update(ID uint32,data map[string]any) (*model.Document,error){
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