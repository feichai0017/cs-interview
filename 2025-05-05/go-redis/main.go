package redis

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/netpoll"
	cmap "github.com/orcaman/concurrent-map/v2"
)

var (
	store     cmap.ConcurrentMap[string, string]
	expireMap cmap.ConcurrentMap[string, time.Time]
	aofFile   *os.File
)

func init() {
	store = cmap.New[string]()
	expireMap = cmap.New[time.Time]()

	var err error
	aofFile, err = os.OpenFile("appendonly.aof", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("open AOF error: %v", err)
	}
	replayAOF()
	go expireCleaner()

}

func main() {
	listener, err := netpoll.CreateListener("tcp", ":6379")
	if err != nil {
		log.Fatalf("listen error: %v", err)
	}

	eventLoop, err := netpoll.NewEventLoop(
		onRequest,
		netpoll.WithOnConnect(onConnect),
		netpoll.WithOnDisconnect(onClose),
		netpoll.WithOnPrepare(onPrepare),
		netpoll.WithReadTimeout(10*time.Second),
	)
	if err != nil {
		log.Fatalf("create event loop error: %v", err)
	}
	log.Println("GO-Redis server listening on:6379")
	if err := eventLoop.Serve(listener); err != nil {
		log.Fatalf("serve error: %v", err)
	}

}

func onPrepare(conn netpoll.Connection) context.Context {
	return context.Background()
}

func onConnect(ctx context.Context, conn netpoll.Connection) context.Context {
	fmt.Printf("[CONNECT] %s\n", conn.RemoteAddr())
	conn.AddCloseCallback(func(c netpoll.Connection) error {
		fmt.Printf("[CLOSE] %s\n", c.RemoteAddr())
		return nil
	})
	return ctx
}

func onRequest(ctx context.Context, conn netpoll.Connection) error {
	reader, writer := conn.Reader(), conn.Writer()
	defer reader.Release()

	msg, err := reader.ReadString(reader.Len())
	if err != nil {
		fmt.Printf("read error: %v\n", err)
		return err
	}
	fmt.Printf("[RECV] %s", msg)

	args, err := parseRESP(msg)
	if err != nil {
		writer.WriteString("-ERR invalid protocol\r\n")
		writer.Flush()
		return nil
	}
	if len(args) == 0 {
		writer.WriteString("-ERR empty command\r\n")
		writer.Flush()
		return nil
	}

	switch cmd := strings.ToUpper(args[0]); cmd {
	case "PING":
		if len(args) == 2 {
			bulk := args[1]
			writer.WriteString(fmt.Sprintf("$%d\r\n%s\r\n", len(bulk), bulk))
		} else {
			writer.WriteString("+PONG\r\n")
		}
	case "ECHO":
		if len(args) != 2 {
			writer.WriteString("-ERR wrong number of args for 'ECHO'\r\n")
		} else {
			msg := args[1]
			writer.WriteString(fmt.Sprintf("$%d\r\n%s\r\n", len(msg), msg))
		}
	case "SET":
		if len(args) != 3 {
			writer.WriteString("-ERR wrong number of args for 'SET'\r\n")
		} else {
			key, val := args[1], args[2]
			store.Set(key, val)
			recordAOF(args)
			writer.WriteString("+OK\r\n")
		}
	case "GET":
		if len(args) != 2 {
			writer.WriteString("-ERR wrong number of args for 'GET'\r\n")
		} else {
			key := args[1]
			if t, ok := expireMap.Get(key); ok && time.Now().After(t) {
				store.Remove(key)
				expireMap.Remove(key)
				writer.WriteString("$-1\r\n")
			} else if val, ok := store.Get(key); !ok {
				writer.WriteString("$-1\r\n")
			} else {
				writer.WriteString(fmt.Sprintf("$%d\r\n%s\r\n", len(val), val))
			}
		}
	case "EXPIRE":
		if len(args) != 3 {
			writer.WriteString("-ERR wrong number of args for 'EXPIRE'\r\n")
		} else {
			key := args[1]
			seconds, err := strconv.Atoi(args[2])
			if err != nil {
				writer.WriteString("-ERR invalid expire time\r\n")
			} else {
				expireMap.Set(key, time.Now().Add(time.Duration(seconds)*time.Second))
				recordAOF(args)
				writer.WriteString("+OK\r\n")
			}
		}
	case "TTL":
		if len(args) != 2 {
			writer.WriteString("-ERR wrong number of args for 'TTL'\r\n")
		} else {
			key := args[1]
			if t, ok := expireMap.Get(key); !ok {
				writer.WriteString(":-1\r\n")
			} else {
				rem := int(time.Until(t).Seconds())
				if rem < 0 {
					rem = -2 // already
				}
				writer.WriteString(fmt.Sprintf(":%d\r\n", rem))
			}
		}

	default:
		writer.WriteString(fmt.Sprintf("-ERR unknown command '%s'\r\n", args[0]))
	}

	writer.Flush()
	return nil

}

func onClose(ctx context.Context, conn netpoll.Connection) {
	return
}

// 记录到 AOF
func recordAOF(args []string) {
	line := "" + fmt.Sprintf("*%d\r\n", len(args))
	for _, a := range args {
		line += fmt.Sprintf("$%d\r\n%s\r\n", len(a), a)
	}
	aofFile.WriteString(line)
	aofFile.Sync()
}

// 重放 AOF
func replayAOF() {
	scanner := bufio.NewScanner(aofFile)
	for scanner.Scan() {
		line := scanner.Text()
		// 直接 parseRESP 并执行 SET/EXPIRE 不输出
		args, err := parseRESP(line + "\r\n")
		if err != nil || len(args) < 1 {
			continue
		}
		switch strings.ToUpper(args[0]) {
		case "SET":
			store.Set(args[1], args[2])
		case "EXPIRE":
			sec, _ := strconv.Atoi(args[2])
			expireMap.Set(args[1], time.Now().Add(time.Duration(sec)*time.Second))
		}
	}
}

// 过期清理
func expireCleaner() {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		expireMap.IterCb(func(key string, t time.Time) {
			if now.After(t) {
				store.Remove(key)
				expireMap.Remove(key)
			}
		})
	}
}

func parseRESP(msg string) ([]string, error) {
	lines := strings.Split(msg, "\r\n")
	if len(lines) < 3 || !strings.HasPrefix(lines[0], "*") {
		return nil, fmt.Errorf("short or invalid message")
	}

	cnt, err := strconv.Atoi(lines[0][1:])
	if err != nil || cnt < 1 {
		return nil, fmt.Errorf("invalid count")
	}
	args := make([]string, 0, cnt)
	idx := 1
	for range cnt {
		if idx >= len(lines) || !strings.HasPrefix(lines[idx], "$") {
			return nil, fmt.Errorf("expected bulk string header")
		}
		// lengthLine := lines[idx]
		idx++
		if idx >= len(lines) {
			return nil, fmt.Errorf("missing bulk data")
		}
		args = append(args, lines[idx])
		idx++
	}
	return args, nil
}
