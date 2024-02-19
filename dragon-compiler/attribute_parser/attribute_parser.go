package attribute_parser

import (
	"fmt"
	"lexer"
)

type AttributeParser struct {
	parserLexer  lexer.Lexer
	reverseToken []lexer.Token
	//用于存储虚拟寄存器的名字
	registerNames []string
	//存储当前已分配寄存器的名字
	regiserStack []string
	//当前可用寄存器名字的下标
	registerNameIdx int
}

func NewAttributeParser(parserLexer lexer.Lexer) *AttributeParser {
	return &AttributeParser{
		parserLexer:     parserLexer,
		reverseToken:    make([]lexer.Token, 0),
		registerNames:   []string{"t0", "t1", "t2", "t3", "t4", "t5", "t6", "t7"},
		regiserStack:    make([]string, 0),
		registerNameIdx: 0,
	}
}

func (a *AttributeParser) putbackToken(token lexer.Token) {
	a.reverseToken = append(a.reverseToken, token)
}

func (a *AttributeParser) getToken() lexer.Token {
	//先看看有没有上次退回去的 token
	if len(a.reverseToken) > 0 {
		token := a.reverseToken[len(a.reverseToken)-1]
		a.reverseToken = a.reverseToken[0 : len(a.reverseToken)-1]
		return token
	}

	token, err := a.parserLexer.Scan()
	if err != nil && token.Tag != lexer.EOF {
		sErr := fmt.Sprintf("get token with err:%s\n", err)
		panic(sErr)
	}

	return token
}

func (a *AttributeParser) match(tag lexer.Tag) bool {
	token := a.getToken()
	if token.Tag != tag {
		a.putbackToken(token)
		return false
	}

	return true
}

func (a *AttributeParser) newName() string {
	//返回一个寄存器的名字
	if a.registerNameIdx >= len(a.registerNames) {
		//没有寄存器可用
		panic("register name running out")
	}
	name := a.registerNames[a.registerNameIdx]
	a.registerNameIdx += 1
	return name
}

func (a *AttributeParser) freeName(name string) {
	//释放当前寄存器名字
	if a.registerNameIdx > len(a.registerNames) {
		panic("register name index out of bound")
	}

	if a.registerNameIdx == 0 {
		panic("register name is full")
	}

	a.registerNameIdx -= 1
	a.registerNames[a.registerNameIdx] = name
}

func (a *AttributeParser) Parse() {
	a.stmt()
}

func (a *AttributeParser) stmt() {
	for a.match(lexer.EOF) != true {
		t := a.newName()
		a.expr(t)
		a.freeName(t)
		if a.match(lexer.SEMI) != true {
			panic("missing ; at the end of expression")
		}
	}
}

func (a *AttributeParser) expr(t string) {
	a.term(t)
	a.expr_prime(t)
}

func (a *AttributeParser) expr_prime(t string) {
	if a.match(lexer.PLUS) {
		t2 := a.newName()
		a.term(t2)
		fmt.Printf("%s += %s\n", t, t2)
		a.freeName(t2)
		a.expr_prime(t)
	}
}

func (a *AttributeParser) term(t string) {
	a.factor(t)
	a.term_prime(t)
}

func (a *AttributeParser) term_prime(t string) {
	if a.match(lexer.MUL) {
		t2 := a.newName()
		a.factor(t2)
		fmt.Printf("%s *= %s\n", t, t2)
		a.freeName(t2)
		a.term_prime(t)
	}
}

func (a *AttributeParser) factor(t string) {
	if a.match(lexer.NUM) {
		fmt.Printf("%s = %s\n", t, a.parserLexer.Lexeme)
	} else if a.match(lexer.LEFT_BRACKET) {
		a.expr(t)
		if a.match(lexer.RIGHT_BRACKET) != true {
			panic("missing ) for expr")
		}
	}
}
