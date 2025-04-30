package lsp

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
)

func TestGetIndexFileName(t *testing.T) {
	// 模拟 CompileConfig
	mockConfig := &CompileConfig{
		File:     "test.c",
		Compiler: "gcc",
		Args:     []string{"-O2", "-Wall"},
	}

	// 计算预期的哈希值
	cmd := []string{mockConfig.Compiler}
	cmd = append(cmd, mockConfig.Args...)
	hash := sha256.Sum256([]byte(strings.Join(cmd, " ")))
	expectedHash := hash[:8]
	// 构造预期的索引文件路径
	expectedFileName := fmt.Sprintf("test.c.%x.idx", expectedHash)

	// 调用被测试函数
	result := GetIndexFileName(mockConfig)

	// 验证结果
	if result != expectedFileName {
		t.Errorf("Expected %s, but got %s", expectedFileName, result)
	}

}

// MockLspContext 模拟 LspContext
type MockLspContext struct {
	workspaceDir string
}

func (m *MockLspContext) WorkSpace() string {
	return m.workspaceDir
}
