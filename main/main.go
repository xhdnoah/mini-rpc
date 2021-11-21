package main

import (
	"fmt"
	"log"
	"minirpc"
	"net"
	"sync"
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
	log.SetFlags(0)
	addr := make(chan string)
	// 使用信道 addr 确保服务端端口监听成功，客户端再请求
	go startServer(addr)
	client, _ := minirpc.Dial("tcp", <-addr)
	defer func() { _ = client.Close() }()

	time.Sleep(time.Second)
	// send request & receive response
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			args := fmt.Sprintf("minirpc req %d", i)
			var reply string
			if err := client.Call("Foo.Sum", args, &reply); err != nil {
				log.Fatal("call Foo.Sum error: ", err)
			}
			log.Println("reply: ", reply)
		}(i)
	}
	wg.Wait()
}
