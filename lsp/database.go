package lsp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"
	"sync"
)

func (s *SemanticTokenCache) GetSemanticTokens(uri string) ([]uint32, bool) {
	tokens, ok := s.fileSemanticToken[uri]
	if ok {
		return tokens.tokens, true
	}
	for _, file := range s.fileSemanticToken {
		if file.includeFiles[uri] != nil {
			return file.includeFiles[uri], true
		}
	}
	return []uint32{}, false
}
func (s *SemanticTokenCache) SetSemanticTokens(uri string, tokens *FileSemanticToken) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.fileSemanticToken[uri] = tokens
}

type DataBase struct {
	semanticTokenCache *SemanticTokenCache
}

func NewDataBase() *DataBase {
	return &DataBase{
		semanticTokenCache: NewSemanticTokenCache()}
}

func isHeaderFile(file string) bool {
	return strings.HasSuffix(file, ".h")
}

func buildHeaderIndex(file string) {
	// TODO
}

func (d *DataBase) buildAst(config *CompileConfig) {
	args := append(config.Args, []string{"-fsyntax-only", "--cuda-device-only"}...)

	// cmd := exec.Command("/opt/tops/bin/topscc", args...)
	cmd := exec.Command(config.Compiler, args...)
	ilog.Println("indexer cmd: ", cmd.String())
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		elog.Fatalf("Failed to get stdout pipe: %v", err)
		return
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	ilog.Println("cmd: ", cmd.String())
	if err := cmd.Start(); err != nil {
		fmt.Println("Error:", err)
		fmt.Println("Stderr:", stderr.String())
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

	cmd.Wait()

	log.Println("run cmd success")
	wg.Wait()

	// semaTokenVisitor := NewSemanticTokenVisitor()
	// ParseClangAstJosnt(astNode, []AstVisitor{semaTokenVisitor})
	// d.semanticTokenCache.SetSemanticTokens(uri, semaTokenVisitor.FileSemanticToken)
}
func (d *DataBase) BuildFileIndex(ctx LspContext, uri string) {
	ilog.Printf("run indexer %s\n", uri)
	file := strings.TrimPrefix(uri, "file://")
	if isHeaderFile(file) {
		buildHeaderIndex(file)
		return
	}
	config := GetCompileConfig(ctx, file)
	defer ilog.Println("run indexer done ", uri)
	d.buildAst(config)
}
