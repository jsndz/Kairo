package handler

import (
	"context"

	"github.com/jsndz/kairo/apps/ai-service/internal/app/service"
	aipb "github.com/jsndz/kairo/gen/go/proto/ai"
)


type AiHandler struct {
	aiService *service.AiService
}


func NewAiHandler() (*AiHandler) {
	return &AiHandler{
		aiService: service.NewAiService(),
	}
}

func (h *AiHandler) Summarize(ctx context.Context, req *aipb.SummarizeRequest, stream aipb.AIService_SummarizeServer) (error)  {
	return nil
}


func (h *AiHandler) Rewrite(ctx context.Context, req *aipb.RewriteRequest, stream aipb.AIService_RewriteServer) (error)  {
	return	nil
}