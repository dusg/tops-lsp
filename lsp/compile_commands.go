package lsp

import "sync"

type CompileConfig struct {
	Args    []string `json:"args"`
	Include []string `json:"include"`
	Defines []string `json:"defines"`
}

type ProjectConfig struct {
	CompileCommands map[string]CompileConfig
	mu              sync.RWMutex
}

var projectConfig = &ProjectConfig{
	CompileCommands: make(map[string]CompileConfig),
}

func GetCompileConfig(filename string) CompileConfig {
	// 实际项目中应从compile_commands.json加载
	return CompileConfig{
		Args: []string{"-std=c++17", "-O3", "-ltops", "-arch", "gcu300"},
	}
}
