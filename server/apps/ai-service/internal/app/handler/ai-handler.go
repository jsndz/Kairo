package handler

import "github.com/jsndz/kairo/apps/ai-service/internal/app/service"


type AiHandler struct {
	aiService *service.AiService
}


func NewAiHandler() (*AiHandler) {
	return &AiHandler{
		aiService: service.NewAiService(),
	}
}

func (h *AiHandler)Summarize()  {
	
}


func (h *AiHandler)Rewrite()  {
	
}