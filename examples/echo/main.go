package main

import (
	"context"

	"github.com/Ow1Dev/NoctiFunc/pkgs/sigil"
)

type Request struct {
	Name string `json:"name"`
}

type Response struct {
	Message string `json:"message"`
}

func HandleRequest(ctx context.Context, r Request) (Response, error) {
	return Response{
	 	Message: "Hello, " + r.Name + "!",
	}, nil
} 

func main() {
	sigil.Start(HandleRequest)
}
