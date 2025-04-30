package lsp

import (
	"sort"
	"sync"
	"tops-lsp/lsp/data"
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

type TokenType uint32
type TokenModifierMask uint32

const (
	// 语义标记类型
	NamespaceType TokenType = iota
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
	DeclarationModifier TokenModifierMask = 1 << 0
	DefinitionModifier  TokenModifierMask = 1 << 1
	StaticModifier      TokenModifierMask = 1 << 2
	ReadonlyModifier    TokenModifierMask = 1 << 3
	AbstractModifier    TokenModifierMask = 1 << 4
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

func (v *SemanticTokenVisitor) AddToken(line, col, tokenLen int, tokenType TokenType, tokenModifiers TokenModifierMask) {
	if line == 0 {
		line = v.curLine
	}
	deltaLine := line - v.curLine
	deltaCol := col - v.curCol
	if deltaLine != 0 {
		deltaCol = col - 1
	}

	// 将 tokenModifiers 拼成 bitmask
	// var modifierMask uint32
	// for _, modifier := range tokenModifiers {
	// 	modifierMask |= 1 << modifier
	// }

	v.FileSemanticToken.tokens = append(v.FileSemanticToken.tokens, uint32(deltaLine), uint32(deltaCol), uint32(tokenLen), uint32(tokenType), uint32(tokenModifiers))
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

	v.AddToken(line, col, tokenLen, VariableType, DeclarationModifier)
}

func (v *SemanticTokenVisitor) EnterFunctionDecl(node AstNode) {
	// 示例逻辑：处理函数声明
	loc := node.Loc()
	if !locValidate(loc) {
		return
	}
	v.AddToken(loc.Line(), loc.Col(), loc.TokLen(), KeywordType, DeclarationModifier)
}

func (v *SemanticTokenVisitor) EnterParmVarDecl(node AstNode) {
	// 示例逻辑：处理参数变量声明
	loc := node.Loc()
	if !locValidate(loc) {
		return
	}
	v.AddToken(loc.Line(), loc.Col(), loc.TokLen(), ParameterType, DefinitionModifier|DeclarationModifier)
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
	l.semanticToken.AddToken(line, col, len(text), KeywordType, NonTokenModifier)
}
func (l *cppLexerListener) EnterBaseType(ctx *parser.BaseTypeContext) {
	// 处理基本类型
	loc := ctx.GetStart()
	line := loc.GetLine()
	col := loc.GetColumn() + 1
	text := loc.GetText()
	// info.Println("baseType: ", text, " line: ", line, " col: ", col)
	l.semanticToken.AddToken(line, col, len(text), TypeType, NonTokenModifier)
}
func (l *cppLexerListener) EnterOperator(ctx *parser.OperatorContext) {
	// 处理操作符
	loc := ctx.GetStart()
	line := loc.GetLine()
	col := loc.GetColumn() + 1
	text := loc.GetText()
	// info.Println("operator: ", text, " line: ", line, " col: ", col)
	l.semanticToken.AddToken(line, col, len(text), OperatorType, NonTokenModifier)
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
	l.semanticToken.AddToken(line, col, len(text), StringType, NonTokenModifier)
}
func (l *cppLexerListener) EnterNumber(ctx *parser.NumberContext) {
	// 处理数字字面量
	loc := ctx.GetStart()
	line := loc.GetLine()
	col := loc.GetColumn() + 1
	text := loc.GetText()
	// info.Println("numberLiteral: ", text, " line: ", line, " col: ", col)
	l.semanticToken.AddToken(line, col, len(text), NumberType, NonTokenModifier)
}
func (l *cppLexerListener) EnterComment(ctx *parser.CommentContext) {
	// 处理注释
	loc := ctx.GetStart()
	line := loc.GetLine()
	col := loc.GetColumn() + 1
	text := loc.GetText()
	// info.Println("comment: ", text, " line: ", line, " col: ", col)
	l.semanticToken.AddToken(line, col, len(text), CommentType, NonTokenModifier)
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

type SemanToken struct {
	Line           uint32
	Character      uint32
	Length         uint32
	TokenType      TokenType
	TokenModifiers TokenModifierMask
}

var NonTokenModifier TokenModifierMask = 0

func ParseSemanticTokenFromAst(ast *AstCache) *FileSemanticToken {
	inc_tokens := map[string][]*SemanToken{}
	tokens := []*SemanToken{}
	strtbl := ast.GetAst().StringTable.Entries

	addToken := func(loc *data.Location, tokentype TokenType, tokenModifiers TokenModifierMask) {
		filepath := strtbl[loc.FileName.Index]
		t := SemanToken{
			Line:           loc.Line,
			Character:      loc.Column,
			Length:         loc.Length,
			TokenType:      tokentype,
			TokenModifiers: tokenModifiers,
		}
		if ast.FindIncludeFile(filepath) {
			inc_tokens[filepath] = append(inc_tokens[filepath], &t)
		} else if filepath == ast.SourceFile() {
			tokens = append(tokens, &t)
		} else {
			elog.Println("unknown file: ", filepath)
		}
	}
	less_func := func(i *SemanToken, j *SemanToken) bool {
		if i.Line == j.Line {
			return i.Character < j.Character
		}
		return i.Line < j.Line
	}
	for _, ref := range ast.GetAst().DeclRefs {
		addToken(ref.Location, VariableType, DeclarationModifier)
	}
	for _, f := range ast.GetAst().FuncDefs {
		addToken(f.Location, FunctionType, DefinitionModifier)
		for _, parm := range f.Parameters {
			addToken(parm.Location, ParameterType, DeclarationModifier|DefinitionModifier)
		}
		for _, v := range f.LocalVars {
			addToken(v.Location, VariableType, DeclarationModifier|DefinitionModifier)
		}
	}
	for _, f := range ast.GetAst().FuncDecls {
		addToken(f.Location, FunctionType, DeclarationModifier)
		for _, parm := range f.Parameters {
			addToken(parm.Location, ParameterType, DeclarationModifier)
		}
	}
	for _, call := range ast.GetAst().GetFuncCalls() {
		addToken(call.Location, FunctionType, NonTokenModifier)
	}
	for _, g := range ast.GetAst().GlobalVars {
		addToken(g.Location, VariableType, DeclarationModifier)
	}

	fst := NewFileSemanticToken()
	sort.Slice(tokens, func(i, j int) bool {
		return less_func(tokens[i], tokens[j])
	})
	fst.tokens = to_relative_tokens(tokens)

	for k, v := range inc_tokens {
		sort.Slice(v, func(i, j int) bool {
			return less_func(v[i], v[j])
		})
		fst.includeFiles[k] = to_relative_tokens(v)
	}

	return fst
}

func to_relative_tokens(tokens []*SemanToken) []uint32 {
	if len(tokens) == 0 {
		return []uint32{}
	}
	rel_tokens := []uint32{}
	cur := tokens[0]
	rel_tokens = append(rel_tokens, cur.Line-1, cur.Character-1, cur.Length, uint32(cur.TokenType), uint32(cur.TokenModifiers))
	for i := 1; i < len(tokens); i++ {
		cur := tokens[i]
		before := tokens[i-1]
		deltaLine := cur.Line - before.Line
		deltaCol := cur.Character - before.Character
		if deltaLine != 0 {
			deltaCol = cur.Character - 1
		}

		rel_tokens = append(rel_tokens, deltaLine, deltaCol, cur.Length, uint32(cur.TokenType), uint32(cur.TokenModifiers))
	}
	return rel_tokens
}
