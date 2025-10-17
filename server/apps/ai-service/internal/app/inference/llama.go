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
	outchan := make(chan string)
	errchan := make(chan error,1)


	go func ()  {
		defer close(outchan)
		defer close(errchan)
		req := Request{
			Model: model,
			Prompt: prompt,
		}

		data ,err := json.Marshal(req);
		if err!=nil{
			errchan <-err
			return
		}

		resp,err := http.Post("http://localhost:11434/api/generate", "application/json",bytes.NewBuffer(data))
		if err!=nil{
			errchan <-err
			return
		}
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan(){
			line := scanner.Text() 
			var msg Response 
			if err := json.Unmarshal([]byte(line), &msg); err != nil { 
				errchan <- err 
				continue
			} 
			outchan <- msg.Response 
			if msg.Done { break }
		}
		if err := scanner.Err(); err != nil { 
			errchan <- err 
		}
		
	}()

	return outchan,errchan
}