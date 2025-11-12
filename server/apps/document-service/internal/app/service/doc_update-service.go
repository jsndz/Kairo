package service

import (
	"time"

	"github.com/jsndz/kairo/apps/document-service/internal/app/model"
	repos "github.com/jsndz/kairo/apps/document-service/internal/app/repo"
	"github.com/rs/zerolog/log"
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
	start := time.Now()
	if update == nil {
		log.Error().Msg("received nil update to create")
		return nil, gorm.ErrInvalidData
	}
	logger := log.With().
		Uint32("doc_id", update.DocID).
		Int("update_state_bytes", len(update.UpdateState)).
		Logger()

	logger.Debug().Msg("creating document update")
	update, err := s.docUpdateRepo.Create(update)
	if err != nil {
		logger.Error().Err(err).Msg("failed to create document update")
		return nil, err
	}
	logger.Debug().
		Dur("elapsed", time.Since(start)).
		Uint32("update_id", update.ID).
		Msg("document update created")
	return update, nil
}

func (s *DocUpdateService) UpdateDocUpdate(updateId uint32, data map[string]any) (*model.DocumentUpdate, error) {
	start := time.Now()
	logger := log.With().
		Uint32("update_id", updateId).
		Int("fields", len(data)).
		Logger()
	logger.Debug().Msg("updating document update")
	update, err := s.docUpdateRepo.Update(updateId, data)
	if err != nil {
		logger.Error().Err(err).Msg("failed to update document update")
		return nil, err
	}
	logger.Debug().
		Dur("elapsed", time.Since(start)).
		Msg("document update updated")
	return update, nil
}

func (s *DocUpdateService) GetDocUpdate(updateId uint32) (*model.DocumentUpdate, error) {
	start := time.Now()
	logger := log.With().Uint32("update_id", updateId).Logger()
	logger.Debug().Msg("fetching document update by id")
	update, err := s.docUpdateRepo.GetFromId(updateId)
	if err != nil {
		logger.Error().Err(err).Msg("failed to fetch document update")
		return nil, err
	}
	logger.Debug().
		Dur("elapsed", time.Since(start)).
		Time("created_at", update.CreatedAt).
		Msg("document update fetched")
	return update, nil
}

func (s *DocUpdateService) DeleteDocUpdate(updateId uint32) error {
	logger := log.With().Uint32("update_id", updateId).Logger()
	logger.Debug().Msg("deleting document update")
	if err := s.docUpdateRepo.Delete(updateId); err != nil {
		logger.Error().Err(err).Msg("failed to delete document update")
		return err
	}
	logger.Debug().Msg("document update deleted")
	return nil
}

func (s *DocUpdateService) GetAfterUpdates(doc_id uint32, updated_at time.Time) (*[]model.DocumentUpdate, error) {
	start := time.Now()
	logger := log.With().
		Uint32("doc_id", doc_id).
		Time("after", updated_at).
		Logger()
	logger.Debug().Msg("fetching document updates after timestamp")
	updates, err := s.docUpdateRepo.GetAfterUpdates(doc_id, updated_at)
	if err != nil {
		logger.Error().Err(err).Msg("failed to fetch document updates after timestamp")
		return nil, err
	}
	count := 0
	if updates != nil {
		count = len(*updates)
	}
	logger.Debug().
		Dur("elapsed", time.Since(start)).
		Int("update_count", count).
		Msg("fetched document updates after timestamp")
	return updates, nil
}
