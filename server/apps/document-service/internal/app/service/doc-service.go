package service

import (
	"github.com/jsndz/kairo/apps/document-service/internal/app/model"
	repos "github.com/jsndz/kairo/apps/document-service/internal/app/repo"
	"github.com/jsndz/kairo/apps/document-service/utils"
	"gorm.io/gorm"
)

type DocService struct{
	docRepo *repos.DocRepository
	docUpdateSrv  *DocUpdateService
}


func NewDocService(db *gorm.DB)*DocService{
	return &DocService{
		docRepo: repos.NewDocRepository(db),
		docUpdateSrv: NewDocUpdateService(db),
	}
}




func (s *DocService) CreateDoc(doc *model.Document) (*model.Document,error){
	
	doc,err :=s.docRepo.Create(doc)
	
	if err!=nil{
		return nil,err
	}
	return doc,nil
}

func (s *DocService) CreateDocWithDelta(doc *model.Document) (*model.Document,error){
	var createdDoc *model.Document
	DB := s.docRepo.GetDB()
	err := DB.Transaction(func(tx *gorm.DB) error {
		docRepoTx := repos.NewDocRepository(tx)
		docCreated, err := docRepoTx.Create(doc)
		if err != nil {
			return err
		}
		createdDoc = docCreated
		updateRepo :=repos.NewDocUpdateRepository(tx)
		initialUpdate := &model.DocumentUpdate{
			DocID: createdDoc.ID,
			UpdateState: createdDoc.CurrentState,
		}
		_, err = updateRepo.Create(initialUpdate)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return createdDoc, err
}


func (s *DocService) UpdateDoc(docId uint32,doc *model.Document)(*model.Document,error){
	data := map[string]any{
        "title": doc.Title,
        "current_state": doc.CurrentState,
    }
	doc,err := s.docRepo.Update(docId,data)
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
 
func (s *DocService) ChangeTitle(doc_id uint32,new_title string) (string, error) {
    doc, err := s.docRepo.Update(doc_id, map[string]interface{}{
        "title": new_title,
    })
    if err != nil {
        return "", err
    }
    return doc.Title, nil
}


func (s *DocService) Save(doc_id uint32)(*model.Document,error){
	doc ,err := s.GetDoc(doc_id)
	if err != nil{
		return nil,err
	}
	docUpdates ,err:= s.docUpdateSrv.GetAfterUpdates(doc_id,doc.UpdatedAt)
	if err != nil{
		return nil,err
	}
	new_state,err := utils.CombineDeltaState(doc.CurrentState, docUpdates)
	if err != nil{
		return nil,err
	}
	newDoc ,err := s.UpdateDoc(doc_id,&model.Document{CurrentState: new_state})
	if err != nil{
		return nil,err
	}
	return newDoc,err
}

func (s *DocService) GetTextContent(doc_id uint32)(string,error) {
	doc,err := s.GetDoc(doc_id)
	if err != nil{
		return "",err
	}
	content := utils.GetContent(doc.CurrentState)

	return content,nil
}