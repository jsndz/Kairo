package service

import (
	"log"

	"github.com/jsndz/kairo/apps/ai-service/internal/app/inference"
	aipb "github.com/jsndz/kairo/gen/go/proto/ai"
)

type AiService struct{

}

func NewAiService() (*AiService) {
	return &AiService{}
}

func (*AiService) Summarize(req *aipb.SummarizeRequest,stream aipb.AIService_SummarizeServer) error {

	outchan,errchan := inference.StreamResponse("mini","")
	select {
		case chunks,ok := <-outchan :
			if !ok {
				stream.Send(&aipb.SummarizeResponse{
					Summary: "",
					Done: true,
				})
				return nil

			}
			stream.Send(&aipb.SummarizeResponse{
				Summary: chunks,
				Done: false,
			})
			
		case err, ok := <-errchan:
			if ok && err != nil {
				log.Println("Stream error:", err)
				return err
			}
			errchan = nil 
		
	}
	return nil
}