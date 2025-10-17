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



func (s *AiServer) Summarize(req *aipb.SummarizeRequest, stream aipb.AIService_SummarizeServer) error {
    return s.h.Summarize(stream.Context(),req,stream)
}

func (s *AiServer) Rewrite(req *aipb.RewriteRequest, stream aipb.AIService_RewriteServer) error {
    return s.h.Rewrite(stream.Context(),req,stream)
}
