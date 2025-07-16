package main

import (
	"context"

	"github.com/Ow1Dev/NoctiFunc/pkgs/sigil"
)

func HandleRequest(ctx context.Context) (string, error) {
	return "Hello world", nil
} 

func main() {
	sigil.Start(HandleRequest)
}
