package utils

import (
	"log"

	y "github.com/skyterra/y-crdt"
)

func GetContent(data []byte) string {
	doc := y.NewDoc("default", true, nil, nil, false)
 
	y.ApplyUpdate(doc, data, nil)

	text := doc.GetText("content")
	log.Println(text.ToString())

	return text.ToString()
}