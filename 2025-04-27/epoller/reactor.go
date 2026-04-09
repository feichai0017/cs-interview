package epoller

import (
	"fmt"
	"net"

	"golang.org/x/sys/unix"
)


func main() {
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		panic(err)
	}
	defer ln.Close()

	fmt.Println("Server is listening on :8080")

	epollFd, err := unix.EpollCreate1(0)
	if err != nil {
		panic(err)
	}

	newConns := make(chan net.Conn, 1024)

	pool := NewWorkerPool(100)

	go acceptLoop(ln, newConns)

	go registerLoop(newConns, epollFd)

	eventLoop(epollFd, pool)
}