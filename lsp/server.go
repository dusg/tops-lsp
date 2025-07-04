package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"tops-lsp/lsp/data"

	"github.com/sourcegraph/jsonrpc2"
)

// 代码补全结构
type CompletionItem struct {
	Label  string `json:"label"`
	Kind   int    `json:"kind"`
	Detail string `json:"detail,omitempty"`
}
type ClangLSPHandler struct {
	conn            *jsonrpc2.Conn
	tempFiles       map[string]string
	diagnostics     map[string][]Diagnostic
	fileDidChange   map[string]bool
	diagMutex       sync.RWMutex
	didChangedTimer map[string]*time.Timer
	wgs             map[string]*sync.WaitGroup
	parsingAST      sync.Map
	FileContent     map[string]string
	workspace       string
}

func NewClangLSPHandler(workspace string) *ClangLSPHandler {
	return &ClangLSPHandler{
		tempFiles:       make(map[string]string),
		diagnostics:     make(map[string][]Diagnostic),
		fileDidChange:   make(map[string]bool),
		didChangedTimer: make(map[string]*time.Timer),
		wgs:             make(map[string]*sync.WaitGroup),
		FileContent:     make(map[string]string),
		workspace:       workspace,
	}
}

type LspContext interface {
	WorkSpace() string
	SetParserAST(uri string, parsing bool)
	publishDiagnostics(uri string, diagnostics []*data.Diagnostic)
}

func (h *ClangLSPHandler) SetParserAST(uri string, parsing bool) {
	h.parsingAST.Store(uri, parsing)
}

func (h *ClangLSPHandler) WorkSpace() string {
	return h.workspace
}

func (h *ClangLSPHandler) CacheDiagnostics(uri string, diagnostics []Diagnostic) {
	h.diagMutex.Lock()
	defer h.diagMutex.Unlock()
	h.diagnostics[uri] = diagnostics
}

func (h *ClangLSPHandler) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	h.conn = conn

	switch req.Method {
	case "initialize":
		h.handleInitialize(ctx, req)
	case "textDocument/didOpen":
		h.handleDidOpen(ctx, req)
	case "textDocument/didSave":
		h.handleDidSave(ctx, req)
	case "textDocument/didChange":
		h.handleDidChange(ctx, req)
	case "textDocument/completion":
		h.handleCompletion(ctx, req)
	case "textDocument/diagnostic":
		h.handleDiagnostic(ctx, req)
	case "textDocument/semanticTokens/full":
		h.handleSemanticTokensFull(ctx, req)
	case "textDocument/definition":
		h.handleDefinition(ctx, req)
	case "shutdown":
		h.handleShutdown(ctx, req)
	case "exit":
		h.handleExit(ctx, req)
	default:
		conn.ReplyWithError(ctx, req.ID,
			&jsonrpc2.Error{Code: jsonrpc2.CodeMethodNotFound, Message: "method not supported", Data: nil})
	}
}

func (h *ClangLSPHandler) CleanUp() {
	for _, tmpPath := range h.tempFiles {
		os.Remove(tmpPath)
	}
	h.tempFiles = make(map[string]string)
}

func (h *ClangLSPHandler) handleInitialize(ctx context.Context, req *jsonrpc2.Request) {
	ilog.Println("初始化请求:", req.ID)
	response := map[string]interface{}{
		"capabilities": map[string]interface{}{
			"textDocumentSync": 1,
			"completionProvider": map[string]interface{}{
				"triggerCharacters": []string{".", "->", "::"},
			},
			"definitionProvider": true,
			// "diagnosticProvider": map[string]interface{}{
			// 	"identifier":            "topscc",
			// 	"interFileDependencies": true,
			// 	"workspaceDiagnostics":  false,
			// },
			"semanticTokensProvider": semanticTokensOptions,
		},
	}
	h.conn.Reply(ctx, req.ID, response)
}

func (h *ClangLSPHandler) handleDidOpen(ctx context.Context, req *jsonrpc2.Request) {
	ilog.Println("文件打开请求:", req.ID)
	var params struct {
		TextDocument struct {
			URI     string `json:"uri"`
			Text    string `json:"text"`
			Version int    `json:"version"`
		} `json:"textDocument"`
	}
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		ilog.Printf("解析错误: %v", err)
		return
	}
	if _, ok := h.wgs[params.TextDocument.URI]; !ok {
		h.wgs[params.TextDocument.URI] = &sync.WaitGroup{}
	}
	h.parsingAST.Store(params.TextDocument.URI, true)
	h.FileContent[params.TextDocument.URI] = params.TextDocument.Text
	// 创建临时文件用于Clang分析
	// tmpPath, ok := h.tempFiles[params.TextDocument.URI]
	// if !ok {
	// 	tmpPath = createTempFile(params.TextDocument.URI, params.TextDocument.Text)
	// 	h.tempFiles[params.TextDocument.URI] = tmpPath
	// }

	// 触发初始诊断
	// h.wgs[params.TextDocument.URI].Add(1)
	h.fileDidChange[params.TextDocument.URI] = false

	h.diagnostics[params.TextDocument.URI] = nil
	go func() {
		// defer h.wgs[params.TextDocument.URI].Done()
		// diagnostics := RunClangDiagnostics(h, params.TextDocument.URI)
		// h.publishDiagnostics(params.TextDocument.URI, diagnostics)
		DataBaseInstance.BuildFileIndex(h, params.TextDocument.URI)
	}()

	// h.parsingAST[params.TextDocument.URI].Lock()
	// go func() {
	// 	defer h.parsingAST[params.TextDocument.URI].Unlock()
	// 	h.dataBase.RunIndexer(params.TextDocument.URI, tmpPath)
	// }()

}

func (h *ClangLSPHandler) handleDidChange(ctx context.Context, req *jsonrpc2.Request) {
	ilog.Println("文件修改请求:", req.ID)
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
		ilog.Printf("解析错误: %v", err)
		return
	}
	h.fileDidChange[params.TextDocument.URI] = true

	DataBaseInstance.semanticTokenCache.SetSemanticTokens(params.TextDocument.URI, nil)
}
func (h *ClangLSPHandler) handleDidSave(ctx context.Context, req *jsonrpc2.Request) {
	ilog.Println("文件Save请求:", req.ID)
	var params struct {
		TextDocument struct {
			URI     string `json:"uri"`
			Version int    `json:"version"`
		} `json:"textDocument"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		ilog.Printf("解析错误: %v", err)
		return
	}

	fileChanged := h.fileDidChange[params.TextDocument.URI]
	if !fileChanged {
		ilog.Println("文件未修改，跳过build索引:")
		return
	}

	h.fileDidChange[params.TextDocument.URI] = false
	h.diagnostics[params.TextDocument.URI] = nil
	h.parsingAST.Store(params.TextDocument.URI, true)
	go func() {
		DataBaseInstance.BuildFileIndex(h, params.TextDocument.URI)
		// diagnostics := RunClangDiagnostics(h, params.TextDocument.URI)
		// h.publishDiagnostics(params.TextDocument.URI, diagnostics)
	}()
	// if h.didChangedTimer[params.TextDocument.URI] != nil {
	// 	h.didChangedTimer[params.TextDocument.URI].Stop()
	// }

	// h.didChangedTimer[params.TextDocument.URI] = time.AfterFunc(500*time.Millisecond, func() {
	// 	// defer mu.Unlock()
	// 	diagnostics := RunClangDiagnostics(params.TextDocument.URI)
	// 	h.publishDiagnostics(params.TextDocument.URI, diagnostics)
	// 	// h.dataBase.RunIndexer(params.TextDocument.URI, "")
	// })
	// h.FileContent[params.TextDocument.URI] = params.ContentChanges[0].Text
	// if tmpPath, ok := h.tempFiles[params.TextDocument.URI]; ok {
	// 	// 更新临时文件
	// 	if len(params.ContentChanges) > 0 {
	// 		os.WriteFile(tmpPath, []byte(params.ContentChanges[0].Text), 0644)
	// 	}
	// 	// h.wgs[params.TextDocument.URI].Add(1)
	// 	mu := h.parsingAST[params.TextDocument.URI]
	// 	mu.Lock()
	// 	// 防抖处理（500ms）
	// 	if h.didChangedTimer[params.TextDocument.URI] != nil {
	// 		h.didChangedTimer[params.TextDocument.URI].Stop()
	// 	}

	// 	h.didChangedTimer[params.TextDocument.URI] = time.AfterFunc(500*time.Millisecond, func() {
	// 		defer mu.Unlock()
	// 		diagnostics := RunClangDiagnostics(params.TextDocument.URI)
	// 		h.publishDiagnostics(params.TextDocument.URI, diagnostics)
	// 		h.dataBase.RunIndexer(params.TextDocument.URI, tmpPath)
	// 	})
	// }
}

type DiagnosticServerCancellationData struct {
	RetriggerRequest bool `json:"retriggerRequest"`
}

func (h *ClangLSPHandler) handleDiagnostic(ctx context.Context, req *jsonrpc2.Request) {
	ilog.Println("诊断请求:", req.ID)
	var params struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
	}
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		ilog.Printf("解析错误: %v", err)
		return
	}

	// h.diagMutex.RLock()
	// defer h.diagMutex.RUnlock()

	// diags, ok := h.diagnostics[params.TextDocument.URI]
	parsing, ok := h.parsingAST.Load(params.TextDocument.URI)
	if !ok || parsing == true {
		data, _ := json.Marshal(DiagnosticServerCancellationData{RetriggerRequest: true})
		h.conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
			Code:    CodeServerCancelled,
			Message: "Is running daignostic try later",
			Data:    (*json.RawMessage)(&data),
		})
		ilog.Println("正在运行诊断，请稍后重试")
		return
	}
	diags := DataBaseInstance.GetDiagnostic(h, params.TextDocument.URI)

	// tmpPath := h.tempFiles[params.TextDocument.URI]
	// diags := h.runClangDiagnostics(params.TextDocument.URI, tmpPath)
	response := map[string]interface{}{
		"kind":  "full",
		"items": diags,
	}
	h.conn.Reply(ctx, req.ID, response)
}

func (h *ClangLSPHandler) handleCompletion(ctx context.Context, req *jsonrpc2.Request) {
	ilog.Println("代码补全请求:", req.ID)
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
		ilog.Printf("解析错误: %v", err)
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

const CodeServerCancelled = -32802

func parseParams[T any](req *jsonrpc2.Request) T {
	var params T
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		elog.Panicf("解析错误: %v", err)
	}
	return params
}
func (h *ClangLSPHandler) handleDefinition(ctx context.Context, req *jsonrpc2.Request) {
	ilog.Println("Definition请求:", req.ID)
	// TODO: 实现跳转到定义功能
	// var params TextDocumentPositionParams
	params := parseParams[TextDocumentPositionParams](req)

	loc := DataBaseInstance.FindDefinition(params.TextDocument.Uri, params.Position)
	// 这里只是一个占位实现，返回空响应
	h.conn.Reply(ctx, req.ID, loc)
}

func (h *ClangLSPHandler) handleSemanticTokensFull(ctx context.Context, req *jsonrpc2.Request) {
	ilog.Println("SemanticTokens请求:", req.ID)
	var params struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
	}
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		elog.Printf("解析错误: %v", err)
		return
	}

	// h.conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{Code: CodeServerCancelled, Message: "is parsing AST, 请求取消"})
	// return
	// tokens := ParseSemanticToken(h.FileContent[params.TextDocument.URI])
	tokens, ok := DataBaseInstance.semanticTokenCache.GetSemanticTokens(params.TextDocument.URI)
	if !ok {
		h.conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{Code: CodeServerCancelled, Message: "is parsing AST, 请求取消"})
		return
	}
	// mu := h.parsingAST[params.TextDocument.URI]
	// if !mu.TryLock() {
	// 	info.Println("SemanticTokens请求被取消:", req.ID)
	// 	h.conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{Code: CodeServerCancelled, Message: "is parsing AST, 请求取消"})
	// 	return
	// }
	// defer mu.Unlock()
	// tokens, ok := h.dataBase.semanticTokenCache.GetSemanticTokens(params.TextDocument.URI)
	// if !ok {
	// 	tokens = []uint32{}
	// }
	// tokens := h.generateSemanticTokens(params.TextDocument.URI)
	response := map[string]interface{}{
		"data": tokens,
	}
	h.conn.Reply(ctx, req.ID, response)
}

func (h *ClangLSPHandler) handleShutdown(ctx context.Context, req *jsonrpc2.Request) {
	ilog.Println("关闭请求:", req.ID)
	h.conn.Reply(ctx, req.ID, nil)
}

func (h *ClangLSPHandler) handleExit(_ context.Context, req *jsonrpc2.Request) {
	ilog.Println("退出请求:", req.ID)
	h.CleanUp()
	os.Exit(0)
}

func (h *ClangLSPHandler) generateSemanticTokens(tmpPath string) []uint32 {
	ilog.Println("生成语义标记:", tmpPath)
	// 示例实现，实际应调用Clang或其他工具生成语义标记
	// 返回的数组格式为 [line, deltaStart, length, tokenType, tokenModifiers]
	return []uint32{
		0, 0, 8, uint32(KeywordType), 0,
		6, 4, 2, uint32(VariableType), 3, // 示例: 第0行，第0列，长度5，类型1，无修饰符
		// 1, 0, 8, KeywordType, 0, // 示例: 第1行，第2列，长度4，类型2，修饰符1
		// 1, 0, 8, KeywordType, 0, // 示例: 第1行，第2列，长度4，类型2，修饰符1
		// 1, 0, 8, KeywordType, 0, // 示例: 第1行，第2列，长度4，类型2，修饰符1
	}
}

func toVsCodeDiagnostic(origin []*data.Diagnostic) []Diagnostic {
	if origin == nil {
		return nil
	}
	diagnostics := make([]Diagnostic, len(origin))
	for i, d := range origin {
		diagnostics[i] = Diagnostic{
			Range: Range{
				Start: Position{
					Line:      d.Range.Start.Line,
					Character: d.Range.Start.Character,
				},
				End: Position{
					Line:      d.Range.End.Line,
					Character: d.Range.End.Character,
				},
			},
			Message:  d.Message,
			Severity: DiagnosticSeverity(d.Severity),
			Source:   d.Source,
		}
	}
	return diagnostics
}

func (h *ClangLSPHandler) publishDiagnostics(uri string, diagnostics []*data.Diagnostic) {
	if diagnostics == nil {
		diagnostics = []*data.Diagnostic{}
	}

	ilog.Println("发布诊断信息:", uri, "\n\t诊断数量:", len(diagnostics), diagnostics)
	// h.CacheDiagnostics(uri, diagnostics)
	h.conn.Notify(context.Background(), "textDocument/publishDiagnostics", map[string]interface{}{
		"uri":         uri,
		"diagnostics": toVsCodeDiagnostic(diagnostics),
	})
}

// func (h *ClangLSPHandler) runClangDiagnostics(uri, tmpPath string) []Diagnostic {
// 	info.Println("运行Clang诊断:", uri)
// 	// 获取编译参数
// 	args := []string{"-fsyntax-only", "--cuda-device-only", "-diagnostic-format=clang", tmpPath}
// 	config := h.getCompileConfig(filepath.Base(tmpPath))
// 	args = append(config.Args, args...)

// 	cmd := exec.Command("/opt/tops/bin/topscc", args...)
// 	info.Println("diagnostic cmd:", cmd.String())
// 	var stderr bytes.Buffer
// 	cmd.Stderr = &stderr
// 	cmd.Run()
// 	if len(stderr.Bytes()) == 0 {
// 		info.Println("没有诊断信息")
// 		return []Diagnostic{}
// 	}
// 	return ParseDiagnostics(stderr.String())
// }

func parseCompletionOutput(output string) []CompletionItem {
	items := []CompletionItem{}
	// re := regexp.MustCompile(`COMPLETION: (\w+)\s?: (.+)`)

	// for _, line := range strings.Split(output, "\n") {
	// 	if matches := re.FindStringSubmatch(line); len(matches) > 2 {
	// 		items = append(items, CompletionItem{
	// 			Label:  matches[1],
	// 			Detail: matches[2],
	// 			Kind:   6, // 6表示Method类型
	// 		})
	// 	}
	// }
	return items
}

func createTempFile(uri, content string) string {
	tmpDir := os.TempDir()
	filename := strings.TrimPrefix(uri, "file://")
	baseName := filepath.Base(filename)
	tmpFile, err := os.CreateTemp(tmpDir, "*-"+baseName)
	if err != nil {
		panic(fmt.Sprintf("无法创建临时文件: %v", err))
	}
	defer tmpFile.Close()

	if _, err := tmpFile.Write([]byte(content)); err != nil {
		panic(fmt.Sprintf("无法写入临时文件: %v", err))
	}

	return tmpFile.Name()
}

func (h *ClangLSPHandler) getCompileConfig(filename string) CompileConfig {
	// 实际项目中应从compile_commands.json加载
	return CompileConfig{
		Args: []string{"-std=c++17", "-O3", "-ltops", "-arch", "gcu300"},
	}
}
