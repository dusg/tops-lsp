// Code generated from Cpp.g4 by ANTLR 4.13.2. DO NOT EDIT.

package parser // Cpp

import "github.com/antlr4-go/antlr/v4"

// BaseCppListener is a complete listener for a parse tree produced by CppParser.
type BaseCppListener struct{}

var _ CppListener = &BaseCppListener{}

// VisitTerminal is called when a terminal node is visited.
func (s *BaseCppListener) VisitTerminal(node antlr.TerminalNode) {}

// VisitErrorNode is called when an error node is visited.
func (s *BaseCppListener) VisitErrorNode(node antlr.ErrorNode) {}

// EnterEveryRule is called when any rule is entered.
func (s *BaseCppListener) EnterEveryRule(ctx antlr.ParserRuleContext) {}

// ExitEveryRule is called when any rule is exited.
func (s *BaseCppListener) ExitEveryRule(ctx antlr.ParserRuleContext) {}

// EnterCpp is called when production cpp is entered.
func (s *BaseCppListener) EnterCpp(ctx *CppContext) {}

// ExitCpp is called when production cpp is exited.
func (s *BaseCppListener) ExitCpp(ctx *CppContext) {}

// EnterStmt is called when production stmt is entered.
func (s *BaseCppListener) EnterStmt(ctx *StmtContext) {}

// ExitStmt is called when production stmt is exited.
func (s *BaseCppListener) ExitStmt(ctx *StmtContext) {}

// EnterNumber is called when production number is entered.
func (s *BaseCppListener) EnterNumber(ctx *NumberContext) {}

// ExitNumber is called when production number is exited.
func (s *BaseCppListener) ExitNumber(ctx *NumberContext) {}

// EnterString is called when production string is entered.
func (s *BaseCppListener) EnterString(ctx *StringContext) {}

// ExitString is called when production string is exited.
func (s *BaseCppListener) ExitString(ctx *StringContext) {}

// EnterIdentifier is called when production identifier is entered.
func (s *BaseCppListener) EnterIdentifier(ctx *IdentifierContext) {}

// ExitIdentifier is called when production identifier is exited.
func (s *BaseCppListener) ExitIdentifier(ctx *IdentifierContext) {}

// EnterComment is called when production comment is entered.
func (s *BaseCppListener) EnterComment(ctx *CommentContext) {}

// ExitComment is called when production comment is exited.
func (s *BaseCppListener) ExitComment(ctx *CommentContext) {}

// EnterOther is called when production other is entered.
func (s *BaseCppListener) EnterOther(ctx *OtherContext) {}

// ExitOther is called when production other is exited.
func (s *BaseCppListener) ExitOther(ctx *OtherContext) {}

// EnterKeyword is called when production keyword is entered.
func (s *BaseCppListener) EnterKeyword(ctx *KeywordContext) {}

// ExitKeyword is called when production keyword is exited.
func (s *BaseCppListener) ExitKeyword(ctx *KeywordContext) {}

// EnterBaseType is called when production baseType is entered.
func (s *BaseCppListener) EnterBaseType(ctx *BaseTypeContext) {}

// ExitBaseType is called when production baseType is exited.
func (s *BaseCppListener) ExitBaseType(ctx *BaseTypeContext) {}

// EnterOperator is called when production operator is entered.
func (s *BaseCppListener) EnterOperator(ctx *OperatorContext) {}

// ExitOperator is called when production operator is exited.
func (s *BaseCppListener) ExitOperator(ctx *OperatorContext) {}
