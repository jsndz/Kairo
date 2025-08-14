package app

import (
	"context"
	"log"

	"github.com/jsndz/kairo/apps/document-service/internal/app/handler"
	docpb "github.com/jsndz/kairo/gen/go/proto/doc"
)

type DocServer struct {
	h *handler.DocHandler
	docpb.UnimplementedDocServiceServer
}

func NewDocServer(h *handler.DocHandler) *DocServer {
	return &DocServer{h: h}
}

func (s *DocServer) CreateNewDocument(ctx context.Context, req *docpb.CreateNewDocumentRequest) (*docpb.CreateNewDocumentResponse, error) {
	resp, err := s.h.CreateNewDoc(ctx, req)
	if err != nil {
		log.Println("CreateNewDocument error:", err)
		return nil, err
	}
	return resp, nil
}

func (s *DocServer) UpdateDoc(ctx context.Context, req *docpb.UpdateDocRequest) (*docpb.UpdateDocResponse, error) {
	resp, err := s.h.UpdateDoc(ctx, req)
	if err != nil {
		log.Println("UpdateDoc error:", err)
		return nil, err
	}
	return resp, nil
}

func (s *DocServer) GetUserDocs(ctx context.Context, req *docpb.GetUserDocsRequest) (*docpb.GetUserDocsResponse, error) {
	resp, err := s.h.GetUserDocs(ctx, req)
	if err != nil {
		log.Println("GetUserDocs error:", err)
		return nil, err
	}
	return resp, nil
}

func (s *DocServer) GetDoc(ctx context.Context, req *docpb.GetDocRequest) (*docpb.GetDocResponse, error) {
	resp, err := s.h.GetDoc(ctx, req)
	if err != nil {
		log.Println("GetDoc error:", err)
		return nil, err
	}
	return resp, nil
}
