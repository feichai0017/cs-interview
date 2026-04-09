package epoller

import (
	"fmt"
	"net"
	"os"
	"sync"
	"syscall"

	"golang.org/x/sys/unix"
)

var fdConnMap sync.Map

func acceptLoop(ln net.Listener, ch chan<- net.Conn) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		ch <- conn
	}
}

func registerLoop(epollFd int, ch <-chan net.Conn) {
	for conn := range ch {
		rawConn, _ := conn.(*net.TCPConn).SyscallConn()

		var fd int
		rawConn.Control(func(f uintptr) {
			fd = int(f)
		})

		ev := &unix.EpollEvent{
			Events: unix.EPOLLIN | unix.EPOLLET,
			Fd:     int32(fd),
		}

		err := unix.EpollCtl(epollFd, unix.EPOLL_CTL_ADD, fd, ev)
		if err != nil {
			fmt.Println("Error adding fd to epoll:", err)
			conn.Close()
		} else {
			fdConnMap.Store(fd, conn)
		}
		
	}
}

func eventLoop(epollFd int, pool *WorkerPool) {
	events := make([]unix.EpollEvent, 1024)

	for {
		n, err := unix.EpollWait(epollFd, events, -1)
		if err != nil {
			if err == syscall.EINTR {
				continue
			}

			fmt.Println("Error in EpollWait:", err)
			os.Exit(1)
		}

		for i := 0; i < n; i++ {
			fd := int(events[i].Fd)
			val, ok := fdConnMap.Load(fd)
			if !ok {
				continue
			}

			conn := val.(net.Conn)

			pool.Submit(func() {
				handleConn(epollFd, fd, conn)
			})
		}
	}
}


func handleConn(epollFd int, fd int, conn net.Conn) {
	buf := make([]byte, 4096)

	for {
		n, err := conn.Read(buf)
		if n == 0 || err != nil {
			fmt.Println("Connection closed:", conn.RemoteAddr())
			syscall.EpollCtl(epollFd, unix.EPOLL_CTL_DEL, fd, nil)
			conn.Close()
			fdConnMap.Delete(fd)
			return
		}

		conn.Write(buf[:n])
	}
}