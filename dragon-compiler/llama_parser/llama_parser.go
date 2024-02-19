package llama_parser

import (
	"fmt"
	"lexer"
)

type Attribute struct {
	left  string
	right string
}

const (
	ERROR   = -1
	EOF     = 0
	SEMI    = 2
	PLUS    = 3
	MUL     = 4
	NUM     = 5
	LP      = 6
	RP      = 7
	EPSILON = 255

	EXPR       = 257
	EXPR_PRIME = 259
	FACTOR     = 260
	STMT       = 256
	TERM       = 258
	TERM_PRIME = 261

	OP_PLUS     = 512
	OP_MUL      = 513
	CREATE_TMP  = 514
	ASSIGN_NAME = 515
	FREE_NAME   = 516
)

type ActionQuery struct {
	state int
	token lexer.Tag
}

type LLAMAParser struct {
	parseStack     []int
	attributeStack []Attribute
	yy_pushtab     [][]int
	yyd            map[ActionQuery]int
	parserLexer    lexer.Lexer
	registerNames  []string
	//存储当前已分配寄存器的名字
	regiserStack []string
	//当前可用寄存器名字的下标
	registerNameIdx int

	reverseToken []lexer.Token
}

func initYyPushTab(parser *LLAMAParser) {
	// stmt -> epsilon
	parser.yy_pushtab = append(parser.yy_pushtab, []int{EPSILON})
	// stmt -> {assign_name} expr {free_name} semi stmt
	parser.yy_pushtab = append(parser.yy_pushtab, []int{STMT, SEMI, FREE_NAME, EXPR, ASSIGN_NAME})
	// expr -> term expr_prime
	parser.yy_pushtab = append(parser.yy_pushtab, []int{EXPR_PRIME, TERM})
	// expr_prime -> PLUS {assign_name} term {op('+')} expr_prime
	parser.yy_pushtab = append(parser.yy_pushtab, []int{EXPR_PRIME, OP_PLUS, TERM, ASSIGN_NAME, PLUS})
	// expr_prime -> epsilon
	parser.yy_pushtab = append(parser.yy_pushtab, []int{EPSILON})
	// term -> factor term_prime
	parser.yy_pushtab = append(parser.yy_pushtab, []int{TERM_PRIME, FACTOR})
	// term_prime -> MUL {assign_name} factor {op('*')} term_prime
	parser.yy_pushtab = append(parser.yy_pushtab, []int{TERM_PRIME, OP_MUL, FACTOR, ASSIGN_NAME, MUL})
	// term_prime -> epsilon
	parser.yy_pushtab = append(parser.yy_pushtab, []int{EPSILON})
	// factor -> NUM {create_tmp}
	parser.yy_pushtab = append(parser.yy_pushtab, []int{CREATE_TMP, NUM})
	// factor -> LP EXPR RP
	parser.yy_pushtab = append(parser.yy_pushtab, []int{RP, EXPR, LP})
}

func initYYDRow(parser *LLAMAParser, state int,
	tags []lexer.Tag, val int) {
	for _, tag := range tags {
		parser.yyd[ActionQuery{
			state: state,
			token: tag,
		}] = val
	}
}

func initActionMap(parser *LLAMAParser) {
	//设置 yyd 表,后面我们能自动生成，这里我们先手动设置
	initYYDRow(parser, STMT,
		[]lexer.Tag{lexer.SEMI, lexer.PLUS,
			lexer.MUL, lexer.RIGHT_BRACKET}, -1)
	initYYDRow(parser, STMT, []lexer.Tag{lexer.EOF}, 0)
	initYYDRow(parser, STMT, []lexer.Tag{lexer.NUM, lexer.LEFT_BRACKET}, 1)

	initYYDRow(parser, EXPR, []lexer.Tag{lexer.EOF, lexer.SEMI, lexer.PLUS,
		lexer.MUL, lexer.RIGHT_BRACKET}, -1)
	initYYDRow(parser, EXPR, []lexer.Tag{lexer.NUM, lexer.LEFT_BRACKET}, 2)

	initYYDRow(parser, TERM, []lexer.Tag{lexer.EOF, lexer.SEMI, lexer.PLUS,
		lexer.MUL, lexer.RIGHT_BRACKET}, -1)
	initYYDRow(parser, TERM, []lexer.Tag{lexer.NUM, lexer.LEFT_BRACKET}, 5)

	initYYDRow(parser, EXPR_PRIME, []lexer.Tag{lexer.EOF,
		lexer.MUL, lexer.NUM, lexer.LEFT_BRACKET}, -1)
	initYYDRow(parser, EXPR_PRIME, []lexer.Tag{lexer.SEMI,
		lexer.RIGHT_BRACKET}, 4)
	initYYDRow(parser, EXPR_PRIME, []lexer.Tag{lexer.PLUS}, 3)

	initYYDRow(parser, FACTOR, []lexer.Tag{lexer.EOF,
		lexer.SEMI, lexer.PLUS, lexer.MUL, lexer.RIGHT_BRACKET}, -1)
	initYYDRow(parser, FACTOR, []lexer.Tag{lexer.NUM}, 8)
	initYYDRow(parser, FACTOR, []lexer.Tag{lexer.LEFT_BRACKET}, 9)

	initYYDRow(parser, TERM_PRIME, []lexer.Tag{lexer.EOF,
		lexer.MUL, lexer.NUM, lexer.MUL, lexer.LEFT_BRACKET}, -1)
	initYYDRow(parser, TERM_PRIME, []lexer.Tag{lexer.MUL}, 6)
	initYYDRow(parser, TERM_PRIME, []lexer.Tag{lexer.SEMI,
		lexer.PLUS, lexer.RIGHT_BRACKET}, 7)
}

func NewLLAMAParser(parserLexer lexer.Lexer) *LLAMAParser {
	parser := &LLAMAParser{
		parseStack:      make([]int, 0),
		attributeStack:  make([]Attribute, 0),
		yy_pushtab:      make([][]int, 0),
		parserLexer:     parserLexer,
		yyd:             make(map[ActionQuery]int),
		registerNames:   []string{"t0", "t1", "t2", "t3", "t4", "t5", "t6", "t7"},
		regiserStack:    make([]string, 0),
		registerNameIdx: 0,
		reverseToken:    make([]lexer.Token, 0),
	}

	initYyPushTab(parser)
	initActionMap(parser)

	parser.parseStack = append(parser.parseStack, STMT)
	parser.attributeStack = append(parser.attributeStack, Attribute{
		left:  "",
		right: "",
	})

	return parser
}

func (l *LLAMAParser) get(state int, tag lexer.Tag) int {
	return l.yyd[ActionQuery{
		state: state,
		token: tag,
	}]
}

func (l *LLAMAParser) newName() string {
	//返回一个寄存器的名字
	if l.registerNameIdx >= len(l.registerNames) {
		//没有寄存器可用
		panic("register name running out")
	}
	name := l.registerNames[l.registerNameIdx]
	l.registerNameIdx += 1
	return name
}

func (l *LLAMAParser) putbackName(name string) {
	//释放当前寄存器名字
	if l.registerNameIdx > len(l.registerNames) {
		panic("register name index out of bound")
	}

	if l.registerNameIdx == 0 {
		panic("register name is full")
	}

	l.registerNameIdx -= 1
	l.registerNames[l.registerNameIdx] = name
}

func (l *LLAMAParser) assignName() {
	//{$1=$2 = newName}
	stackTop := len(l.attributeStack) - 1
	t := l.newName()
	//$1 相当于 stackTop - 1, $2 相当于 stackTop - 2
	l.attributeStack[stackTop-1].right = t
	l.attributeStack[stackTop-2].right = t
}

func (l *LLAMAParser) freeName() {
	//{freeName($0)}
	stackTop := len(l.attributeStack) - 1
	//$0 对应栈顶元素
	l.putbackName(l.attributeStack[stackTop].right)
}

func (l *LLAMAParser) op(action string) {
	//$$对应栈顶元素的 left 字段
	//$0 对应栈顶元素的 right 字段
	stackTop := len(l.attributeStack) - 1
	fmt.Printf("%s %s= %s\n", l.attributeStack[stackTop].left,
		action, l.attributeStack[stackTop].right)
	l.freeName()
}

func (l *LLAMAParser) createTmp() {
	//$$ 对应栈顶元素的 left 字段
	stackTop := len(l.attributeStack) - 1
	fmt.Printf("%s = %s\n", l.attributeStack[stackTop].left,
		l.parserLexer.Lexeme)
}

func (l *LLAMAParser) putbackToken(token lexer.Token) {
	l.reverseToken = append(l.reverseToken, token)
}

func (l *LLAMAParser) getToken() lexer.Token {
	//先看看有没有上次退回去的 token
	if len(l.reverseToken) > 0 {
		token := l.reverseToken[len(l.reverseToken)-1]
		l.reverseToken = l.reverseToken[0 : len(l.reverseToken)-1]
		return token
	}

	token, err := l.parserLexer.Scan()
	if err != nil && token.Tag != lexer.EOF {
		sErr := fmt.Sprintf("get token with err:%s\n", err)
		panic(sErr)
	}

	return token
}

func (l *LLAMAParser) match(tag lexer.Tag) {
	token := l.getToken()
	if token.Tag != tag {
		panic("terminal symbol no match")
	}
}

func (l *LLAMAParser) takeAction(action int, rightOnTop string) {
	actions := l.yy_pushtab[action]
	for _, val := range actions {
		l.parseStack = append(l.parseStack, val)
		l.attributeStack = append(l.attributeStack, Attribute{
			left:  rightOnTop,
			right: rightOnTop,
		})
	}
}

func (l *LLAMAParser) isAction(symbol int) bool {
	return symbol >= 512
}

/*
stmt -> epsilon | {$1=$2=newName()} expr {freeName($0)} SEMI stmt
expr -> term expr_prime
expr_prime -> PLUS {$1=$2=newName()} term {printf("%s+=%s\n", $$, $0); freeName($0);} expr_prime| epsilon
term -> factor term_prime
term_prime -> MUL {$1=$2=newName();} factor {printf("%s *= %s\n", $$, $0); freeName($0);} term_prime | epsilon
factor -> NUM {printf("%s=%s\n", $$, lexer.lexeme)}

*/

func (l *LLAMAParser) Parse() {
	for len(l.parseStack) != 0 {
		symbol := l.parseStack[len(l.parseStack)-1]
		//顶部堆栈元素相当于表达式箭头左边的非终结符 s
		l.parseStack = l.parseStack[0 : len(l.parseStack)-1]
		rightOnTop := l.attributeStack[len(l.attributeStack)-1].right
		if l.isAction(symbol) != true {
			l.attributeStack = l.attributeStack[0 : len(l.attributeStack)-1]
		}

		switch symbol {
		//匹配行动
		case ASSIGN_NAME:
			l.assignName()
			l.attributeStack = l.attributeStack[0 : len(l.attributeStack)-1]
		case CREATE_TMP:
			l.createTmp()
			l.attributeStack = l.attributeStack[0 : len(l.attributeStack)-1]
		case FREE_NAME:
			l.freeName()
			l.attributeStack = l.attributeStack[0 : len(l.attributeStack)-1]
		case OP_PLUS:
			l.op("+")
			l.attributeStack = l.attributeStack[0 : len(l.attributeStack)-1]
		case OP_MUL:
			l.op("*")
			l.attributeStack = l.attributeStack[0 : len(l.attributeStack)-1]
		//匹配终结符
		case EOF:
			l.match(lexer.EOF)
		case NUM:
			l.match(lexer.NUM)
		case SEMI:
			l.match(lexer.SEMI)
		case PLUS:
			l.match(lexer.PLUS)
		case MUL:
			l.match(lexer.MUL)
		case LP:
			l.match(lexer.LEFT_BRACKET)
		case RP:
			l.match(lexer.RIGHT_BRACKET)
		//匹配非终结符
		case STMT:
			fallthrough
		case EXPR:
			fallthrough
		case EXPR_PRIME:
			fallthrough
		case TERM:
			fallthrough
		case TERM_PRIME:
			fallthrough
		case FACTOR:
			token := l.getToken()
			l.putbackToken(token)

			action := l.get(symbol, token.Tag)
			if action == -1 {
				panic("parse error, not action for given symbol and input")
			}
			l.takeAction(action, rightOnTop)
		}

	}
}
