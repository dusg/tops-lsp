package lsp

import (
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"tops-lsp/lsp/data"

	"google.golang.org/protobuf/proto"
)

func (s *SemanticTokenCache) GetSemanticTokens(uri string) ([]uint32, bool) {
	tokens := s.fileSemanticToken[uri]
	if tokens != nil {
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
	if tokens == nil {
		delete(s.fileSemanticToken, uri)
		return
	}
	s.fileSemanticToken[uri] = tokens
}

type AstCache struct {
	ast_          *data.TranslationUnit
	idxFile_      string
	includeFiles_ []string
}

func newAstCache(file string, ast *data.TranslationUnit) *AstCache {
	incs := []string{}

	for _, finfo := range ast.IncludedHeaders {
		incs = append(incs, ast.StringTable.Entries[finfo.FileName.Index])
	}
	sort.Strings(incs)

	return &AstCache{
		ast_:          ast,
		idxFile_:      file,
		includeFiles_: incs,
	}
}

func (a *AstCache) IncludeFiles() []string {
	return a.includeFiles_
}
func (a *AstCache) FindIncludeFile(inc string) bool {
	i := sort.SearchStrings(a.IncludeFiles(), inc)
	if i < len(a.IncludeFiles()) && a.IncludeFiles()[i] == inc {
		return true
	}
	return false
}
func (a *AstCache) GetAst() *data.TranslationUnit {
	return a.ast_
}
func (a *AstCache) SourceFile() string {
	return a.ast_.GetFilePath()
}

type DataBase struct {
	semanticTokenCache *SemanticTokenCache
	astCache           map[string]*AstCache
	astCacheMutex      sync.Mutex
}

var DataBaseInstance *DataBase = NewDataBase()

func NewDataBase() *DataBase {
	return &DataBase{
		semanticTokenCache: NewSemanticTokenCache(),
		astCache:           make(map[string]*AstCache),
		astCacheMutex:      sync.Mutex{},
	}
}

func (d *DataBase) AddAstCache(file string, ast *AstCache) {
	d.astCacheMutex.Lock()
	defer d.astCacheMutex.Unlock()
	d.astCache[file] = ast
}

func (d *DataBase) GetAstCache(file string) *AstCache {
	d.astCacheMutex.Lock()
	defer d.astCacheMutex.Unlock()

	if !isHeaderFile(file) {
		ast, ok := d.astCache[file]
		if ok {
			return ast
		}
	}
	// 如果没有找到，尝试从头文件中获取
	for _, cache := range d.astCache {
		i := sort.SearchStrings(cache.IncludeFiles(), file)
		if i < len(cache.IncludeFiles()) && cache.IncludeFiles()[i] == file {
			return cache
		}
	}
	return nil
}

func isHeaderFile(file string) bool {
	return strings.HasSuffix(file, ".h")
}

func buildHeaderIndex(file string) {
	// TODO
}

func GetCacheDir(ctx LspContext) string {
	// 获取工作区目录
	workSpace := ctx.WorkSpace()
	// 拼接缓存目录路径
	cacheDir := workSpace + "/.tops-lsp"
	// 创建缓存目录
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		log.Fatalf("Failed to create cache directory: %v", err)
	}
	return cacheDir
}

type AstBuilder struct {
	worker *AsyncWorker
}

func newAstBuilder() *AstBuilder {
	return &AstBuilder{
		worker: nil,
	}
}

var astBuilders_ = make(map[string]*AstBuilder)
var astBuilderMutex_ = &sync.Mutex{}

func (builder *AstBuilder) buildAst(ctx LspContext, config *CompileConfig, output string) *AstCache {
	args := append(config.Args, []string{"-fsyntax-only", "--cuda-device-only", "-w"}...)
	tmpOut, _ := os.CreateTemp("", "tops-lsp-"+filepath.Base(output)+".*.idx")
	defer os.Remove(tmpOut.Name())
	// append plugin args
	args = append(args, []string{"-Xclang", "-load", "-Xclang", GetClangPluginPath(),
		"-Xclang", "-plugin", "-Xclang", "tops-lsp",
		"-Xclang", "-plugin-arg-tops-lsp", "-Xclang", tmpOut.Name()}...)

	cmd := exec.Command(config.Compiler, args...)
	ilog.Println("indexer cmd: ", cmd.String())
	cmd.Start()
	select {
	case <-builder.worker.waitCancel():
		cmd.Process.Kill()
		ilog.Println("Indexing 被取消")
		return nil
	case <-builder.worker.waitCmd(cmd):
		// 命令完成
	}

	// 从临时文件读取 TranslationUnit
	dataBytes, err := os.ReadFile(tmpOut.Name())
	if err != nil {
		elog.Printf("Failed to read temp file: %v", err)
		return nil
	}

	if builder.worker.IsCanceled() {
		ilog.Println("Indexing 被取消")
		return nil
	}

	ast := data.TranslationUnit{}
	if err := proto.Unmarshal(dataBytes, &ast); err != nil {
		elog.Printf("Failed to unmarshal TranslationUnit: %v", err)
		return nil
	}
	if builder.worker.IsCanceled() {
		ilog.Println("Indexing 被取消")
		return nil
	}
	cache := newAstCache(output, &ast)
	DataBaseInstance.AddAstCache(config.File, cache)
	ilog.Println("save to index file: ", output)
	err = os.WriteFile(output, dataBytes, 0644)
	if err != nil {
		elog.Printf("Failed to save index file: %v", err)
		return nil
	}
	return cache
}

func (d *DataBase) BuildFileIndex(ctx LspContext, uri string) {
	ilog.Printf("run indexer %s\n", uri)
	file := strings.TrimPrefix(uri, "file://")
	if isHeaderFile(file) {
		buildHeaderIndex(file)
		return
	}
	config := GetCompileConfig(ctx, file)
	cache_dir := GetCacheDir(ctx)
	MakeDirIfNotExist(cache_dir)
	idxFile := GetIndexFileName(config)
	idxFile = path.Join(cache_dir, idxFile)
	astBuilderMutex_.Lock()

	builder, found := astBuilders_[uri]
	if found {
		builder.worker.Cancel()
	}
	builder = newAstBuilder()
	astBuilders_[uri] = builder

	builder.worker = AsyncRun(func(_ *AsyncWorker) {
		d.semanticTokenCache.SetSemanticTokens(uri, nil)
		ast := builder.buildAst(ctx, config, idxFile)
		if ast == nil {
			return
		}
		// 处理语义标记
		st := ParseSemanticTokenFromAst(ast)
		if builder.worker.IsCanceled() {
			return
		}
		d.semanticTokenCache.SetSemanticTokens(uri, st)
	})
	// cancel last task

	astBuilderMutex_.Unlock()
	defer ilog.Println("run indexer done ", uri)
}
