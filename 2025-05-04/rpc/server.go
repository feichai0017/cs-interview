package rpc

import (
	"fmt"
	"log"
	"net"
	"reflect"
)


type RPCServer struct {
	addr 	string
	funcs 	map[string]reflect.Value
}



func NewServer(addr string) *RPCServer {
	return &RPCServer{
		addr: 	addr,
		funcs: 	make(map[string]reflect.Value),
	}
}

func (s *RPCServer) Register(name string, fn any) {
	s.funcs[name] = reflect.ValueOf(fn);
}

func (s *RPCServer) Run() error {
	listener, err:= net.Listen("tcp", s.addr);
	if err != nil {
		return err
	}

	log.Printf("RPC Server listening on %s\n", s.addr);
	for {
		conn, err := listener.Accept();
		if err != nil {
			log.Println("accept error: ", err);
			continue;
		}
		go s.handleConn(conn);
	}
}

func (s *RPCServer) handleConn(conn net.Conn) {
	defer conn.Close();
	for {
		rawReq, err := ReadFrame(conn);
		if err != nil {
			log.Println("read frame error: ", err);
			return;
		}
		req, err := Decode(rawReq)
		if err != nil {
			log.Println("decode error", err);
			continue;
		}

		resp := s.execute(req);

		rawResp, _ := Encode(resp);
		if err := WriteFrame(conn, rawResp); err != nil {
			log.Println("send frame error:", err);
			return;
		}
		
	}
}

func (s *RPCServer) execute(req RPCdata) RPCdata {
	f, ok := s.funcs[req.Name];
	if !ok {
		errMsg := fmt.Sprintf("method %s not registered", req.Name);
		return RPCdata{
			Name: req.Name,
			Args: nil,
			Err: errMsg,
		}
	}

	in := make([]reflect.Value, len(req.Args))
	for i, arg := range req.Args {
		in[i] = reflect.ValueOf(arg)
	}

	out := f.Call(in);

	resArgs := make([]any, len(out)-1);
	for i := range len(out) - 1 {
		resArgs[i] = out[i];
	}
	var errStr string;
	if e, ok := out[len(out)-1].Interface().(error); ok && e != nil {
		errStr = e.Error()
	}
	return RPCdata{Name: req.Name, Args: resArgs, Err: errStr}
}