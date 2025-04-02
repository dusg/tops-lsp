package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sourcegraph/jsonrpc2"
)

// 编译参数管理
type CompileConfig struct {
	Args    []string `json:"args"`
	Include []string `json:"include"`
	Defines []string `json:"defines"`
}

type ProjectConfig struct {
	CompileCommands map[string]CompileConfig // 文件名到编译配置
	mu              sync.RWMutex
}

var projectConfig = &ProjectConfig{
	CompileCommands: make(map[string]CompileConfig),
}

// LSP处理核心
type ClangLSPHandler struct {
	conn      *jsonrpc2.Conn
	config    *ProjectConfig
	tempFiles map[string]string // 临时文件缓存
}

type DiagnosticSeverity int

const (
	Error       DiagnosticSeverity = 1
	Warning     DiagnosticSeverity = 2
	Information DiagnosticSeverity = 3
	Hint        DiagnosticSeverity = 4
)

// 诊断信息结构
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}
type Diagnostic struct {
	Range struct {
		Start Position `json:"start"`
		End   Position `json:"end"`
	} `json:"range"`
	Severity DiagnosticSeverity `json:"severity"`
	Message  string             `json:"message"`
	Source   string             `json:"source"`
}

// 代码补全结构
type CompletionItem struct {
	Label  string `json:"label"`
	Kind   int    `json:"kind"`
	Detail string `json:"detail,omitempty"`
}

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

func main() {
	ln, err := net.Listen("tcp", "127.0.0.1:4389")
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()

	log.Println("TopsCC-based LSP running on :4389")
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("Accept error:", err)
			continue
		}
		log.Println("Client connected:", conn.RemoteAddr())

		handler := &ClangLSPHandler{tempFiles: make(map[string]string)}
		jsonrpc2.NewConn(context.Background(),
			jsonrpc2.NewBufferedStream(conn, jsonrpc2.VSCodeObjectCodec{}),
			handler)
	}
}

// LSP方法处理
func (h *ClangLSPHandler) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	h.conn = conn

	switch req.Method {
	case "initialize":
		h.handleInitialize(ctx, req)
	case "textDocument/didOpen":
		h.handleDidOpen(ctx, req)
	case "textDocument/didChange":
		h.handleDidChange(ctx, req)
	case "textDocument/completion":
		h.handleCompletion(ctx, req)
	case "textDocument/diagnostic":
		h.handleDiagnostic(ctx, req)
	case "textDocument/definition":
		// h.handleDefinition(ctx, req)
	default:
		conn.ReplyWithError(ctx, req.ID,
			&jsonrpc2.Error{Code: jsonrpc2.CodeMethodNotFound, Message: "method not supported", Data: nil})
	}
}

// 初始化处理
func (h *ClangLSPHandler) handleInitialize(ctx context.Context, req *jsonrpc2.Request) {
	log.Println("初始化请求:", req.ID)
	response := map[string]interface{}{
		"capabilities": map[string]interface{}{
			"textDocumentSync": 1,
			"completionProvider": map[string]interface{}{
				"triggerCharacters": []string{".", "->", "::"},
			},
			"definitionProvider": false,
			"diagnosticProvider": map[string]interface{}{
				"identifier":            "topscc",
				"interFileDependencies": true,
				"workspaceDiagnostics":  false,
			},
		},
	}
	h.conn.Reply(ctx, req.ID, response)
}

func (h *ClangLSPHandler) publishDiagnostics(uri string, diagnostics []Diagnostic) {
	log.Println("发布诊断信息:", uri)

	h.conn.Notify(context.Background(), "textDocument/publishDiagnostics", map[string]interface{}{
		"uri":         uri,
		"diagnostics": diagnostics,
	})
}

// 文件打开处理
func (h *ClangLSPHandler) handleDidOpen(ctx context.Context, req *jsonrpc2.Request) {
	log.Println("文件打开请求:", req.ID)
	var params struct {
		TextDocument struct {
			URI     string `json:"uri"`
			Text    string `json:"text"`
			Version int    `json:"version"`
		} `json:"textDocument"`
	}
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		log.Printf("解析错误: %v", err)
		return
	}

	// 创建临时文件用于Clang分析
	tmpPath := createTempFile(params.TextDocument.URI, params.TextDocument.Text)
	h.tempFiles[params.TextDocument.URI] = tmpPath

	// 触发初始诊断
	go func() {
		diagnostics := h.runClangDiagnostics(params.TextDocument.URI, tmpPath)
		h.publishDiagnostics(params.TextDocument.URI, diagnostics)
	}()
}

// 文件变更处理
func (h *ClangLSPHandler) handleDidChange(ctx context.Context, req *jsonrpc2.Request) {
	log.Println("文件变更请求:", req.ID)
	var params struct {
		TextDocument struct {
			URI     string `json:"uri"`
			Version int    `json:"version"`
		} `json:"textDocument"`
		ContentChanges []struct {
			Text string `json:"text"`
		} `json:"contentChanges"`
	}
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		log.Printf("解析错误: %v", err)
		return
	}

	if tmpPath, ok := h.tempFiles[params.TextDocument.URI]; ok {
		// 更新临时文件
		if len(params.ContentChanges) > 0 {
			os.WriteFile(tmpPath, []byte(params.ContentChanges[0].Text), 0644)
		}
		// 防抖处理（500ms）
		time.AfterFunc(500*time.Millisecond, func() {
			diagnostics := h.runClangDiagnostics(params.TextDocument.URI, tmpPath)
			h.publishDiagnostics(params.TextDocument.URI, diagnostics)
		})
	}
}
func (h *ClangLSPHandler) handleDiagnostic(ctx context.Context, req *jsonrpc2.Request) {
	log.Println("诊断请求:", req.ID)
	var params struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
	}
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		log.Printf("解析错误: %v", err)
		return
	}

	tmpPath := h.tempFiles[params.TextDocument.URI]
	diags := h.runClangDiagnostics(params.TextDocument.URI, tmpPath)
	if diags == nil {
		diags = []Diagnostic{}
	}
	response := map[string]interface{}{
		"kind":  "full",
		"items": diags,
	}
	h.conn.Reply(ctx, req.ID, response)
}

// 运行Clang诊断
func (h *ClangLSPHandler) runClangDiagnostics(uri, tmpPath string) []Diagnostic {
	log.Println("运行Clang诊断:", uri)
	// 获取编译参数
	args := []string{"-fsyntax-only", "-diagnostic-format=clang", tmpPath}
	config := h.getCompileConfig(filepath.Base(tmpPath))
	args = append(config.Args, args...)

	cmd := exec.Command("/opt/tops/bin/topscc", args...)
	log.Println("diagnostic cmd:", cmd.String())
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		log.Println("diagnostic cmd output:", err.Error())
		return []Diagnostic{}
	}
	if len(stderr.Bytes()) == 0 {
		log.Println("没有诊断信息")
		return []Diagnostic{}
	}
	var diagnostics []Diagnostic
	// parse stderr output every line
	lines := strings.Split(stderr.String(), "\n")
	for _, line := range lines {
		if strings.Contains(line, "error:") || strings.Contains(line, "warning:") {
			// 解析诊断信息
			var diag Diagnostic
			parts := strings.Split(line, ":")
			lineNum, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
			colNum, _ := strconv.Atoi(strings.TrimSpace(parts[2]))
			level := strings.TrimSpace(parts[3])
			msg := strings.TrimSpace(parts[4])
			if strings.Contains(level, "error") {
				diag.Severity = Error
			} else if strings.Contains(level, "warning") {
				diag.Severity = Warning
			}
			diag.Message = msg
			diag.Range.Start.Line = lineNum
			diag.Range.Start.Character = colNum
			diag.Range.End.Line = lineNum
			diag.Range.End.Character = colNum + 1
			diagnostics = append(diagnostics, diag)
		}
	}

	// 转换诊断信息位置
	for i := range diagnostics {
		adjustDiagnosticRange(&diagnostics[i])
	}

	return diagnostics
}

// 代码补全处理
func (h *ClangLSPHandler) handleCompletion(ctx context.Context, req *jsonrpc2.Request) {
	log.Println("代码补全请求:", req.ID)
	var params struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
		Position struct {
			Line      int `json:"line"`
			Character int `json:"character"`
		} `json:"position"`
	}
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		log.Printf("解析错误: %v", err)
		return
	}

	tmpPath := h.tempFiles[params.TextDocument.URI]
	line := params.Position.Line + 1
	col := params.Position.Character + 1

	// 调用Clang补全
	cmd := exec.Command("clang", "-Xclang", "-code-completion-at",
		fmt.Sprintf("%s:%d:%d", tmpPath, line, col), tmpPath)
	output, _ := cmd.CombinedOutput()

	items := parseCompletionOutput(string(output))
	h.conn.Reply(ctx, req.ID, items)
}

// 解析Clang补全输出
func parseCompletionOutput(output string) []CompletionItem {
	var items []CompletionItem
	re := regexp.MustCompile(`COMPLETION: (\w+)\s?: (.+)`)

	for _, line := range strings.Split(output, "\n") {
		if matches := re.FindStringSubmatch(line); len(matches) > 2 {
			items = append(items, CompletionItem{
				Label:  matches[1],
				Detail: matches[2],
				Kind:   6, // 6表示Method类型
			})
		}
	}
	return items
}

// 辅助函数：创建临时文件
func createTempFile(uri, content string) string {
	tmpDir := os.TempDir()
	filename := strings.TrimPrefix(uri, "file://")
	tmpPath := filepath.Join(tmpDir, filepath.Base(filename))
	os.WriteFile(tmpPath, []byte(content), 0644)
	return tmpPath
}

// 调整诊断范围到LSP位置格式
func adjustDiagnosticRange(d *Diagnostic) {
	// Clang的行号从1开始，LSP从0开始
	d.Range.Start.Line--
	d.Range.Start.Character--
	d.Range.End.Line--
	d.Range.End.Character--
}

// 获取编译配置（示例实现）
func (h *ClangLSPHandler) getCompileConfig(filename string) CompileConfig {
	// 实际项目中应从compile_commands.json加载
	return CompileConfig{
		Args: []string{"-std=c++17", "-O3", "-ltops", "-arch", "gcu300"},
	}
}
