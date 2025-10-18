package handler

import (
	"context"

	"github.com/jsndz/kairo/apps/ai-service/internal/app/service"
	aipb "github.com/jsndz/kairo/gen/go/proto/ai"
	docpb "github.com/jsndz/kairo/gen/go/proto/doc"
)


type AiHandler struct {
	docClient docpb.DocServiceClient	
	aiService *service.AiService
}


func NewAiHandler(docClient docpb.DocServiceClient) (*AiHandler) {
	return &AiHandler{
		docClient: docClient,
		aiService: service.NewAiService(),
	}
}

func (h *AiHandler) Summarize(ctx context.Context, req *aipb.SummarizeRequest, stream aipb.AIService_SummarizeServer) (error)  {
	doc,err := h.docClient.GetDoc(ctx,&docpb.GetDocRequest{ Id: req.DocId})
	if err!= nil{
		return err
	}
	return nil
}


func (h *AiHandler) Rewrite(ctx context.Context, req *aipb.RewriteRequest, stream aipb.AIService_RewriteServer) (error)  {
	return	nil
}