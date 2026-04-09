package rpc

import (
	"encoding/gob"
	"fmt"
	"net"
	"time"
)

type User struct {
    Name string
    Age  int
}

// 远程方法签名：返回 User 和 error
func QueryUser(id int) (User, error) {
    db := map[int]User{
        1: {"Alice", 30},
        2: {"Bob", 25},
    }
    if u, ok := db[id]; ok {
        return u, nil
    }
    return User{}, fmt.Errorf("no user with id %d", id)
}

func main() {
    // 由于 gob 序列化 interface{}，需要注册具体类型
    gob.Register(User{})

    addr := "localhost:3212"
    // 1. 启动服务器
    srv := NewServer(addr)
    srv.Register("QueryUser", QueryUser)
    go srv.Run()

    // 等待服务器就绪
    time.Sleep(500 * time.Millisecond)

    // 2. 启动客户端
    conn, _ := net.Dial("tcp", addr)
    cli := NewClient(conn);

    // 3. 准备本地 stub
    var Query func(int) (User, error)
    cli.Call("QueryUser", &Query)

    // 4. 发起 RPC 调用
    u, err := Query(1)
    fmt.Println(u, err)
    u2, err := Query(3)
    fmt.Println(u2, err)
}