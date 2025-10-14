package inference

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
)
type Request struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type Response struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

func StreamResponse(model string, prompt string) (<-chan string, <-chan error) {
	outChan := make(chan string)
	errChan := make(chan error, 1)

	go func() {
		defer close(outChan)
		defer close(errChan)

		reqBody := Request{
			Model:  model,
			Prompt: prompt,
		}

		body, err := json.Marshal(reqBody)
		if err != nil {
			errChan <- err
			return
		}

		resp, err := http.Post("http://localhost:11434/api/generate", "application/json", bytes.NewBuffer(body))
		if err != nil {
			errChan <- err
			return
		}
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			var msg Response
			if err := json.Unmarshal([]byte(line), &msg); err != nil {
				errChan <- err
				continue
			}
			outChan <- msg.Response
			if msg.Done {
				break
			}
		}

		if err := scanner.Err(); err != nil {
			errChan <- err
		}
	}()

	return outChan, errChan
}