package lsp

import (
	"fmt"
	"strings"
	// "tops-lsp/parser"
)

type AstNode map[string]interface{}

func (n AstNode) Kind() string {
	if n["kind"] == nil {
		return ""
	}
	return n["kind"].(string)
}
func (n AstNode) ID() string {
	if n["id"] == nil {
		return ""
	}
	return n["id"].(string)
}
func (n AstNode) Name() string {
	if n["name"] == nil {
		return ""
	}
	return n["name"].(string)
}
func (n AstNode) Children() []AstNode {
	if n["inner"] == nil {
		return []AstNode{}
	}
	arr := n["inner"].([]interface{})
	nodes := make([]AstNode, len(arr))
	for i, v := range arr {
		nodes[i] = v.(map[string]interface{})
	}
	return nodes
}

type AstNodeLoc map[string]interface{}

func (n AstNode) Loc() AstNodeLoc {
	if n["loc"] == nil {
		return nil
	}
	return n["loc"].(map[string]interface{})
}

func (n AstNodeLoc) File() string {
	if n["file"] == nil {
		return ""
	}
	return n["file"].(string)
}
func (n AstNodeLoc) Line() int {
	if n["line"] == nil {
		return 0
	}
	return int(n["line"].(float64))
}
func (n AstNodeLoc) Col() int {
	if n["col"] == nil {
		return 0
	}
	return int(n["col"].(float64))
}
func (n AstNodeLoc) TokLen() int {
	if n["tokLen"] == nil {
		return 0
	}
	return int(n["tokLen"].(float64))
}

type AstVisitor interface {
	EnterVarDecl(node AstNode)
	ExitVarDecl(node AstNode)
	EnterFunctionDecl(node AstNode)
	ExitFunctionDecl(node AstNode)
}

type AstVisitorBase struct{}

func (v *AstVisitorBase) EnterVarDecl(node AstNode) {
	// 默认实现
}
func (v *AstVisitorBase) ExitVarDecl(node AstNode) {
	// 默认实现
}
func (v *AstVisitorBase) EnterFunctionDecl(node AstNode) {
	// 默认实现
}
func (v *AstVisitorBase) ExitFunctionDecl(node AstNode) {
	// 默认实现
}

// ForEach applies a given function to each element in a slice of AstVisitor.
func ForEach(visitors []AstVisitor, fn func(AstVisitor)) {
	for _, visitor := range visitors {
		fn(visitor)
	}
}

// 解析Clang生成的AST文本
func ParseClangAstJosnt(astJson map[string]interface{}, listeners []AstVisitor) AstNode {
	var visitor func(node AstNode)
	visitor = func(node AstNode) {
		if node.Kind() == "FunctionDecl" {
			ForEach(listeners, func(l AstVisitor) {
				l.EnterFunctionDecl(node)
			})
			defer ForEach(listeners, func(l AstVisitor) {
				l.ExitFunctionDecl(node)
			})
			// 处理函数声明
		} else if node.Kind() == "VarDecl" {
			ForEach(listeners, func(l AstVisitor) {
				l.EnterVarDecl(node)
			})
			defer ForEach(listeners, func(l AstVisitor) {
				l.ExitVarDecl(node)
			})
			// 处理变量声明
		} else if node.Kind() == "ParmVarDecl" {
			ForEach(listeners, func(l AstVisitor) {
				l.EnterVarDecl(node) // 使用 EnterVarDecl 处理参数变量声明
			})
			defer ForEach(listeners, func(l AstVisitor) {
				l.ExitVarDecl(node) // 使用 ExitVarDecl 处理参数变量声明
			})
		}
		for _, c := range node.Children() {
			visitor(c)
		}
	}
	visitor(astJson)
	return nil
}



// 打印AST树（示例）
func PrintAST(node AstNode, indent int) {
	prefix := strings.Repeat("  ", indent)
	fmt.Printf("%s%s", prefix, node.Kind())
	fmt.Println()

	for _, child := range node.Children() {
		PrintAST(child, indent+1)
	}
}
