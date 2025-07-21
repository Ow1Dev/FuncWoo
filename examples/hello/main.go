package main

import (
	"github.com/Ow1Dev/NoctiFunc/pkgs/sigil"
)

func HandleRequest() (string, error) {
	return "Hello world", nil
} 

func main() {
	sigil.Start(HandleRequest)
}
