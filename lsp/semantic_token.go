package lsp

import (
	"sync"
	"tops-lsp/parser"

	"github.com/antlr4-go/antlr/v4"
)

var SemanticTokenTypes []string = []string{
	"namespace",
	"type",
	"class",
	"enum",
	"struct",
	"typeParameter",
	"parameter",
	"variable",
	"enumMember",
	"function",
	"method",
	"macro",
	"keyword",
	"comment",
	"string",
	"number",
	"operator",
	"property",
}

const (
	// 语义标记类型
	NamespaceType = iota
	TypeType
	ClassType
	EnumType
	StructType
	TypeParameterType
	ParameterType
	VariableType
	EnumMemberType
	FunctionType
	MethodType
	MacroType
	KeywordType
	CommentType
	StringType
	NumberType
	OperatorType
	PropertType
)

var SemanticTokenModifiers []string = []string{
	"declaration",
	"definition",
	"static",
	"readonly",
	"abstract",
}

const (
	// 语义标记修饰符
	DeclarationModifier = iota
	DefinitionModifier
	StaticModifier
	ReadonlyModifier
	AbstractModifier
)

type SemanticToken interface {
	GetLine() int
	GetCharacter() int
	GetLength() int
	GetTokenType() int
	GetTokenModifiers() []int
	GetFile() *string
	GetTokenStr() string
}

type SemanticTokenLegend struct {
	TokenTypes     []string `json:"tokenTypes"`
	TokenModifiers []string `json:"tokenModifiers"`
}

type SemanticTokensOptions struct {
	Legend SemanticTokenLegend `json:"legend"`
	Full   bool                `json:"full"`
}

var semanticTokensOptions = SemanticTokensOptions{
	Legend: SemanticTokenLegend{
		TokenTypes:     SemanticTokenTypes,
		TokenModifiers: SemanticTokenModifiers,
	},
	Full: true,
}

type FileSemanticToken struct {
	tokens       []uint32
	includeFiles map[string][]uint32
}

func NewFileSemanticToken() *FileSemanticToken {
	return &FileSemanticToken{
		tokens:       []uint32{},
		includeFiles: make(map[string][]uint32),
	}
}

type SemanticTokenCache struct {
	fileSemanticToken map[string]*FileSemanticToken
	mutex             sync.RWMutex
}

func NewSemanticTokenCache() *SemanticTokenCache {
	return &SemanticTokenCache{
		fileSemanticToken: make(map[string]*FileSemanticToken),
	}
}

type SemanticTokenVisitor struct {
	AstVisitorBase
	FileSemanticToken *FileSemanticToken
	curLine           int
	curCol            int
	curLen            int
}

func NewSemanticTokenVisitor() *SemanticTokenVisitor {
	return &SemanticTokenVisitor{
		FileSemanticToken: NewFileSemanticToken(),
		curLine:           1,
		curCol:            0,
	}
}

func (v *SemanticTokenVisitor) AddToken(line, col, tokenLen int, tokenType int, tokenModifiers []int) {
	if line == 0 {
		line = v.curLine
	}
	deltaLine := line - v.curLine
	deltaCol := col - v.curCol
	if deltaLine != 0 {
		deltaCol = col - 1
	}

	// 将 tokenModifiers 拼成 bitmask
	var modifierMask uint32
	for _, modifier := range tokenModifiers {
		modifierMask |= 1 << modifier
	}

	v.FileSemanticToken.tokens = append(v.FileSemanticToken.tokens, uint32(deltaLine), uint32(deltaCol), uint32(tokenLen), uint32(tokenType), modifierMask)
	v.curLine = line
	v.curCol = col
	v.curLen = tokenLen
}

func locValidate(loc AstNodeLoc) bool {
	if loc == nil {
		return false
	}
	if loc.File() != "" {
		return false
	}
	if loc["includedFrom"] != nil {
		return false
	}
	if loc.Col() == 0 {
		return false
	}
	return true
}
func (v *SemanticTokenVisitor) EnterVarDecl(node AstNode) {
	// 示例逻辑：处理变量声明
	loc := node.Loc()
	if !locValidate(loc) {
		return
	}
	line := loc.Line()
	col := loc.Col()
	tokenLen := loc.TokLen()

	v.AddToken(line, col, tokenLen, VariableType, []int{DeclarationModifier})
}

func (v *SemanticTokenVisitor) EnterFunctionDecl(node AstNode) {
	// 示例逻辑：处理函数声明
	loc := node.Loc()
	if !locValidate(loc) {
		return
	}
	v.AddToken(loc.Line(), loc.Col(), loc.TokLen(), KeywordType, []int{DeclarationModifier})
}

func (v *SemanticTokenVisitor) EnterParmVarDecl(node AstNode) {
	// 示例逻辑：处理参数变量声明
	loc := node.Loc()
	if !locValidate(loc) {
		return
	}
	v.AddToken(loc.Line(), loc.Col(), loc.TokLen(), ParameterType, []int{DefinitionModifier, DeclarationModifier})
}

type cppLexerListener struct {
	parser.BaseCppListener
	semanticToken *SemanticTokenVisitor
}

func (l *cppLexerListener) EnterKeyword(ctx *parser.KeywordContext) {
	// 处理关键字
	loc := ctx.GetStart()
	line := loc.GetLine()
	col := loc.GetColumn() + 1
	text := loc.GetText()
	// info.Println("keyword: ", text, " line: ", line, " col: ", col)
	l.semanticToken.AddToken(line, col, len(text), KeywordType, []int{})
}
func (l *cppLexerListener) EnterBaseType(ctx *parser.BaseTypeContext) {
	// 处理基本类型
	loc := ctx.GetStart()
	line := loc.GetLine()
	col := loc.GetColumn() + 1
	text := loc.GetText()
	// info.Println("baseType: ", text, " line: ", line, " col: ", col)
	l.semanticToken.AddToken(line, col, len(text), TypeType, []int{})
}
func (l *cppLexerListener) EnterOperator(ctx *parser.OperatorContext) {
	// 处理操作符
	loc := ctx.GetStart()
	line := loc.GetLine()
	col := loc.GetColumn() + 1
	text := loc.GetText()
	// info.Println("operator: ", text, " line: ", line, " col: ", col)
	l.semanticToken.AddToken(line, col, len(text), OperatorType, []int{})
}
func (l *cppLexerListener) EnterIdentifier(ctx *parser.IdentifierContext) {
	// 处理标识符
	// loc := ctx.GetStart()
	// line := loc.GetLine()
	// col := loc.GetColumn() + 1
	// text := loc.GetText()
	// info.Println("identifier: ", text, " line: ", line, " col: ", col)
	// l.semanticToken.AddToken(line, col, len(text), VariableType, []int{DeclarationModifier})
}
func (l *cppLexerListener) EnterString(ctx *parser.StringContext) {
	// 处理字符串字面量
	loc := ctx.GetStart()
	line := loc.GetLine()
	col := loc.GetColumn() + 1
	text := loc.GetText()
	// info.Println("string: ", text, " line: ", line, " col: ", col)
	l.semanticToken.AddToken(line, col, len(text), StringType, []int{})
}
func (l *cppLexerListener) EnterNumber(ctx *parser.NumberContext) {
	// 处理数字字面量
	loc := ctx.GetStart()
	line := loc.GetLine()
	col := loc.GetColumn() + 1
	text := loc.GetText()
	// info.Println("numberLiteral: ", text, " line: ", line, " col: ", col)
	l.semanticToken.AddToken(line, col, len(text), NumberType, []int{})
}
func (l *cppLexerListener) EnterComment(ctx *parser.CommentContext) {
	// 处理注释
	loc := ctx.GetStart()
	line := loc.GetLine()
	col := loc.GetColumn() + 1
	text := loc.GetText()
	// info.Println("comment: ", text, " line: ", line, " col: ", col)
	l.semanticToken.AddToken(line, col, len(text), CommentType, []int{})
}
func (l *cppLexerListener) EnterStmt(ctx *parser.StmtContext) {
}
func (l *cppLexerListener) EnterOther(ctx *parser.OtherContext) {
}
func ParseSemanticToken(text string) []uint32 {
	inputStream := antlr.NewInputStream(text)
	lexer := parser.NewCppLexer(inputStream)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := parser.NewCppParser(stream)

	listener := &cppLexerListener{
		semanticToken: NewSemanticTokenVisitor(),
	}
	antlr.ParseTreeWalkerDefault.Walk(listener, parser.Cpp())

	// 遍历 tokenStream

	return listener.semanticToken.FileSemanticToken.tokens
}
