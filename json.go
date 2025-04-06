package main

import (
	"encoding/json"
	"fmt"
	"log"
)

type JSONResponse struct {
	Data []map[string]any `json:"data"`
}

type JSONErrorResponse struct {
	Error   string `json:"error"`
	Status  int    `json:"status"`
	Message string `json:"message"`
}

type WSResponse struct {
	Metadata map[string]any `json:"metadata"`
	Payload  map[string]any `json:"payload"`
}

func PrettyPrint(v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Println(string(b))
}
