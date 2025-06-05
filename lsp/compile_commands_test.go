package lsp

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseCommand(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{
			input:    `gcc --tops-device-lib=libacoreop_gcu400.bc file.c`,
			expected: []string{"gcc", `--tops-device-lib=libacoreop_gcu400.bc`, "file.c"},
		},
		{
			input:    `gcc -o output file.c`,
			expected: []string{"gcc", "-o", "output", "file.c"},
		},
		{
			input:    `gcc --flag="value with spaces" file.c`,
			expected: []string{"gcc", `--flag="value with spaces"`, "file.c"},
		},
		{
			input:    `gcc -DNAME="value" file.c`,
			expected: []string{"gcc", `-DNAME="value"`, "file.c"},
		},

		{
			input:    `gcc -I"path/to/include" file.c`,
			expected: []string{"gcc", `-I"path/to/include"`, "file.c"},
		},
		{
			input:    `gcc -o "output file" "source file.c"`,
			expected: []string{"gcc", "-o", "output file", "source file.c"},
		},
	}

	for _, test := range tests {
		result := parseCommand(test.input)
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("parseCommand(%q) = %v; want %v", test.input, result, test.expected)
		}
	}
}

func TestLoadCompileCommands(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(GetTestDir(), "compile_commands.json")
	commands, err := loadCompileCommands(testFile)
	if err != nil {
		t.Fatalf("loadCompileCommands error: %v", err)
	}
	cfg, ok := commands["/tmp/project/test.cpp"]
	if !ok {
		t.Fatalf("expected file not found in commands")
	}
	if cfg.Compiler != "/opt/tops/bin/topscc" {
		t.Errorf("unexpected compiler: %s", cfg.Compiler)
	}
	if len(cfg.Args) == 0 || cfg.Args[len(cfg.Args)-1] != "/tmp/project/test.cpp" {
		t.Errorf("unexpected args: %v", cfg.Args)
	}
	if cfg.Directory != "/tmp/project" {
		t.Errorf("unexpected directory: %s", cfg.Directory)
	}

	// 测试文件不存在时的错误处理
	_, err = loadCompileCommands(filepath.Join(tmpDir, "not_exist.json"))
	if err == nil {
		t.Errorf("expected error for non-existent file")
	}
}
