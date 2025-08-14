package handlers

import (
	"log"
	"strconv"

	"github.com/gin-gonic/gin"
	docpb "github.com/jsndz/kairo/gen/go/proto/doc"
)


type DocHandlers struct{
	DocClient docpb.DocServiceClient
}


func(h *DocHandlers) CreateDoc(ctx *gin.Context)  {
	var req docpb.CreateNewDocumentRequest
	if err := ctx.ShouldBindBodyWithJSON(&req); err!=nil{
		ctx.JSON(400,gin.H{"error":"Unable parse json"})
		return 
	}

	res,err:= h.DocClient.CreateNewDocument(ctx,&req)

	if err!= nil{
		ctx.JSON(400,gin.H{"error":"Unable to get data"})
		return
	}
	ctx.JSON(200,res)
}

func(h *DocHandlers) UpdateDoc(ctx *gin.Context)  {
	var req docpb.UpdateDocRequest

    idStr := ctx.Param("id")
    id, parseErr := strconv.ParseUint(idStr, 10, 32)
    if parseErr != nil {
        ctx.JSON(400, gin.H{"error": "Invalid document ID"})
        return
    }

    if bindErr := ctx.ShouldBindJSON(&req); bindErr != nil {
        ctx.JSON(400, gin.H{"error": "Unable to parse JSON"})
        return
    }

    data := &docpb.UpdateDocRequest{
        Id:           uint32(id),
        Title:        req.Title,
        CurrentState: req.CurrentState,
    }

    res, grpcErr := h.DocClient.UpdateDoc(ctx, data)
    if grpcErr != nil {
        ctx.JSON(500, gin.H{"error": "Unable to update document"})
        return
    }

    ctx.JSON(200, res)
}

func(h *DocHandlers) GetUserDocs(ctx *gin.Context)  {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	

	res,err:= h.DocClient.GetUserDocs(ctx,&docpb.GetUserDocsRequest{UserId: uint32(id)})

	if err!= nil{
		ctx.JSON(400,gin.H{"error":"Unable to get dataY"})
		log.Print(err)
		return
	}
	ctx.JSON(200,res)
}

func(h *DocHandlers) GetDoc(ctx *gin.Context)  {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		ctx.JSON(400, gin.H{"error": "Invalid ID"})
		return
	}
	res,err:= h.DocClient.GetDoc(ctx,&docpb.GetDocRequest{Id: uint32(id)})

	if err!= nil{
		ctx.JSON(400,gin.H{"error":"Unable to get data"})
		return
	}
	ctx.JSON(200,res)
}