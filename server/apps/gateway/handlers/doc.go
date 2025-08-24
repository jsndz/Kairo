package handlers

import (
	"log"
	"strconv"

	"github.com/gin-gonic/gin"
	authpb "github.com/jsndz/kairo/gen/go/proto/auth"
	docpb "github.com/jsndz/kairo/gen/go/proto/doc"
)


type DocHandlers struct{
	DocClient docpb.DocServiceClient
	AuthClient authpb.AuthServiceClient
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
		ctx.JSON(400,gin.H{"error":"Unable to get data"})
		return
	}
	ctx.JSON(200,res)
}

func (h *DocHandlers) GetDoc(ctx *gin.Context) {
	idStr := ctx.Param("id")
	accept:= ctx.GetHeader("Accept")
	docID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		ctx.JSON(400, gin.H{"error": "Invalid document ID"})
		return
	}

	userIDVal, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}

	userIDStr, ok := userIDVal.(string)
	if !ok {
		ctx.JSON(400, gin.H{"error": "Invalid user ID"})
		return
	}
	userIDUint64, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		ctx.JSON(400, gin.H{"error": "Invalid user ID format"})
		return
	}
	docResp, err := h.DocClient.GetDoc(ctx, &docpb.GetDocRequest{Id: uint32(docID)})
	if err != nil {
		ctx.JSON(400, gin.H{"error": "Unable to get document"})
		return
	}

	wsTokenResp, err := h.AuthClient.CreateWSToken(ctx, &authpb.CreateWSTokenRequest{
		UserId: uint32(userIDUint64),
		DocId:  uint32(docID),
	})
	if err != nil {
		ctx.JSON(400, gin.H{"error": "Unable to generate WS token"})
		return
	}

	ctx.SetCookie("kairo_ws_token", wsTokenResp.Token, 300, "/", "", false, false)

	log.Println(docResp.Doc.CurrentState)

	switch accept{
		case "application/json":{
			ctx.JSON(200, gin.H{
					"document": gin.H{
						"id":        docResp.Doc.Id,
						"title":     docResp.Doc.Title,
						"user_id":   docResp.Doc.UserId,
						"createdAt": docResp.Doc.CreatedAt,
						"updatedAt": docResp.Doc.UpdatedAt,
					},
					"ws_token": wsTokenResp.Token,
			})
		}
	    case "application/octet-stream":{
			    ctx.Data(200, "application/octet-stream", docResp.Doc.CurrentState)
		}
	}
}



func (h *DocHandlers) ChangeDocName(ctx *gin.Context) {
	var req docpb.ChangeDocNameRequest
	idStr := ctx.Param("id")
	docID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		ctx.JSON(400, gin.H{"error": "Invalid document ID"})
		return
	}
	if bindErr := ctx.ShouldBindJSON(&req); bindErr != nil {
        ctx.JSON(400, gin.H{ "error": "Unable to parse JSON" })
        return
    }

	docResp, err := h.DocClient.ChangeDocName(ctx, &docpb.ChangeDocNameRequest{DocId: uint32(docID),NewTitle: req.NewTitle})
	if err != nil {
		ctx.JSON(400, gin.H{"error": "Unable to get document"})
		return
	}
	ctx.JSON(200, gin.H{
		"new_title": docResp.NewTitle,
	})
}


