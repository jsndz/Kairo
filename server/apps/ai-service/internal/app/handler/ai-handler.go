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
	content, err := h.docClient.GetTextContent(ctx, &docpb.GetTextContentRequest{DocId: req.DocId})
	if err!= nil{
		return err
	}
	err = h.aiService.Summarize(content.Text,stream)
	return err
}


func (h *AiHandler) Rewrite(ctx context.Context, req *aipb.RewriteRequest, stream aipb.AIService_RewriteServer) (error)  {
	return	nil
}