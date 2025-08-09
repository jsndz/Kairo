package handlers

import (
	"github.com/gin-gonic/gin"
	docpb "github.com/jsndz/kairo-proto/proto/doc"
)


type DocHandlers struct{
	DocClient docpb.DocServiceClient
}


func(h *DocHandlers) CreateDoc(ctx *gin.Context)  {
	var req docpb.CreateDocumentRequest
	if err := ctx.ShouldBindBodyWithJSON(&req); err!=nil{
		ctx.JSON(400,gin.H{"error":"Unable parse json"})
		return 
	}

	res,err:= h.DocClient.CreateDocument(ctx,&req)

	if err!= nil{
		ctx.JSON(400,gin.H{"error":"Unable to get data"})
		return
	}
	ctx.JSON(200,res)
}

