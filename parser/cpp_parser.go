// Code generated from Cpp.g4 by ANTLR 4.13.2. DO NOT EDIT.

package parser // Cpp

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/antlr4-go/antlr/v4"
)

// Suppress unused import errors
var _ = fmt.Printf
var _ = strconv.Itoa
var _ = sync.Once{}

type CppParser struct {
	*antlr.BaseParser
}

var CppParserStaticData struct {
	once                   sync.Once
	serializedATN          []int32
	LiteralNames           []string
	SymbolicNames          []string
	RuleNames              []string
	PredictionContextCache *antlr.PredictionContextCache
	atn                    *antlr.ATN
	decisionToDFA          []*antlr.DFA
}

func cppParserInit() {
	staticData := &CppParserStaticData
	staticData.LiteralNames = []string{
		"", "", "", "", "", "", "", "", "", "", "'alignas'", "'alignof'", "'asm'",
		"'auto'", "'bool'", "'break'", "'case'", "'catch'", "'char'", "'char16_t'",
		"'char32_t'", "'class'", "'const'", "'constexpr'", "'const_cast'", "'continue'",
		"'decltype'", "'default'", "'delete'", "'do'", "'double'", "'dynamic_cast'",
		"'else'", "'enum'", "'explicit'", "'export'", "'extern'", "'false'",
		"'final'", "'float'", "'for'", "'friend'", "'goto'", "'if'", "'inline'",
		"'int'", "'long'", "'mutable'", "'namespace'", "'new'", "'noexcept'",
		"'nullptr'", "'operator'", "'override'", "'private'", "'protected'",
		"'public'", "'register'", "'reinterpret_cast'", "'return'", "'short'",
		"'signed'", "'sizeof'", "'static'", "'static_assert'", "'static_cast'",
		"'struct'", "'switch'", "'template'", "'this'", "'thread_local'", "'throw'",
		"'true'", "'try'", "'typedef'", "'typeid'", "'typename'", "'union'",
		"'unsigned'", "'using'", "'virtual'", "'void'", "'volatile'", "'wchar_t'",
		"'while'", "'__attribute__'", "'('", "')'", "'['", "']'", "'{'", "'}'",
		"'+'", "'-'", "'*'", "'/'", "'%'", "'^'", "'&'", "'|'", "'~'", "", "'='",
		"'<'", "'>'", "'+='", "'-='", "'*='", "'/='", "'%='", "'^='", "'&='",
		"'|='", "'<<='", "'>>='", "'=='", "'!='", "'<='", "'>='", "", "", "'++'",
		"'--'", "','", "'->*'", "'->'", "'?'", "':'", "'::'", "';'", "'.'",
		"'.*'", "'...'",
	}
	staticData.SymbolicNames = []string{
		"", "IntegerLiteral", "CharacterLiteral", "FloatingLiteral", "StringLiteral",
		"BooleanLiteral", "PointerLiteral", "UserDefinedLiteral", "MultiLineMacro",
		"Directive", "Alignas", "Alignof", "Asm", "Auto", "Bool", "Break", "Case",
		"Catch", "Char", "Char16", "Char32", "Class", "Const", "Constexpr",
		"Const_cast", "Continue", "Decltype", "Default", "Delete", "Do", "Double",
		"Dynamic_cast", "Else", "Enum", "Explicit", "Export", "Extern", "False_",
		"Final", "Float", "For", "Friend", "Goto", "If", "Inline", "Int", "Long",
		"Mutable", "Namespace", "New", "Noexcept", "Nullptr", "Operator", "Override",
		"Private", "Protected", "Public", "Register", "Reinterpret_cast", "Return",
		"Short", "Signed", "Sizeof", "Static", "Static_assert", "Static_cast",
		"Struct", "Switch", "Template", "This", "Thread_local", "Throw", "True_",
		"Try", "Typedef", "Typeid_", "Typename_", "Union", "Unsigned", "Using",
		"Virtual", "Void", "Volatile", "Wchar", "While", "Attribute", "LeftParen",
		"RightParen", "LeftBracket", "RightBracket", "LeftBrace", "RightBrace",
		"Plus", "Minus", "Star", "Div", "Mod", "Caret", "And", "Or", "Tilde",
		"Not", "Assign", "Less", "Greater", "PlusAssign", "MinusAssign", "StarAssign",
		"DivAssign", "ModAssign", "XorAssign", "AndAssign", "OrAssign", "LeftShiftAssign",
		"RightShiftAssign", "Equal", "NotEqual", "LessEqual", "GreaterEqual",
		"AndAnd", "OrOr", "PlusPlus", "MinusMinus", "Comma", "ArrowStar", "Arrow",
		"Question", "Colon", "Doublecolon", "Semi", "Dot", "DotStar", "Ellipsis",
		"Identifier", "DecimalLiteral", "OctalLiteral", "HexadecimalLiteral",
		"BinaryLiteral", "Integersuffix", "UserDefinedIntegerLiteral", "UserDefinedFloatingLiteral",
		"UserDefinedStringLiteral", "UserDefinedCharacterLiteral", "Whitespace",
		"Newline", "BlockComment", "LineComment",
	}
	staticData.RuleNames = []string{
		"cpp", "stmt", "number", "string", "identifier", "comment", "other",
		"keyword", "baseType", "operator",
	}
	staticData.PredictionContextCache = antlr.NewPredictionContextCache()
	staticData.serializedATN = []int32{
		4, 1, 146, 57, 2, 0, 7, 0, 2, 1, 7, 1, 2, 2, 7, 2, 2, 3, 7, 3, 2, 4, 7,
		4, 2, 5, 7, 5, 2, 6, 7, 6, 2, 7, 7, 7, 2, 8, 7, 8, 2, 9, 7, 9, 1, 0, 5,
		0, 22, 8, 0, 10, 0, 12, 0, 25, 9, 0, 1, 0, 1, 0, 1, 1, 1, 1, 1, 1, 1, 1,
		1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 3, 1, 39, 8, 1, 1, 2, 1, 2, 1, 3, 1,
		3, 1, 4, 1, 4, 1, 5, 1, 5, 1, 6, 1, 6, 1, 7, 1, 7, 1, 8, 1, 8, 1, 9, 1,
		9, 1, 9, 0, 0, 10, 0, 2, 4, 6, 8, 10, 12, 14, 16, 18, 0, 6, 3, 0, 1, 1,
		3, 3, 5, 7, 3, 0, 2, 2, 4, 4, 141, 141, 1, 0, 145, 146, 14, 0, 13, 13,
		21, 23, 33, 34, 38, 38, 40, 41, 48, 48, 53, 59, 62, 62, 64, 64, 66, 66,
		68, 68, 74, 74, 77, 77, 79, 80, 8, 0, 14, 14, 18, 18, 30, 30, 39, 39, 45,
		46, 60, 60, 81, 81, 83, 83, 1, 0, 92, 122, 56, 0, 23, 1, 0, 0, 0, 2, 38,
		1, 0, 0, 0, 4, 40, 1, 0, 0, 0, 6, 42, 1, 0, 0, 0, 8, 44, 1, 0, 0, 0, 10,
		46, 1, 0, 0, 0, 12, 48, 1, 0, 0, 0, 14, 50, 1, 0, 0, 0, 16, 52, 1, 0, 0,
		0, 18, 54, 1, 0, 0, 0, 20, 22, 3, 2, 1, 0, 21, 20, 1, 0, 0, 0, 22, 25,
		1, 0, 0, 0, 23, 21, 1, 0, 0, 0, 23, 24, 1, 0, 0, 0, 24, 26, 1, 0, 0, 0,
		25, 23, 1, 0, 0, 0, 26, 27, 5, 0, 0, 1, 27, 1, 1, 0, 0, 0, 28, 39, 5, 9,
		0, 0, 29, 39, 5, 8, 0, 0, 30, 39, 3, 14, 7, 0, 31, 39, 3, 16, 8, 0, 32,
		39, 3, 18, 9, 0, 33, 39, 3, 6, 3, 0, 34, 39, 3, 4, 2, 0, 35, 39, 3, 8,
		4, 0, 36, 39, 3, 10, 5, 0, 37, 39, 3, 12, 6, 0, 38, 28, 1, 0, 0, 0, 38,
		29, 1, 0, 0, 0, 38, 30, 1, 0, 0, 0, 38, 31, 1, 0, 0, 0, 38, 32, 1, 0, 0,
		0, 38, 33, 1, 0, 0, 0, 38, 34, 1, 0, 0, 0, 38, 35, 1, 0, 0, 0, 38, 36,
		1, 0, 0, 0, 38, 37, 1, 0, 0, 0, 39, 3, 1, 0, 0, 0, 40, 41, 7, 0, 0, 0,
		41, 5, 1, 0, 0, 0, 42, 43, 7, 1, 0, 0, 43, 7, 1, 0, 0, 0, 44, 45, 5, 133,
		0, 0, 45, 9, 1, 0, 0, 0, 46, 47, 7, 2, 0, 0, 47, 11, 1, 0, 0, 0, 48, 49,
		9, 0, 0, 0, 49, 13, 1, 0, 0, 0, 50, 51, 7, 3, 0, 0, 51, 15, 1, 0, 0, 0,
		52, 53, 7, 4, 0, 0, 53, 17, 1, 0, 0, 0, 54, 55, 7, 5, 0, 0, 55, 19, 1,
		0, 0, 0, 2, 23, 38,
	}
	deserializer := antlr.NewATNDeserializer(nil)
	staticData.atn = deserializer.Deserialize(staticData.serializedATN)
	atn := staticData.atn
	staticData.decisionToDFA = make([]*antlr.DFA, len(atn.DecisionToState))
	decisionToDFA := staticData.decisionToDFA
	for index, state := range atn.DecisionToState {
		decisionToDFA[index] = antlr.NewDFA(state, index)
	}
}

// CppParserInit initializes any static state used to implement CppParser. By default the
// static state used to implement the parser is lazily initialized during the first call to
// NewCppParser(). You can call this function if you wish to initialize the static state ahead
// of time.
func CppParserInit() {
	staticData := &CppParserStaticData
	staticData.once.Do(cppParserInit)
}

// NewCppParser produces a new parser instance for the optional input antlr.TokenStream.
func NewCppParser(input antlr.TokenStream) *CppParser {
	CppParserInit()
	this := new(CppParser)
	this.BaseParser = antlr.NewBaseParser(input)
	staticData := &CppParserStaticData
	this.Interpreter = antlr.NewParserATNSimulator(this, staticData.atn, staticData.decisionToDFA, staticData.PredictionContextCache)
	this.RuleNames = staticData.RuleNames
	this.LiteralNames = staticData.LiteralNames
	this.SymbolicNames = staticData.SymbolicNames
	this.GrammarFileName = "Cpp.g4"

	return this
}

// CppParser tokens.
const (
	CppParserEOF                         = antlr.TokenEOF
	CppParserIntegerLiteral              = 1
	CppParserCharacterLiteral            = 2
	CppParserFloatingLiteral             = 3
	CppParserStringLiteral               = 4
	CppParserBooleanLiteral              = 5
	CppParserPointerLiteral              = 6
	CppParserUserDefinedLiteral          = 7
	CppParserMultiLineMacro              = 8
	CppParserDirective                   = 9
	CppParserAlignas                     = 10
	CppParserAlignof                     = 11
	CppParserAsm                         = 12
	CppParserAuto                        = 13
	CppParserBool                        = 14
	CppParserBreak                       = 15
	CppParserCase                        = 16
	CppParserCatch                       = 17
	CppParserChar                        = 18
	CppParserChar16                      = 19
	CppParserChar32                      = 20
	CppParserClass                       = 21
	CppParserConst                       = 22
	CppParserConstexpr                   = 23
	CppParserConst_cast                  = 24
	CppParserContinue                    = 25
	CppParserDecltype                    = 26
	CppParserDefault                     = 27
	CppParserDelete                      = 28
	CppParserDo                          = 29
	CppParserDouble                      = 30
	CppParserDynamic_cast                = 31
	CppParserElse                        = 32
	CppParserEnum                        = 33
	CppParserExplicit                    = 34
	CppParserExport                      = 35
	CppParserExtern                      = 36
	CppParserFalse_                      = 37
	CppParserFinal                       = 38
	CppParserFloat                       = 39
	CppParserFor                         = 40
	CppParserFriend                      = 41
	CppParserGoto                        = 42
	CppParserIf                          = 43
	CppParserInline                      = 44
	CppParserInt                         = 45
	CppParserLong                        = 46
	CppParserMutable                     = 47
	CppParserNamespace                   = 48
	CppParserNew                         = 49
	CppParserNoexcept                    = 50
	CppParserNullptr                     = 51
	CppParserOperator                    = 52
	CppParserOverride                    = 53
	CppParserPrivate                     = 54
	CppParserProtected                   = 55
	CppParserPublic                      = 56
	CppParserRegister                    = 57
	CppParserReinterpret_cast            = 58
	CppParserReturn                      = 59
	CppParserShort                       = 60
	CppParserSigned                      = 61
	CppParserSizeof                      = 62
	CppParserStatic                      = 63
	CppParserStatic_assert               = 64
	CppParserStatic_cast                 = 65
	CppParserStruct                      = 66
	CppParserSwitch                      = 67
	CppParserTemplate                    = 68
	CppParserThis                        = 69
	CppParserThread_local                = 70
	CppParserThrow                       = 71
	CppParserTrue_                       = 72
	CppParserTry                         = 73
	CppParserTypedef                     = 74
	CppParserTypeid_                     = 75
	CppParserTypename_                   = 76
	CppParserUnion                       = 77
	CppParserUnsigned                    = 78
	CppParserUsing                       = 79
	CppParserVirtual                     = 80
	CppParserVoid                        = 81
	CppParserVolatile                    = 82
	CppParserWchar                       = 83
	CppParserWhile                       = 84
	CppParserAttribute                   = 85
	CppParserLeftParen                   = 86
	CppParserRightParen                  = 87
	CppParserLeftBracket                 = 88
	CppParserRightBracket                = 89
	CppParserLeftBrace                   = 90
	CppParserRightBrace                  = 91
	CppParserPlus                        = 92
	CppParserMinus                       = 93
	CppParserStar                        = 94
	CppParserDiv                         = 95
	CppParserMod                         = 96
	CppParserCaret                       = 97
	CppParserAnd                         = 98
	CppParserOr                          = 99
	CppParserTilde                       = 100
	CppParserNot                         = 101
	CppParserAssign                      = 102
	CppParserLess                        = 103
	CppParserGreater                     = 104
	CppParserPlusAssign                  = 105
	CppParserMinusAssign                 = 106
	CppParserStarAssign                  = 107
	CppParserDivAssign                   = 108
	CppParserModAssign                   = 109
	CppParserXorAssign                   = 110
	CppParserAndAssign                   = 111
	CppParserOrAssign                    = 112
	CppParserLeftShiftAssign             = 113
	CppParserRightShiftAssign            = 114
	CppParserEqual                       = 115
	CppParserNotEqual                    = 116
	CppParserLessEqual                   = 117
	CppParserGreaterEqual                = 118
	CppParserAndAnd                      = 119
	CppParserOrOr                        = 120
	CppParserPlusPlus                    = 121
	CppParserMinusMinus                  = 122
	CppParserComma                       = 123
	CppParserArrowStar                   = 124
	CppParserArrow                       = 125
	CppParserQuestion                    = 126
	CppParserColon                       = 127
	CppParserDoublecolon                 = 128
	CppParserSemi                        = 129
	CppParserDot                         = 130
	CppParserDotStar                     = 131
	CppParserEllipsis                    = 132
	CppParserIdentifier                  = 133
	CppParserDecimalLiteral              = 134
	CppParserOctalLiteral                = 135
	CppParserHexadecimalLiteral          = 136
	CppParserBinaryLiteral               = 137
	CppParserIntegersuffix               = 138
	CppParserUserDefinedIntegerLiteral   = 139
	CppParserUserDefinedFloatingLiteral  = 140
	CppParserUserDefinedStringLiteral    = 141
	CppParserUserDefinedCharacterLiteral = 142
	CppParserWhitespace                  = 143
	CppParserNewline                     = 144
	CppParserBlockComment                = 145
	CppParserLineComment                 = 146
)

// CppParser rules.
const (
	CppParserRULE_cpp        = 0
	CppParserRULE_stmt       = 1
	CppParserRULE_number     = 2
	CppParserRULE_string     = 3
	CppParserRULE_identifier = 4
	CppParserRULE_comment    = 5
	CppParserRULE_other      = 6
	CppParserRULE_keyword    = 7
	CppParserRULE_baseType   = 8
	CppParserRULE_operator   = 9
)

// ICppContext is an interface to support dynamic dispatch.
type ICppContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	EOF() antlr.TerminalNode
	AllStmt() []IStmtContext
	Stmt(i int) IStmtContext

	// IsCppContext differentiates from other interfaces.
	IsCppContext()
}

type CppContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyCppContext() *CppContext {
	var p = new(CppContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = CppParserRULE_cpp
	return p
}

func InitEmptyCppContext(p *CppContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = CppParserRULE_cpp
}

func (*CppContext) IsCppContext() {}

func NewCppContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *CppContext {
	var p = new(CppContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = CppParserRULE_cpp

	return p
}

func (s *CppContext) GetParser() antlr.Parser { return s.parser }

func (s *CppContext) EOF() antlr.TerminalNode {
	return s.GetToken(CppParserEOF, 0)
}

func (s *CppContext) AllStmt() []IStmtContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IStmtContext); ok {
			len++
		}
	}

	tst := make([]IStmtContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IStmtContext); ok {
			tst[i] = t.(IStmtContext)
			i++
		}
	}

	return tst
}

func (s *CppContext) Stmt(i int) IStmtContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IStmtContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IStmtContext)
}

func (s *CppContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *CppContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *CppContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(CppListener); ok {
		listenerT.EnterCpp(s)
	}
}

func (s *CppContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(CppListener); ok {
		listenerT.ExitCpp(s)
	}
}

func (p *CppParser) Cpp() (localctx ICppContext) {
	localctx = NewCppContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 0, CppParserRULE_cpp)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(23)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for ((int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&-2) != 0) || ((int64((_la-64)) & ^0x3f) == 0 && ((int64(1)<<(_la-64))&-1) != 0) || ((int64((_la-128)) & ^0x3f) == 0 && ((int64(1)<<(_la-128))&524287) != 0) {
		{
			p.SetState(20)
			p.Stmt()
		}

		p.SetState(25)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(26)
		p.Match(CppParserEOF)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IStmtContext is an interface to support dynamic dispatch.
type IStmtContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	Directive() antlr.TerminalNode
	MultiLineMacro() antlr.TerminalNode
	Keyword() IKeywordContext
	BaseType() IBaseTypeContext
	Operator() IOperatorContext
	String_() IStringContext
	Number() INumberContext
	Identifier() IIdentifierContext
	Comment() ICommentContext
	Other() IOtherContext

	// IsStmtContext differentiates from other interfaces.
	IsStmtContext()
}

type StmtContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyStmtContext() *StmtContext {
	var p = new(StmtContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = CppParserRULE_stmt
	return p
}

func InitEmptyStmtContext(p *StmtContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = CppParserRULE_stmt
}

func (*StmtContext) IsStmtContext() {}

func NewStmtContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *StmtContext {
	var p = new(StmtContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = CppParserRULE_stmt

	return p
}

func (s *StmtContext) GetParser() antlr.Parser { return s.parser }

func (s *StmtContext) Directive() antlr.TerminalNode {
	return s.GetToken(CppParserDirective, 0)
}

func (s *StmtContext) MultiLineMacro() antlr.TerminalNode {
	return s.GetToken(CppParserMultiLineMacro, 0)
}

func (s *StmtContext) Keyword() IKeywordContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IKeywordContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IKeywordContext)
}

func (s *StmtContext) BaseType() IBaseTypeContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IBaseTypeContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IBaseTypeContext)
}

func (s *StmtContext) Operator() IOperatorContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IOperatorContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IOperatorContext)
}

func (s *StmtContext) String_() IStringContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IStringContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IStringContext)
}

func (s *StmtContext) Number() INumberContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(INumberContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(INumberContext)
}

func (s *StmtContext) Identifier() IIdentifierContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IIdentifierContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IIdentifierContext)
}

func (s *StmtContext) Comment() ICommentContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ICommentContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ICommentContext)
}

func (s *StmtContext) Other() IOtherContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IOtherContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IOtherContext)
}

func (s *StmtContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *StmtContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *StmtContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(CppListener); ok {
		listenerT.EnterStmt(s)
	}
}

func (s *StmtContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(CppListener); ok {
		listenerT.ExitStmt(s)
	}
}

func (p *CppParser) Stmt() (localctx IStmtContext) {
	localctx = NewStmtContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 2, CppParserRULE_stmt)
	p.SetState(38)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 1, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(28)
			p.Match(CppParserDirective)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(29)
			p.Match(CppParserMultiLineMacro)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 3:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(30)
			p.Keyword()
		}

	case 4:
		p.EnterOuterAlt(localctx, 4)
		{
			p.SetState(31)
			p.BaseType()
		}

	case 5:
		p.EnterOuterAlt(localctx, 5)
		{
			p.SetState(32)
			p.Operator()
		}

	case 6:
		p.EnterOuterAlt(localctx, 6)
		{
			p.SetState(33)
			p.String_()
		}

	case 7:
		p.EnterOuterAlt(localctx, 7)
		{
			p.SetState(34)
			p.Number()
		}

	case 8:
		p.EnterOuterAlt(localctx, 8)
		{
			p.SetState(35)
			p.Identifier()
		}

	case 9:
		p.EnterOuterAlt(localctx, 9)
		{
			p.SetState(36)
			p.Comment()
		}

	case 10:
		p.EnterOuterAlt(localctx, 10)
		{
			p.SetState(37)
			p.Other()
		}

	case antlr.ATNInvalidAltNumber:
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// INumberContext is an interface to support dynamic dispatch.
type INumberContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	IntegerLiteral() antlr.TerminalNode
	FloatingLiteral() antlr.TerminalNode
	BooleanLiteral() antlr.TerminalNode
	PointerLiteral() antlr.TerminalNode
	UserDefinedLiteral() antlr.TerminalNode

	// IsNumberContext differentiates from other interfaces.
	IsNumberContext()
}

type NumberContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyNumberContext() *NumberContext {
	var p = new(NumberContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = CppParserRULE_number
	return p
}

func InitEmptyNumberContext(p *NumberContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = CppParserRULE_number
}

func (*NumberContext) IsNumberContext() {}

func NewNumberContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *NumberContext {
	var p = new(NumberContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = CppParserRULE_number

	return p
}

func (s *NumberContext) GetParser() antlr.Parser { return s.parser }

func (s *NumberContext) IntegerLiteral() antlr.TerminalNode {
	return s.GetToken(CppParserIntegerLiteral, 0)
}

func (s *NumberContext) FloatingLiteral() antlr.TerminalNode {
	return s.GetToken(CppParserFloatingLiteral, 0)
}

func (s *NumberContext) BooleanLiteral() antlr.TerminalNode {
	return s.GetToken(CppParserBooleanLiteral, 0)
}

func (s *NumberContext) PointerLiteral() antlr.TerminalNode {
	return s.GetToken(CppParserPointerLiteral, 0)
}

func (s *NumberContext) UserDefinedLiteral() antlr.TerminalNode {
	return s.GetToken(CppParserUserDefinedLiteral, 0)
}

func (s *NumberContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *NumberContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *NumberContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(CppListener); ok {
		listenerT.EnterNumber(s)
	}
}

func (s *NumberContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(CppListener); ok {
		listenerT.ExitNumber(s)
	}
}

func (p *CppParser) Number() (localctx INumberContext) {
	localctx = NewNumberContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 4, CppParserRULE_number)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(40)
		_la = p.GetTokenStream().LA(1)

		if !((int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&234) != 0) {
			p.GetErrorHandler().RecoverInline(p)
		} else {
			p.GetErrorHandler().ReportMatch(p)
			p.Consume()
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IStringContext is an interface to support dynamic dispatch.
type IStringContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	StringLiteral() antlr.TerminalNode
	CharacterLiteral() antlr.TerminalNode
	UserDefinedStringLiteral() antlr.TerminalNode

	// IsStringContext differentiates from other interfaces.
	IsStringContext()
}

type StringContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyStringContext() *StringContext {
	var p = new(StringContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = CppParserRULE_string
	return p
}

func InitEmptyStringContext(p *StringContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = CppParserRULE_string
}

func (*StringContext) IsStringContext() {}

func NewStringContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *StringContext {
	var p = new(StringContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = CppParserRULE_string

	return p
}

func (s *StringContext) GetParser() antlr.Parser { return s.parser }

func (s *StringContext) StringLiteral() antlr.TerminalNode {
	return s.GetToken(CppParserStringLiteral, 0)
}

func (s *StringContext) CharacterLiteral() antlr.TerminalNode {
	return s.GetToken(CppParserCharacterLiteral, 0)
}

func (s *StringContext) UserDefinedStringLiteral() antlr.TerminalNode {
	return s.GetToken(CppParserUserDefinedStringLiteral, 0)
}

func (s *StringContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *StringContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *StringContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(CppListener); ok {
		listenerT.EnterString(s)
	}
}

func (s *StringContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(CppListener); ok {
		listenerT.ExitString(s)
	}
}

func (p *CppParser) String_() (localctx IStringContext) {
	localctx = NewStringContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 6, CppParserRULE_string)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(42)
		_la = p.GetTokenStream().LA(1)

		if !(_la == CppParserCharacterLiteral || _la == CppParserStringLiteral || _la == CppParserUserDefinedStringLiteral) {
			p.GetErrorHandler().RecoverInline(p)
		} else {
			p.GetErrorHandler().ReportMatch(p)
			p.Consume()
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IIdentifierContext is an interface to support dynamic dispatch.
type IIdentifierContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	Identifier() antlr.TerminalNode

	// IsIdentifierContext differentiates from other interfaces.
	IsIdentifierContext()
}

type IdentifierContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyIdentifierContext() *IdentifierContext {
	var p = new(IdentifierContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = CppParserRULE_identifier
	return p
}

func InitEmptyIdentifierContext(p *IdentifierContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = CppParserRULE_identifier
}

func (*IdentifierContext) IsIdentifierContext() {}

func NewIdentifierContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *IdentifierContext {
	var p = new(IdentifierContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = CppParserRULE_identifier

	return p
}

func (s *IdentifierContext) GetParser() antlr.Parser { return s.parser }

func (s *IdentifierContext) Identifier() antlr.TerminalNode {
	return s.GetToken(CppParserIdentifier, 0)
}

func (s *IdentifierContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *IdentifierContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *IdentifierContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(CppListener); ok {
		listenerT.EnterIdentifier(s)
	}
}

func (s *IdentifierContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(CppListener); ok {
		listenerT.ExitIdentifier(s)
	}
}

func (p *CppParser) Identifier() (localctx IIdentifierContext) {
	localctx = NewIdentifierContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 8, CppParserRULE_identifier)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(44)
		p.Match(CppParserIdentifier)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// ICommentContext is an interface to support dynamic dispatch.
type ICommentContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	LineComment() antlr.TerminalNode
	BlockComment() antlr.TerminalNode

	// IsCommentContext differentiates from other interfaces.
	IsCommentContext()
}

type CommentContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyCommentContext() *CommentContext {
	var p = new(CommentContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = CppParserRULE_comment
	return p
}

func InitEmptyCommentContext(p *CommentContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = CppParserRULE_comment
}

func (*CommentContext) IsCommentContext() {}

func NewCommentContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *CommentContext {
	var p = new(CommentContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = CppParserRULE_comment

	return p
}

func (s *CommentContext) GetParser() antlr.Parser { return s.parser }

func (s *CommentContext) LineComment() antlr.TerminalNode {
	return s.GetToken(CppParserLineComment, 0)
}

func (s *CommentContext) BlockComment() antlr.TerminalNode {
	return s.GetToken(CppParserBlockComment, 0)
}

func (s *CommentContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *CommentContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *CommentContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(CppListener); ok {
		listenerT.EnterComment(s)
	}
}

func (s *CommentContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(CppListener); ok {
		listenerT.ExitComment(s)
	}
}

func (p *CppParser) Comment() (localctx ICommentContext) {
	localctx = NewCommentContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 10, CppParserRULE_comment)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(46)
		_la = p.GetTokenStream().LA(1)

		if !(_la == CppParserBlockComment || _la == CppParserLineComment) {
			p.GetErrorHandler().RecoverInline(p)
		} else {
			p.GetErrorHandler().ReportMatch(p)
			p.Consume()
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IOtherContext is an interface to support dynamic dispatch.
type IOtherContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser
	// IsOtherContext differentiates from other interfaces.
	IsOtherContext()
}

type OtherContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyOtherContext() *OtherContext {
	var p = new(OtherContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = CppParserRULE_other
	return p
}

func InitEmptyOtherContext(p *OtherContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = CppParserRULE_other
}

func (*OtherContext) IsOtherContext() {}

func NewOtherContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *OtherContext {
	var p = new(OtherContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = CppParserRULE_other

	return p
}

func (s *OtherContext) GetParser() antlr.Parser { return s.parser }
func (s *OtherContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *OtherContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *OtherContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(CppListener); ok {
		listenerT.EnterOther(s)
	}
}

func (s *OtherContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(CppListener); ok {
		listenerT.ExitOther(s)
	}
}

func (p *CppParser) Other() (localctx IOtherContext) {
	localctx = NewOtherContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 12, CppParserRULE_other)
	p.EnterOuterAlt(localctx, 1)
	p.SetState(48)
	p.MatchWildcard()

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IKeywordContext is an interface to support dynamic dispatch.
type IKeywordContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	Typedef() antlr.TerminalNode
	Using() antlr.TerminalNode
	Namespace() antlr.TerminalNode
	Class() antlr.TerminalNode
	Struct() antlr.TerminalNode
	Union() antlr.TerminalNode
	Enum() antlr.TerminalNode
	Template() antlr.TerminalNode
	Friend() antlr.TerminalNode
	Public() antlr.TerminalNode
	Private() antlr.TerminalNode
	Protected() antlr.TerminalNode
	Virtual() antlr.TerminalNode
	Override() antlr.TerminalNode
	Final() antlr.TerminalNode
	Explicit() antlr.TerminalNode
	Constexpr() antlr.TerminalNode
	Static_assert() antlr.TerminalNode
	Return() antlr.TerminalNode
	Auto() antlr.TerminalNode
	Sizeof() antlr.TerminalNode
	Reinterpret_cast() antlr.TerminalNode
	Register() antlr.TerminalNode
	Const() antlr.TerminalNode
	For() antlr.TerminalNode

	// IsKeywordContext differentiates from other interfaces.
	IsKeywordContext()
}

type KeywordContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyKeywordContext() *KeywordContext {
	var p = new(KeywordContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = CppParserRULE_keyword
	return p
}

func InitEmptyKeywordContext(p *KeywordContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = CppParserRULE_keyword
}

func (*KeywordContext) IsKeywordContext() {}

func NewKeywordContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *KeywordContext {
	var p = new(KeywordContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = CppParserRULE_keyword

	return p
}

func (s *KeywordContext) GetParser() antlr.Parser { return s.parser }

func (s *KeywordContext) Typedef() antlr.TerminalNode {
	return s.GetToken(CppParserTypedef, 0)
}

func (s *KeywordContext) Using() antlr.TerminalNode {
	return s.GetToken(CppParserUsing, 0)
}

func (s *KeywordContext) Namespace() antlr.TerminalNode {
	return s.GetToken(CppParserNamespace, 0)
}

func (s *KeywordContext) Class() antlr.TerminalNode {
	return s.GetToken(CppParserClass, 0)
}

func (s *KeywordContext) Struct() antlr.TerminalNode {
	return s.GetToken(CppParserStruct, 0)
}

func (s *KeywordContext) Union() antlr.TerminalNode {
	return s.GetToken(CppParserUnion, 0)
}

func (s *KeywordContext) Enum() antlr.TerminalNode {
	return s.GetToken(CppParserEnum, 0)
}

func (s *KeywordContext) Template() antlr.TerminalNode {
	return s.GetToken(CppParserTemplate, 0)
}

func (s *KeywordContext) Friend() antlr.TerminalNode {
	return s.GetToken(CppParserFriend, 0)
}

func (s *KeywordContext) Public() antlr.TerminalNode {
	return s.GetToken(CppParserPublic, 0)
}

func (s *KeywordContext) Private() antlr.TerminalNode {
	return s.GetToken(CppParserPrivate, 0)
}

func (s *KeywordContext) Protected() antlr.TerminalNode {
	return s.GetToken(CppParserProtected, 0)
}

func (s *KeywordContext) Virtual() antlr.TerminalNode {
	return s.GetToken(CppParserVirtual, 0)
}

func (s *KeywordContext) Override() antlr.TerminalNode {
	return s.GetToken(CppParserOverride, 0)
}

func (s *KeywordContext) Final() antlr.TerminalNode {
	return s.GetToken(CppParserFinal, 0)
}

func (s *KeywordContext) Explicit() antlr.TerminalNode {
	return s.GetToken(CppParserExplicit, 0)
}

func (s *KeywordContext) Constexpr() antlr.TerminalNode {
	return s.GetToken(CppParserConstexpr, 0)
}

func (s *KeywordContext) Static_assert() antlr.TerminalNode {
	return s.GetToken(CppParserStatic_assert, 0)
}

func (s *KeywordContext) Return() antlr.TerminalNode {
	return s.GetToken(CppParserReturn, 0)
}

func (s *KeywordContext) Auto() antlr.TerminalNode {
	return s.GetToken(CppParserAuto, 0)
}

func (s *KeywordContext) Sizeof() antlr.TerminalNode {
	return s.GetToken(CppParserSizeof, 0)
}

func (s *KeywordContext) Reinterpret_cast() antlr.TerminalNode {
	return s.GetToken(CppParserReinterpret_cast, 0)
}

func (s *KeywordContext) Register() antlr.TerminalNode {
	return s.GetToken(CppParserRegister, 0)
}

func (s *KeywordContext) Const() antlr.TerminalNode {
	return s.GetToken(CppParserConst, 0)
}

func (s *KeywordContext) For() antlr.TerminalNode {
	return s.GetToken(CppParserFor, 0)
}

func (s *KeywordContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *KeywordContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *KeywordContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(CppListener); ok {
		listenerT.EnterKeyword(s)
	}
}

func (s *KeywordContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(CppListener); ok {
		listenerT.ExitKeyword(s)
	}
}

func (p *CppParser) Keyword() (localctx IKeywordContext) {
	localctx = NewKeywordContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 14, CppParserRULE_keyword)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(50)
		_la = p.GetTokenStream().LA(1)

		if !(((int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&5755885397953486848) != 0) || ((int64((_la-64)) & ^0x3f) == 0 && ((int64(1)<<(_la-64))&107541) != 0)) {
			p.GetErrorHandler().RecoverInline(p)
		} else {
			p.GetErrorHandler().ReportMatch(p)
			p.Consume()
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IBaseTypeContext is an interface to support dynamic dispatch.
type IBaseTypeContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	Bool() antlr.TerminalNode
	Char() antlr.TerminalNode
	Wchar() antlr.TerminalNode
	Short() antlr.TerminalNode
	Int() antlr.TerminalNode
	Long() antlr.TerminalNode
	Float() antlr.TerminalNode
	Double() antlr.TerminalNode
	Void() antlr.TerminalNode

	// IsBaseTypeContext differentiates from other interfaces.
	IsBaseTypeContext()
}

type BaseTypeContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyBaseTypeContext() *BaseTypeContext {
	var p = new(BaseTypeContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = CppParserRULE_baseType
	return p
}

func InitEmptyBaseTypeContext(p *BaseTypeContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = CppParserRULE_baseType
}

func (*BaseTypeContext) IsBaseTypeContext() {}

func NewBaseTypeContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *BaseTypeContext {
	var p = new(BaseTypeContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = CppParserRULE_baseType

	return p
}

func (s *BaseTypeContext) GetParser() antlr.Parser { return s.parser }

func (s *BaseTypeContext) Bool() antlr.TerminalNode {
	return s.GetToken(CppParserBool, 0)
}

func (s *BaseTypeContext) Char() antlr.TerminalNode {
	return s.GetToken(CppParserChar, 0)
}

func (s *BaseTypeContext) Wchar() antlr.TerminalNode {
	return s.GetToken(CppParserWchar, 0)
}

func (s *BaseTypeContext) Short() antlr.TerminalNode {
	return s.GetToken(CppParserShort, 0)
}

func (s *BaseTypeContext) Int() antlr.TerminalNode {
	return s.GetToken(CppParserInt, 0)
}

func (s *BaseTypeContext) Long() antlr.TerminalNode {
	return s.GetToken(CppParserLong, 0)
}

func (s *BaseTypeContext) Float() antlr.TerminalNode {
	return s.GetToken(CppParserFloat, 0)
}

func (s *BaseTypeContext) Double() antlr.TerminalNode {
	return s.GetToken(CppParserDouble, 0)
}

func (s *BaseTypeContext) Void() antlr.TerminalNode {
	return s.GetToken(CppParserVoid, 0)
}

func (s *BaseTypeContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *BaseTypeContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *BaseTypeContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(CppListener); ok {
		listenerT.EnterBaseType(s)
	}
}

func (s *BaseTypeContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(CppListener); ok {
		listenerT.ExitBaseType(s)
	}
}

func (p *CppParser) BaseType() (localctx IBaseTypeContext) {
	localctx = NewBaseTypeContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 16, CppParserRULE_baseType)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(52)
		_la = p.GetTokenStream().LA(1)

		if !(((int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&1153027608552947712) != 0) || _la == CppParserVoid || _la == CppParserWchar) {
			p.GetErrorHandler().RecoverInline(p)
		} else {
			p.GetErrorHandler().ReportMatch(p)
			p.Consume()
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IOperatorContext is an interface to support dynamic dispatch.
type IOperatorContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	Plus() antlr.TerminalNode
	Minus() antlr.TerminalNode
	Star() antlr.TerminalNode
	Div() antlr.TerminalNode
	Mod() antlr.TerminalNode
	Caret() antlr.TerminalNode
	And() antlr.TerminalNode
	Or() antlr.TerminalNode
	Tilde() antlr.TerminalNode
	Not() antlr.TerminalNode
	Assign() antlr.TerminalNode
	Less() antlr.TerminalNode
	Greater() antlr.TerminalNode
	PlusAssign() antlr.TerminalNode
	MinusAssign() antlr.TerminalNode
	StarAssign() antlr.TerminalNode
	DivAssign() antlr.TerminalNode
	ModAssign() antlr.TerminalNode
	XorAssign() antlr.TerminalNode
	AndAssign() antlr.TerminalNode
	OrAssign() antlr.TerminalNode
	LeftShiftAssign() antlr.TerminalNode
	RightShiftAssign() antlr.TerminalNode
	Equal() antlr.TerminalNode
	NotEqual() antlr.TerminalNode
	LessEqual() antlr.TerminalNode
	GreaterEqual() antlr.TerminalNode
	AndAnd() antlr.TerminalNode
	OrOr() antlr.TerminalNode
	PlusPlus() antlr.TerminalNode
	MinusMinus() antlr.TerminalNode

	// IsOperatorContext differentiates from other interfaces.
	IsOperatorContext()
}

type OperatorContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyOperatorContext() *OperatorContext {
	var p = new(OperatorContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = CppParserRULE_operator
	return p
}

func InitEmptyOperatorContext(p *OperatorContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = CppParserRULE_operator
}

func (*OperatorContext) IsOperatorContext() {}

func NewOperatorContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *OperatorContext {
	var p = new(OperatorContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = CppParserRULE_operator

	return p
}

func (s *OperatorContext) GetParser() antlr.Parser { return s.parser }

func (s *OperatorContext) Plus() antlr.TerminalNode {
	return s.GetToken(CppParserPlus, 0)
}

func (s *OperatorContext) Minus() antlr.TerminalNode {
	return s.GetToken(CppParserMinus, 0)
}

func (s *OperatorContext) Star() antlr.TerminalNode {
	return s.GetToken(CppParserStar, 0)
}

func (s *OperatorContext) Div() antlr.TerminalNode {
	return s.GetToken(CppParserDiv, 0)
}

func (s *OperatorContext) Mod() antlr.TerminalNode {
	return s.GetToken(CppParserMod, 0)
}

func (s *OperatorContext) Caret() antlr.TerminalNode {
	return s.GetToken(CppParserCaret, 0)
}

func (s *OperatorContext) And() antlr.TerminalNode {
	return s.GetToken(CppParserAnd, 0)
}

func (s *OperatorContext) Or() antlr.TerminalNode {
	return s.GetToken(CppParserOr, 0)
}

func (s *OperatorContext) Tilde() antlr.TerminalNode {
	return s.GetToken(CppParserTilde, 0)
}

func (s *OperatorContext) Not() antlr.TerminalNode {
	return s.GetToken(CppParserNot, 0)
}

func (s *OperatorContext) Assign() antlr.TerminalNode {
	return s.GetToken(CppParserAssign, 0)
}

func (s *OperatorContext) Less() antlr.TerminalNode {
	return s.GetToken(CppParserLess, 0)
}

func (s *OperatorContext) Greater() antlr.TerminalNode {
	return s.GetToken(CppParserGreater, 0)
}

func (s *OperatorContext) PlusAssign() antlr.TerminalNode {
	return s.GetToken(CppParserPlusAssign, 0)
}

func (s *OperatorContext) MinusAssign() antlr.TerminalNode {
	return s.GetToken(CppParserMinusAssign, 0)
}

func (s *OperatorContext) StarAssign() antlr.TerminalNode {
	return s.GetToken(CppParserStarAssign, 0)
}

func (s *OperatorContext) DivAssign() antlr.TerminalNode {
	return s.GetToken(CppParserDivAssign, 0)
}

func (s *OperatorContext) ModAssign() antlr.TerminalNode {
	return s.GetToken(CppParserModAssign, 0)
}

func (s *OperatorContext) XorAssign() antlr.TerminalNode {
	return s.GetToken(CppParserXorAssign, 0)
}

func (s *OperatorContext) AndAssign() antlr.TerminalNode {
	return s.GetToken(CppParserAndAssign, 0)
}

func (s *OperatorContext) OrAssign() antlr.TerminalNode {
	return s.GetToken(CppParserOrAssign, 0)
}

func (s *OperatorContext) LeftShiftAssign() antlr.TerminalNode {
	return s.GetToken(CppParserLeftShiftAssign, 0)
}

func (s *OperatorContext) RightShiftAssign() antlr.TerminalNode {
	return s.GetToken(CppParserRightShiftAssign, 0)
}

func (s *OperatorContext) Equal() antlr.TerminalNode {
	return s.GetToken(CppParserEqual, 0)
}

func (s *OperatorContext) NotEqual() antlr.TerminalNode {
	return s.GetToken(CppParserNotEqual, 0)
}

func (s *OperatorContext) LessEqual() antlr.TerminalNode {
	return s.GetToken(CppParserLessEqual, 0)
}

func (s *OperatorContext) GreaterEqual() antlr.TerminalNode {
	return s.GetToken(CppParserGreaterEqual, 0)
}

func (s *OperatorContext) AndAnd() antlr.TerminalNode {
	return s.GetToken(CppParserAndAnd, 0)
}

func (s *OperatorContext) OrOr() antlr.TerminalNode {
	return s.GetToken(CppParserOrOr, 0)
}

func (s *OperatorContext) PlusPlus() antlr.TerminalNode {
	return s.GetToken(CppParserPlusPlus, 0)
}

func (s *OperatorContext) MinusMinus() antlr.TerminalNode {
	return s.GetToken(CppParserMinusMinus, 0)
}

func (s *OperatorContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *OperatorContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *OperatorContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(CppListener); ok {
		listenerT.EnterOperator(s)
	}
}

func (s *OperatorContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(CppListener); ok {
		listenerT.ExitOperator(s)
	}
}

func (p *CppParser) Operator() (localctx IOperatorContext) {
	localctx = NewOperatorContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 18, CppParserRULE_operator)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(54)
		_la = p.GetTokenStream().LA(1)

		if !((int64((_la-92)) & ^0x3f) == 0 && ((int64(1)<<(_la-92))&2147483647) != 0) {
			p.GetErrorHandler().RecoverInline(p)
		} else {
			p.GetErrorHandler().ReportMatch(p)
			p.Consume()
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}
