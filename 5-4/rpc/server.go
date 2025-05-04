package rpc

import (
	"bufio"
	"net"
	"strings"
	"sync"
)

var (
	svcMu 	 sync.RWMutex
	svcMap = make(map[string]any)
)

// Register 将一个服务实例（struct 指针）按照名称注册
func Register(name string, svc any) {
	svcMu.Lock();
	defer svcMu.Unlock();
	svcMap[name] = svc;
}

// Serve 接收一个以监听的 Listener 并处理所有连接
func Serve(l net.Listener) error {
	for {
		conn, err := l.Accept();
		if err != nil {
			return err;
		}
		go handleConn(conn);
	}
}

func handleConn(conn net.Conn) {
	defer conn.Close();
	for {
		frame, err := readFrame(conn);
		if err != nil {
			return;
		}
		go dispatch(conn, frame);
	}
}


func dispatch(conn net.Conn, frame *Frame)