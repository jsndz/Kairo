package app

import (
	"github.com/jsndz/kairo/apps/ai-service/internal/app/handler"
	aipb "github.com/jsndz/kairo/gen/go/proto/ai"
)

type AiServer struct{
	h *handler.AiHandler
	aipb.UnimplementedAIServiceServer
}

func NewAiServer(h *handler.AiHandler) (*AiServer) {
	return &AiServer{
		h:h,
	}
}