package main

import (
	"context"

	"github.com/Ow1Dev/FuncWoo/pkgs/sigil"
)

func HandleRequest(ctx context.Context) (string, error) {
	return "Hello world", nil
} 

func main() {
	sigil.Start(HandleRequest)
}
