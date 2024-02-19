module dragon-compiler

go 1.17

require (
	lexer v0.0.0-00010101000000-000000000000
	llama_parser v0.0.0-00010101000000-000000000000
)

replace lexer => ./lexer

replace simple_parser => ./parser

replace expression_parser => ./expression_parser

replace pda => ./pda

replace augmented_parser => ./augmented_parser

replace attribute_parser => ./attribute_parser

replace llama_parser => ./llama_parser
