// Code generated from Cpp.g4 by ANTLR 4.13.2. DO NOT EDIT.

package parser // Cpp

import "github.com/antlr4-go/antlr/v4"

// CppListener is a complete listener for a parse tree produced by CppParser.
type CppListener interface {
	antlr.ParseTreeListener

	// EnterCpp is called when entering the cpp production.
	EnterCpp(c *CppContext)

	// EnterStmt is called when entering the stmt production.
	EnterStmt(c *StmtContext)

	// EnterNumber is called when entering the number production.
	EnterNumber(c *NumberContext)

	// EnterString is called when entering the string production.
	EnterString(c *StringContext)

	// EnterIdentifier is called when entering the identifier production.
	EnterIdentifier(c *IdentifierContext)

	// EnterComment is called when entering the comment production.
	EnterComment(c *CommentContext)

	// EnterOther is called when entering the other production.
	EnterOther(c *OtherContext)

	// EnterKeyword is called when entering the keyword production.
	EnterKeyword(c *KeywordContext)

	// EnterBaseType is called when entering the baseType production.
	EnterBaseType(c *BaseTypeContext)

	// EnterOperator is called when entering the operator production.
	EnterOperator(c *OperatorContext)

	// ExitCpp is called when exiting the cpp production.
	ExitCpp(c *CppContext)

	// ExitStmt is called when exiting the stmt production.
	ExitStmt(c *StmtContext)

	// ExitNumber is called when exiting the number production.
	ExitNumber(c *NumberContext)

	// ExitString is called when exiting the string production.
	ExitString(c *StringContext)

	// ExitIdentifier is called when exiting the identifier production.
	ExitIdentifier(c *IdentifierContext)

	// ExitComment is called when exiting the comment production.
	ExitComment(c *CommentContext)

	// ExitOther is called when exiting the other production.
	ExitOther(c *OtherContext)

	// ExitKeyword is called when exiting the keyword production.
	ExitKeyword(c *KeywordContext)

	// ExitBaseType is called when exiting the baseType production.
	ExitBaseType(c *BaseTypeContext)

	// ExitOperator is called when exiting the operator production.
	ExitOperator(c *OperatorContext)
}
