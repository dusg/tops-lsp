package lsp_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os/exec"
	"sync"
	"testing"
	"tops-lsp/lsp"
)

type myVisitor struct {
	lsp.AstVisitorBase
}

func (v *myVisitor) EnterVarDecl(node lsp.AstNode) {
	log.Printf("EnterVarDecl: %s\n", node.Name())
}

func (v *myVisitor) EnterFunctionDecl(node lsp.AstNode) {
	log.Printf("EnterFunctionDecl: %s\n", node.Name())
}

func TestAst(t *testing.T) {
	// 步骤1：生成Clang AST（需提前生成）
	// 示例命令：clang -Xclang -ast-dump -fsyntax-only test.c > ast.txt

	// filePath := "/home/carl.du/work/tops-lsp/test-files/ast.cc"
	filePath := "/home/carl.du/work/tops-lsp/test-files/test.tops"
	args := []string{"-fsyntax-only", "-Xclang", "-ast-dump=json", "-arch", "gcu300", "-ltops", "--cuda-device-only", filePath}
	// args := []string{"-fsyntax-only", "-E", "-arch", "gcu300", "-ltops", "--cuda-device-only", filePath, "-o", "-"}

	cmd := exec.Command("/opt/tops/bin/topscc", args...)
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Failed to get stdout pipe: %v", err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	log.Println("cmd: ", cmd.String())
	if err := cmd.Start(); err != nil {
		fmt.Println("Error:", err)
		fmt.Println("Stderr:", stderr.String())
		t.FailNow()
		return
	}
	decoder := json.NewDecoder(stdoutPipe)
	wg := sync.WaitGroup{}
	wg.Add(1)
	var astNode map[string]interface{}
	go func() {
		defer wg.Done()
		err := decoder.Decode(&astNode)
		if err != nil && err != io.EOF {
			log.Fatalf("Failed to decode JSON: %v", err)
		}
		// 处理解析后的 AST 节点
		log.Printf("Parsed AST Node done\n")
	}()

	if err := cmd.Wait(); err != nil {
		fmt.Println("Error:", err)
		fmt.Println("Stderr:", stderr.String())
	}

	log.Println("run cmd success")
	wg.Wait()

	lsp.ParseClangAstJosnt(astNode, []lsp.AstVisitor{&myVisitor{}})
	// os.WriteFile(filePath+".ast", stdout.Bytes(), 0644)
	// log.Println("save ast file success")
	// 步骤2：解析AST文件
	// ast := lsp.ParseAST(stdout.String(), filePath)
	// ast := lsp.ParseClangAstJosnt(stdout.String(), filePath)

	// 步骤3：遍历特定节点（例如main函数）
	// var mainFunc lsp.ASTNode
	// var visitAST func(prefix string, n lsp.ASTNode)
	// visitAST = func(prefix string, n lsp.ASTNode) {
	// 	if n == nil {
	// 		return
	// 	}
	// 	// fmt.Printf("%s%s\n", prefix, n.Dump())
	// 	for _, child := range n.GetChildren() {
	// 		visitAST(prefix+" ", child)
	// 	}
	// }
	// visitAST("", ast)
	log.Println("Donel")
}

// type MyListener struct {
// 	parser.BaseastListener
// }

// func (l *MyListener) EnterNode(ctx *parser.NodeContext) {
// 	fmt.Println("enter node:", ctx.GetText())
// }
// func TestParser(t *testing.T) {
// 	filePath := "/home/carl.du/work/tops-lsp/test-files/ast.cc"
// 	args := []string{"-fsyntax-only", "-Xclang", "-ast-dump", "-arch", "gcu300", "-ltops", filePath}

// 	cmd := exec.Command("/opt/tops/bin/clang++", args...)
// 	var stderr bytes.Buffer
// 	var stdout bytes.Buffer
// 	cmd.Stderr = &stderr
// 	cmd.Stdout = &stdout
// 	fmt.Println("cmd: ", cmd.String())
// 	err := cmd.Run()
// 	if (err != nil) {
// 		fmt.Println("Error:", err)
// 		fmt.Println("Stderr:", stderr.String())
// 		t.FailNow()
// 		return
// 	}
// 	os.WriteFile(filePath+".ast", stdout.Bytes(), 0644)
// 	is := antlr.NewInputStream(stdout.String())
// 	lexer := parser.NewastLexer(is)
// 	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)

// 	// 创建解析器和监听器
// 	astParser := parser.NewastParser(stream)
// 	antlr.ParseTreeWalkerDefault.Walk(&MyListener{}, astParser.Ast())
// 	// 步骤2：解析AST文件
// }
