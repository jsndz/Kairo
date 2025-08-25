package handler

import (
	"context"
	"time"

	"github.com/jsndz/kairo/apps/document-service/internal/app/model"
	"github.com/jsndz/kairo/apps/document-service/internal/app/service"
	docpb "github.com/jsndz/kairo/gen/go/proto/doc"
	"gorm.io/gorm"
)

type DocHandler struct{
	DocService *service.DocService
}

func NewDocHandler(db *gorm.DB,) * DocHandler {
	return &DocHandler{ 
		DocService: service.NewDocService(db),
	}
}

func toProtoDocument(doc *model.Document) *docpb.Document {
    return &docpb.Document{
        Id:           doc.ID,
        UserId:       doc.UserID,
        Title:        doc.Title,
        CurrentState: doc.CurrentState,
        CreatedAt:    doc.CreatedAt.Format(time.RFC3339),
        UpdatedAt:    doc.UpdatedAt.Format(time.RFC3339),
    }
}

func (h *DocHandler) CreateNewDoc(ctx context.Context,req *docpb.CreateNewDocumentRequest) (*docpb.CreateNewDocumentResponse,error){
	doc := &model.Document{
		Title:        "untitled doc",
		UserID:       req.UserId,
		CurrentState: []byte{},
	}
	createdDoc, err := h.DocService.CreateDocWithDelta(doc)
	if err != nil {
		return &docpb.CreateNewDocumentResponse{}, err
	}
	docProto := toProtoDocument(createdDoc)
	return &docpb.CreateNewDocumentResponse{
		Doc: docProto,
	}, nil
}

func (h *DocHandler) UpdateDoc(ctx context.Context, req *docpb.UpdateDocRequest) (*docpb.UpdateDocResponse, error) {
    updatedDoc, err := h.DocService.UpdateDoc(req.Id, &model.Document{
        Title:        req.Title,
        CurrentState: req.CurrentState,
    })
    if err != nil {
        return nil, err
    }

    return &docpb.UpdateDocResponse{
        Doc: toProtoDocument(updatedDoc),
    }, nil
}

func (h *DocHandler) GetDoc(ctx context.Context, req *docpb.GetDocRequest) (*docpb.GetDocResponse, error) {
    doc, err := h.DocService.GetDoc(req.Id)
    if err != nil {
        return nil, err
    }

    return &docpb.GetDocResponse{
        Doc: toProtoDocument(doc),
    }, nil
}

func (h *DocHandler) GetUserDocs(ctx context.Context,req *docpb.GetUserDocsRequest)(*docpb.GetUserDocsResponse,error){
	docs,err := h.DocService.GetUserDocs(req.UserId)
	if err != nil {
        return nil, err
    }
	resp:=&docpb.GetUserDocsResponse{}
	for i := range *docs {
		resp.Docs = append(resp.Docs,toProtoDocument(&(*docs)[i]))
	}
	return resp,err
}

func (h *DocHandler) ChangeDocName(ctx context.Context,req *docpb.ChangeDocNameRequest)(*docpb.ChangeDocNameResponse,error){
	title,err := h.DocService.ChangeTitle(req.DocId,req.NewTitle)
	if err != nil {
        return nil, err
    }
	resp:=&docpb.ChangeDocNameResponse{
		NewTitle: title,
	}
	
	return resp,err
}

func (h *DocHandler) Save(ctx context.Context,req *docpb.AutoSaveRequest)(*docpb.AutoSaveResponse,error){
	_,err := h.DocService.Save(req.DocId)
	if err != nil {
		resp:=&docpb.AutoSaveResponse{
			Success:false,
		}
        return resp, err
    }
	resp:=&docpb.AutoSaveResponse{
		Success:true,
	}
	
	return resp,err
}