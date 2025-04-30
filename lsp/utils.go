package lsp

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// GetExecutablePath 返回当前可执行文件的路径
func GetExecutablePath() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Abs(exePath)
}

func GetClangPluginPath() string {
	exePath, err := GetExecutablePath()
	if err != nil {
		elog.Panicln("GetExecutablePath failed: ", err)
	}
	// 获取当前可执行文件的目录
	dir := filepath.Dir(exePath)
	// 拼接插件路径
	pluginPath := filepath.Join(dir, "libtops-lsp.so")
	return pluginPath
}

func GetIndexFileName(config *CompileConfig) string {
	// 获取原文件名
	originalFileName := filepath.Base(config.File)

	// 计算编译参数的哈希值
	cmd := []string{config.Compiler}
	cmd = append(cmd, config.Args...)
	hash := sha256.Sum256([]byte(strings.Join(cmd, " ")))
	// 构造索引文件名
	indexFileName := originalFileName + "." + fmt.Sprintf("%x", hash[:8]) + ".idx"

	// 返回完整路径
	return indexFileName
}

func MakeDirIfNotExist(path string) error {
	// 检查目录是否存在
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// 如果目录不存在，则创建目录
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}
	return nil
}

type AsyncWorker struct {
	canceled bool
}

func newAsyncWorker() *AsyncWorker {
	return &AsyncWorker{
		canceled: false,
	}
}

func (w *AsyncWorker) waitCmd(cmd *exec.Cmd) <-chan bool {
	// 等待命令完成
	done := make(chan bool, 1)
	go func() {
		cmd.Wait()
		done <- true
	}()
	return done
}

func (w *AsyncWorker) waitCancel() <-chan bool {
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
func (w *AsyncWorker) Cancel() {
	// 取消诊断
	w.canceled = true
}

func (w *AsyncWorker) IsCanceled() bool {
	return w.canceled
}

func AsyncRun(runner func(ctx *AsyncWorker)) *AsyncWorker {
	worker := newAsyncWorker()
	go func() {
		runner(worker)
	}()
	return worker
}
