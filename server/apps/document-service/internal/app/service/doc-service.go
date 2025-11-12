package service

import (
	"time"

	"github.com/jsndz/kairo/apps/document-service/internal/app/model"
	repos "github.com/jsndz/kairo/apps/document-service/internal/app/repo"
	"github.com/jsndz/kairo/apps/document-service/utils"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type DocService struct {
	docRepo      *repos.DocRepository
	docUpdateSrv *DocUpdateService
}

func NewDocService(db *gorm.DB) *DocService {
	return &DocService{
		docRepo:      repos.NewDocRepository(db),
		docUpdateSrv: NewDocUpdateService(db),
	}
}

func (s *DocService) CreateDoc(doc *model.Document) (*model.Document, error) {
	start := time.Now()
	if doc == nil {
		log.Error().Msg("received nil document to create")
		return nil, gorm.ErrInvalidData
	}
	logger := log.With().
		Uint32("user_id", doc.UserID).
		Str("title", doc.Title).
		Logger()

	logger.Info().Msg("creating document")
	createdDoc, err := s.docRepo.Create(doc)
	if err != nil {
		logger.Error().Err(err).Msg("failed to create document")
		return nil, err
	}
	logger.Info().
		Dur("elapsed", time.Since(start)).
		Uint32("doc_id", createdDoc.ID).
		Msg("document created successfully")
	return createdDoc, nil
}

func (s *DocService) CreateDocWithDelta(doc *model.Document) (*model.Document, error) {
	start := time.Now()
	if doc == nil {
		log.Error().Msg("received nil document to create with delta")
		return nil, gorm.ErrInvalidData
	}
	logger := log.With().
		Uint32("user_id", doc.UserID).
		Str("title", doc.Title).
		Logger()

	logger.Info().Msg("creating document with initial delta")
	var createdDoc *model.Document
	DB := s.docRepo.GetDB()
	err := DB.Transaction(func(tx *gorm.DB) error {
		txLogger := logger.With().Str("component", "transaction").Logger()
		docRepoTx := repos.NewDocRepository(tx)
		docCreated, err := docRepoTx.Create(doc)
		if err != nil {
			txLogger.Error().Err(err).Msg("failed to create document in transaction")
			return err
		}
		createdDoc = docCreated
		updateRepo := repos.NewDocUpdateRepository(tx)
		initialUpdate := &model.DocumentUpdate{
			DocID:       createdDoc.ID,
			UpdateState: createdDoc.CurrentState,
		}
		_, err = updateRepo.Create(initialUpdate)
		if err != nil {
			txLogger.Error().Err(err).Uint32("doc_id", createdDoc.ID).Msg("failed to create initial document update")
			return err
		}
		txLogger.Debug().Uint32("doc_id", createdDoc.ID).Msg("initial document update created")
		return nil
	})
	if err != nil {
		logger.Error().Err(err).Msg("transaction failed while creating document with delta")
		return nil, err
	}
	logger.Info().
		Dur("elapsed", time.Since(start)).
		Uint32("doc_id", createdDoc.ID).
		Msg("document with delta created successfully")
	return createdDoc, err
}

func (s *DocService) UpdateDoc(docId uint32, doc *model.Document) (*model.Document, error) {
	start := time.Now()
	if doc == nil {
		log.Error().Uint32("doc_id", docId).Msg("received nil document payload for update")
		return nil, gorm.ErrInvalidData
	}
	stateBytes := len(doc.CurrentState)
	logger := log.With().
		Uint32("doc_id", docId).
		Int("state_bytes", stateBytes).
		Str("title", doc.Title).
		Logger()

	logger.Info().Msg("updating document")
	data := map[string]any{
		"current_state": doc.CurrentState,
	}
	doc, err := s.docRepo.Update(docId, data)
	if err != nil {
		logger.Error().Err(err).Msg("failed to update document")
		return nil, err
	}
	logger.Info().
		Dur("elapsed", time.Since(start)).
		Msg("document updated successfully")
	return doc, nil
}

func (s *DocService) GetDoc(docId uint32) (*model.Document, error) {
	start := time.Now()
	logger := log.With().Uint32("doc_id", docId).Logger()
	logger.Debug().Msg("fetching document by id")
	doc, err := s.docRepo.GetFromId((docId))
	if err != nil {
		logger.Error().Err(err).Msg("failed to fetch document")
		return nil, err
	}
	logger.Debug().
		Dur("elapsed", time.Since(start)).
		Time("updated_at", doc.UpdatedAt).
		Msg("document fetched")
	return doc, nil
}

func (s *DocService) GetUserDocs(user_id uint32) (*[]model.Document, error) {
	start := time.Now()
	logger := log.With().Uint32("user_id", user_id).Logger()
	logger.Info().Msg("fetching documents for user")
	docs, err := s.docRepo.GetAll(user_id)
	if err != nil {
		logger.Error().Err(err).Msg("failed to fetch user documents")
		return nil, err
	}
	count := 0
	if docs != nil {
		count = len(*docs)
	}
	logger.Info().
		Dur("elapsed", time.Since(start)).
		Int("doc_count", count).
		Msg("fetched documents for user")
	return docs, nil
}

func (s *DocService) ChangeTitle(doc_id uint32, new_title string) (string, error) {
	start := time.Now()
	logger := log.With().
		Uint32("doc_id", doc_id).
		Str("new_title", new_title).
		Logger()
	logger.Info().Msg("changing document title")
	doc, err := s.docRepo.Update(doc_id, map[string]interface{}{
		"title": new_title,
	})
	if err != nil {
		logger.Error().Err(err).Msg("failed to change document title")
		return "", err
	}
	logger.Info().
		Dur("elapsed", time.Since(start)).
		Msg("document title changed successfully")
	return doc.Title, nil
}

func (s *DocService) Save(doc_id uint32) (*model.Document, error) {
	start := time.Now()
	logger := log.With().Uint32("doc_id", doc_id).Logger()
	logger.Info().Msg("starting document save")
	doc, err := s.GetDoc(doc_id)
	if err != nil {
		logger.Error().Err(err).Msg("failed to fetch document for save")
		return nil, err
	}
	docUpdates, err := s.docUpdateSrv.GetAfterUpdates(doc_id, doc.UpdatedAt)
	if err != nil {
		logger.Error().Err(err).Time("doc_updated_at", doc.UpdatedAt).Msg("failed to fetch document updates")
		return nil, err
	}
	updateCount := 0
	if docUpdates != nil {
		updateCount = len(*docUpdates)
	}
	if updateCount == 0 {
		logger.Debug().Msg("no new updates to apply; returning existing document")
		return doc, nil
	}

	new_state, err := utils.CombineDeltaState(doc.CurrentState, docUpdates)
	if err != nil {
		logger.Error().Err(err).Msg("failed to combine delta state during save")
		return nil, err
	}

	newDoc, err := s.UpdateDoc(doc_id, &model.Document{CurrentState: new_state})
	if err != nil {
		logger.Error().Err(err).Msg("failed to update document with new state during save")
		return nil, err
	}
	logger.Info().
		Dur("elapsed", time.Since(start)).
		Int("applied_updates", updateCount).
		Msg("document saved successfully")
	return newDoc, nil
}

func (s *DocService) GetTextContent(doc_id uint32) (string, error) {
	start := time.Now()
	logger := log.With().Uint32("doc_id", doc_id).Logger()
	logger.Debug().Msg("getting text content for document")
	doc, err := s.GetDoc(doc_id)
	if err != nil {
		logger.Error().Err(err).Msg("failed to fetch document for text content")
		return "", err
	}
	content := utils.GetContent(doc.CurrentState)

	logger.Debug().
		Dur("elapsed", time.Since(start)).
		Int("content_length", len(content)).
		Msg("retrieved text content for document")
	return content, nil
}
