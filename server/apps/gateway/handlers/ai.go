package handlers

import aipb "github.com/jsndz/kairo/gen/go/proto/ai"

type AIHandler struct{
	AiClient aipb.AIServiceClient
}