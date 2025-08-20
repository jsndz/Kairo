package service

import (
	"time"

	"github.com/jsndz/kairo/apps/document-service/internal/app/model"
	repos "github.com/jsndz/kairo/apps/document-service/internal/app/repo"
	"gorm.io/gorm"
)

type DocUpdateService struct {
	docUpdateRepo *repos.DocUpdateRepository
}

func NewDocUpdateService(db *gorm.DB) *DocUpdateService {
	return &DocUpdateService{
		docUpdateRepo: repos.NewDocUpdateRepository(db),
	}
}

func (s *DocUpdateService) CreateDocUpdate(update *model.DocumentUpdate) (*model.DocumentUpdate, error) {
	update, err := s.docUpdateRepo.Create(update)
	if err != nil {
		return nil, err
	}
	return update, nil
}

func (s *DocUpdateService) UpdateDocUpdate(updateId uint32, data map[string]any) (*model.DocumentUpdate, error) {
	update, err := s.docUpdateRepo.Update(updateId, data)
	if err != nil {
		return nil, err
	}
	return update, nil
}

func (s *DocUpdateService) GetDocUpdate(updateId uint32) (*model.DocumentUpdate, error) {
	update, err := s.docUpdateRepo.GetFromId(updateId)
	if err != nil {
		return nil, err
	}
	return update, nil
}

func (s *DocUpdateService) DeleteDocUpdate(updateId uint32) error {
	return s.docUpdateRepo.Delete(updateId)
}

func (s *DocUpdateService) GetAfterUpdates(doc_id uint32,updated_at time.Time) (*[]model.DocumentUpdate, error) {
	return s.docUpdateRepo.GetAfterUpdates(doc_id,updated_at)
}
