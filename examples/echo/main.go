package main

import (
	"context"

	"github.com/Ow1Dev/NoctiFunc/pkgs/sigil"
)

type Request struct {
	Name string `json:"name"`
}

func HandleRequest(ctx context.Context, r Request) (string, error) {
	return "Hello, " + r.Name + "!", nil
} 

func main() {
	sigil.Start(HandleRequest)
}
