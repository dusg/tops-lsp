package lsp

import (
	"bytes"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	reParentheses    = regexp.MustCompile(`\([^\)]*\)`)
	reDoubleQuotes   = regexp.MustCompile(`"[^"]*"`)
	reSingleQuotes   = regexp.MustCompile(`'[^']*'`)
	reSquareBrackets = regexp.MustCompile(`\[[^\]]*\]`)
	reAngleBrackets  = regexp.MustCompile(`<[^>]*>`)
	reIdentifier     = regexp.MustCompile(`[a-zA-Z0-9_\.]+`)
)

type DiagnosticSeverity int32

const (
	Error       DiagnosticSeverity = 1
	Warning     DiagnosticSeverity = 2
	Information DiagnosticSeverity = 3
	Hint        DiagnosticSeverity = 4
)

type Diagnostic struct {
	Range    Range              `json:"range"`
	Severity DiagnosticSeverity `json:"severity"`
	Message  string             `json:"message"`
	Source   string             `json:"source"`
}

func ParseDiagnostics(diagText string, fileContent string) []Diagnostic {
	var diagnostics []Diagnostic = []Diagnostic{}
	// parse stderr output every line
	lines := strings.Split(diagText, "\n")
	fileLines := strings.Split(fileContent, "\n")
	for _, line := range lines {
		if strings.Contains(line, "error:") || strings.Contains(line, "warning:") {
			// 解析诊断信息
			var diag Diagnostic
			parts := strings.Split(line, ":")
			if len(parts) < 5 {
				elog.Println("诊断信息格式错误:", line)
				diag.Severity = Error
				diag.Message = line
				diag.Range.Start.Line = 1
				diag.Range.Start.Character = 1
				diag.Range.End.Line = 1
				diag.Range.End.Character = 1
				diagnostics = append(diagnostics, diag)
				continue
			}
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
			diag.Range.Start.Line = uint32(lineNum)
			diag.Range.Start.Character = uint32(colNum)

			// 使用封装的函数匹配内容
			lineContent := fileLines[lineNum-1]
			startIdx := colNum - 1
			matchedStr, ok := matchContent(lineContent, startIdx)
			if ok {
				diag.Range.End.Line = uint32(lineNum)
				diag.Range.End.Character = uint32(colNum + len(matchedStr))
			} else {
				diag.Range.End.Line = uint32(lineNum)
				diag.Range.End.Character = uint32(colNum + 4)
			}

			diagnostics = append(diagnostics, diag)
		}
	}

	// 转换诊断信息位置
	for i := range diagnostics {
		adjustDiagnosticRange(&diagnostics[i])
	}

	return diagnostics
}

// 封装匹配逻辑的函数
func matchContent(lineContent string, startIdx int) (string, bool) {
	var matches []int
	if startIdx >= 0 && startIdx < len(lineContent) {
		switch lineContent[startIdx] {
		case '(':
			matches = reParentheses.FindStringIndex(lineContent[startIdx:])
		case '"':
			matches = reDoubleQuotes.FindStringIndex(lineContent[startIdx:])
		case '\'':
			matches = reSingleQuotes.FindStringIndex(lineContent[startIdx:])
		case '[':
			matches = reSquareBrackets.FindStringIndex(lineContent[startIdx:])
		case '<':
			matches = reAngleBrackets.FindStringIndex(lineContent[startIdx:])
		default:
			matches = reIdentifier.FindStringIndex(lineContent[startIdx:])
		}
		if matches != nil {
			matchedStr := lineContent[startIdx : startIdx+matches[1]]
			return matchedStr, true
		}
	}
	return "", false
}

// 调整诊断范围到LSP位置格式
func adjustDiagnosticRange(d *Diagnostic) {
	// Clang的行号从1开始，LSP从0开始
	d.Range.Start.Line--
	d.Range.Start.Character--
	d.Range.End.Line--
	d.Range.End.Character--
}

type diagnosticWorker struct {
	canceled bool
	ctx      LspContext
}

func newDiagnosticWorker(ctx LspContext) *diagnosticWorker {
	return &diagnosticWorker{
		canceled: false,
		ctx:      ctx,
	}
}

func (w *diagnosticWorker) waitCmd(cmd *exec.Cmd) <-chan bool {
	// 等待命令完成
	done := make(chan bool, 1)
	go func() {
		cmd.Wait()
		done <- true
	}()
	return done
}

func (w *diagnosticWorker) testCancel() <-chan bool {
	// 等待取消
	done := make(chan bool, 1)
	go func() {
		for {
			if w.canceled {
				done <- true
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()
	return done
}
func (w *diagnosticWorker) Cancel() {
	// 取消诊断
	w.canceled = true
}

func (w *diagnosticWorker) asyncRun(uri string) <-chan []Diagnostic {
	ch := make(chan []Diagnostic)
	go func() {
		w.canceled = false
		// 运行诊断
		ilog.Println("运行Clang诊断:", uri)
		// 获取编译参数
		file := strings.TrimPrefix(uri, "file://")
		args := []string{"-fsyntax-only", "--cuda-device-only", "-diagnostic-format=clang"}
		config := GetCompileConfig(w.ctx, file)
		args = append(config.Args, args...)

		cmd := exec.Command(config.Compiler, args...)
		ilog.Println("diagnostic cmd:", cmd.String())
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		// 启动命令
		cmd.Start()

		content, _ := os.ReadFile(file)
		// 检查是否取消
		select {
		case <-w.testCancel():
			// 如果取消，杀死进程并返回
			cmd.Process.Kill()
			ilog.Println("诊断被取消")
			ch <- nil
			return
		case <-w.waitCmd(cmd):
			// 命令完成
		}

		if len(stderr.Bytes()) == 0 {
			ilog.Println("没有诊断信息")
			ch <- []Diagnostic{}
			return
		}

		ch <- ParseDiagnostics(stderr.String(), string(content))
	}()
	return ch
}

var workers = make(map[string]*diagnosticWorker)
var diagMutex = sync.Mutex{}

func RunClangDiagnostics(ctx LspContext, uri string) []Diagnostic {
	// info.Println("RunClangDiagnostics:", uri)
	diagMutex.Lock()
	worker, found := workers[uri]
	if found {
		worker.Cancel()
	}
	worker = newDiagnosticWorker(ctx)
	workers[uri] = worker
	// cancel last task
	feature := worker.asyncRun(uri)

	diagMutex.Unlock()
	return <-feature
}
