package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"tops-lsp/lsp"

	"github.com/sourcegraph/jsonrpc2"
)

func main() {
	ln, err := net.Listen("tcp", "127.0.0.1:0") // 使用端口 0，让系统分配一个随机端口
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()

	fmt.Printf("TopsCC-based LSP running on %v\n", ln.Addr().String()) // 明确打印监听地址和端口
	conn, err := ln.Accept()
	if err != nil {
		log.Println("Accept error:", err)
		return
	}
	log.Println("Client connected:", conn.RemoteAddr())

	handler := lsp.NewClangLSPHandler()
	defer handler.CleanUp()

	// 捕获进程退出信号
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		log.Println("Received termination signal, cleaning up...")
		handler.CleanUp()
		os.Exit(0)
	}()

	jsonrpc2Conn := jsonrpc2.NewConn(context.Background(),
		jsonrpc2.NewBufferedStream(conn, jsonrpc2.VSCodeObjectCodec{}),
		handler)

	// 等待连接关闭
	<-jsonrpc2Conn.DisconnectNotify()
	log.Println("Client disconnected:", conn.RemoteAddr())
}
