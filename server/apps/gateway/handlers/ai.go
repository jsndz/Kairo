package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	aipb "github.com/jsndz/kairo/gen/go/proto/ai"
)

type AIHandler struct{
	AiClient aipb.AIServiceClient
}


func (h *AIHandler) Summarize(ctx *gin.Context) {
	idStr := ctx.Param("id")
	docID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		ctx.JSON(400, gin.H{"error": "Unable to save document "})
		return
	}
	ctx.JSON(200, gin.H{
		"Success": docID,
	})
}