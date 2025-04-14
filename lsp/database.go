package lsp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os/exec"
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

func (d *DataBase) RunIndexer(uri, tmpPath string) {
	info.Printf("run indexer %s\n", uri)
	defer info.Println("run indexer done ", uri)
	// args := []string{"-fsyntax-only", "-Xclang", "-ast-dump=json", "-arch", "gcu300", "-ltops", "--cuda-device-only", tmpPath}
	args := []string{"-fsyntax-only", "-Xclang", "-ast-dump=json", "-I/home/carl.du/work/tops-lsp/test-files", tmpPath}

	// cmd := exec.Command("/opt/tops/bin/topscc", args...)
	cmd := exec.Command("/opt/tops/bin/clang++", args...)
	info.Println("indexer cmd: ", cmd.String())
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		error.Fatalf("Failed to get stdout pipe: %v", err)
		return
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	info.Println("cmd: ", cmd.String())
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

	semaTokenVisitor := NewSemanticTokenVisitor()
	ParseClangAstJosnt(astNode, []AstVisitor{semaTokenVisitor})
	d.semanticTokenCache.SetSemanticTokens(uri, semaTokenVisitor.FileSemanticToken)
}
