package main

import (
	"encoding/json"
	"fmt"
	"log"
	"minirpc"
	"minirpc/codec"
	"net"
	"time"
)

func startServer(addr chan string) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal("network error: ", err)
	}
	log.Println("start tpc server on ", l.Addr())
	addr <- l.Addr().String()
	minirpc.Accept(l)
}

func main() {
	addr := make(chan string)
	// 使用信道 addr 确保服务端端口监听成功，客户端再请求
	go startServer(addr)

	conn, _ := net.Dial("tcp", <-addr)
	defer func() { _ = conn.Close() }()

	time.Sleep(time.Second)
	// 发送 Option 进行协议交换
	_ = json.NewEncoder(conn).Encode(minirpc.DefaultOption)
	cc := codec.NewGobCodec(conn)
	// send request & receive response
	for i := 0; i < 5; i++ {
		h := &codec.Header{
			ServiceMethod: "Foo.Sum",
			Seq:           uint64(i),
		}
		_ = cc.Write(h, fmt.Sprintf("mini req %d", h.Seq))
		_ = cc.ReadHeader(h)
		var reply string
		// 解析服务端响应
		_ = cc.ReadBody(&reply)
		log.Println("reply:", reply)
	}
}
