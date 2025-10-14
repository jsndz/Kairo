package inference

import (
	"context"
	"os"

	"github.com/sashabaranov/go-openai"
)


func StreamOpenAIResponse(prompt string) (<-chan string, <-chan error) {
	outChan := make(chan string)
	errChan := make(chan error, 1)

	go func() {
		defer close(outChan)
		defer close(errChan)

		client := openai.NewClient(os.Getenv("OPENAI_API_KEY"))

		stream, err := client.CreateChatCompletionStream(
			context.Background(),
			openai.ChatCompletionRequest{
				Model: "gpt-4o-mini", 
				Messages: []openai.ChatCompletionMessage{
					{Role: "user", Content: prompt},
				},
			},
		)
		if err != nil {
			errChan <- err
			return
		}
		defer stream.Close()

		for {
			response, err := stream.Recv()
			if err != nil {
				if err.Error() != "EOF" {
					errChan <- err
				}
				break
			}
			for _, choice := range response.Choices {
				outChan <- choice.Delta.Content
			}
		}
	}()

	return outChan, errChan
}
