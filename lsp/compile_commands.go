package lsp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

type CompileConfig struct {
	Compiler  string
	Args      []string
	File      string
	Directory string
}

type compileCommand struct {
	Directory string   `json:"directory"`
	File      string   `json:"file"`
	Command   string   `json:"command"`
	Arguments []string `json:"arguments"`
}

type ProjectConfig struct {
	CompileCommands map[string]CompileConfig
	mu              sync.RWMutex
}

var projectConfig = &ProjectConfig{
	CompileCommands: make(map[string]CompileConfig),
}

func parseCommand(command string) []string {
	// 使用正则表达式解析带引号的参数和普通参数
	re := regexp.MustCompile(`"([^"]*)"|(\S+="[^"]*")|([^\s=]+)`)
	matches := re.FindAllStringSubmatch(command, -1)

	var args []string
	for _, match := range matches {
		if match[1] != "" {
			args = append(args, match[1]) // 引号中的内容
		} else if match[2] != "" {
			args = append(args, match[2]) // 普通参数
		} else if match[3] != "" {
			args = append(args, match[3]) // 带等号的参数
		}
	}
	return args
}

func isTopsCC(cmd *CompileConfig) bool {
	isTopsCC := strings.HasSuffix(cmd.Compiler, "tops/bin/topscc")
	return isTopsCC
}
func isTopsClang(cmd *CompileConfig) bool {
	isTopsClang := strings.HasSuffix(cmd.Compiler, "tops/bin/clang++") ||
		strings.HasSuffix(cmd.Compiler, "tops/bin/clang")
	return isTopsClang
}

func isTopsFile(file string) bool {
	return strings.HasSuffix(file, ".tops")
}

func adjustCommand(cmd *CompileConfig) *CompileConfig {
	args := []string{}
	for _, arg := range args {
		if arg == "agcu300" || arg == "agcu400" || arg == "agcu200" {
			arg = strings.Replace(arg, "agcu", "gcu", 1)
		}

		if strings.Contains(arg, "agcu200+gcu210") ||
			strings.Contains(arg, "agcu300+gcu400") ||
			strings.Contains(arg, "agcu400+gcu500") {
			continue
		}
		args = append(args, arg)
	}
	cmd.Args = args
	return cmd
}

func loadCompileCommands(path string) (map[string]CompileConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var rawCommands []compileCommand
	if err := json.NewDecoder(file).Decode(&rawCommands); err != nil {
		return nil, err
	}

	commands := make(map[string]CompileConfig)
	for _, raw := range rawCommands {
		args := raw.Arguments
		if len(args) == 0 && len(raw.Command) > 0 {
			args = parseCommand(raw.Command) // 使用解析函数处理 Command
		}
		cmd := &CompileConfig{
			Compiler:  args[0],
			Args:      args[1:],
			File:      raw.File,
			Directory: raw.Directory,
		}
		commands[raw.File] = *adjustCommand(cmd)

	}
	return commands, nil
}

func FallbackCompileConfig(ctx LspContext, filename string) *CompileConfig {
	return &CompileConfig{
		Args:      []string{"-std=c++17", "-O3", "-ltops", "-arch", "gcu300"},
		Compiler:  "/opt/tops/bin/topscc",
		File:      filename,
		Directory: ctx.WorkSpace(),
	}
}

func GetCompileConfig(ctx LspContext, filename string) *CompileConfig {
	projectConfig.mu.RLock()
	if config, exists := projectConfig.CompileCommands[filename]; exists {
		projectConfig.mu.RUnlock()
		return &config
	}
	projectConfig.mu.RUnlock()

	// 尝试从文件所在目录加载 compile_commands.json
	dir := filepath.Dir(filename)
	compileCommandsPath := filepath.Join(dir, "compile_commands.json")
	commands, err := loadCompileCommands(compileCommandsPath)
	if err != nil {
		// 如果文件所在目录没有找到，则尝试从 workspace 目录加载
		workspacePath := filepath.Join(ctx.WorkSpace(), "compile_commands.json")
		commands, err = loadCompileCommands(workspacePath)
		if err != nil {
			// 如果都没有找到，返回默认配置
			return FallbackCompileConfig(ctx, filename)
		}
	}

	projectConfig.mu.Lock()
	for file, config := range commands {
		projectConfig.CompileCommands[file] = config
	}
	projectConfig.mu.Unlock()

	if config, exists := commands[filename]; exists {
		return &config
	}
	panic("unreachable code reached")
}
