package handler

import (
	"context"

	"github.com/jsndz/kairo/apps/document-service/internal/app/model"
	"github.com/jsndz/kairo/apps/document-service/internal/app/service"
	docpb "github.com/jsndz/kairo/gen/go/proto/doc"
	"gorm.io/gorm"
)


type DocUpdateHandler struct{
	DocUpdateService *service.DocUpdateService
}

func NewDocUpdateHandler(db *gorm.DB) * DocUpdateHandler {
	return &DocUpdateHandler{ 
		DocUpdateService: service.NewDocUpdateService(db),
	}
}

func (h * DocUpdateHandler) CreateDelta(ctx context.Context,req *docpb.CreateDeltaRequest)(*docpb.CreateDeltaResponse,error){
	_,err:=h.DocUpdateService.CreateDocUpdate(&model.DocumentUpdate{
		DocID: req.DocId,
		UpdateState: req.Delta,
	})
	if err!=nil{
		return &docpb.CreateDeltaResponse{},err
	}
	return &docpb.CreateDeltaResponse{},nil
}