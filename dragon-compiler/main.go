package main

import (
	"lexer"
	"llama_parser"
)

func main() {
	exprLexer := lexer.NewLexer("1+2*(4+3);")
	attributeParser := llama_parser.NewLLAMAParser(exprLexer)
	attributeParser.Parse()
}
