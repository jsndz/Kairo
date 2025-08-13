package service

import (
	"github.com/jsndz/kairo/apps/document-service/internal/app/model"
	repos "github.com/jsndz/kairo/apps/document-service/internal/app/repo"
	"gorm.io/gorm"
)

type DocService struct{
	docRepo *repos.DocRepository
}


func NewDocService(db *gorm.DB)*DocService{
	return &DocService{
		docRepo: repos.NewDocRepository(db),
	}
}


func (s *DocService) CreateDoc(doc *model.Document) (*model.Document,error){
	doc,err :=s.docRepo.Create(doc)
	if err!=nil{
		return nil,err
	}
	return doc,nil
}


func (s *DocService) UpdateDoc(docId uint32,doc *model.Document)(*model.Document,error){
	doc,err := s.UpdateDoc(docId,doc)
	if err!=nil{
		return nil,err
	}
	return doc,nil
}

func (s *DocService) GetDoc(docId uint32)(*model.Document,error){
	doc,err := s.docRepo.GetFromId((docId))
	if err!=nil{
		return nil,err
	}
	return doc,nil
}
 
func (s *DocService) GetUserDocs(user_id uint32)(*[]model.Document,error){
	docs,err := s.docRepo.GetAll(user_id)
	if err!=nil{
		return nil,err
	}
	return docs,nil
}
 