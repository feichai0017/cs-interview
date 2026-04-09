package rpc

import (
	"errors"
	"net"
	"reflect"
)


type Client struct {
	conn net.Conn
}

func NewClient(connection net.Conn) *Client {
	return &Client{connection};
}

func (c *Client) Call(name string, fPtr any) {
	fnVal := reflect.ValueOf(fPtr).Elem()
	fnType := fnVal.Type();

	wrapper := func(in []reflect.Value) []reflect.Value {
		args := make([]any, len(in));
		for i, v := range in {
			args[i] = v.Interface();
		}

		req := RPCdata{Name: name, Args: args}
		rawReq, _ := Encode(req);
		err := WriteFrame(c.conn, rawReq)
		if err != nil {
			return errorResults(fnType, err)
		}
		rawResp, err := ReadFrame(c.conn)
		if err != nil {
			return errorResults(fnType, err)
		}
		resp, _ := Decode(rawResp)
		if resp.Err != "" {
			return errorResults(fnType, errors.New(resp.Err))
		}
		out := make([]reflect.Value, fnType.NumOut())
        for i := range fnType.NumOut() {
            if i < len(resp.Args) {
                out[i] = reflect.ValueOf(resp.Args[i])
            } else {
                out[i] = reflect.Zero(fnType.Out(i))
            }
        }
        return out
	}

	fnVal.Set(reflect.MakeFunc(fnType, wrapper))
}


// errorResults 统一构造出错时的返回值（最后一个为 error）
func errorResults(fnType reflect.Type, err error) []reflect.Value {
    out := make([]reflect.Value, fnType.NumOut())
    for i := range fnType.NumOut()-1 {
        out[i] = reflect.Zero(fnType.Out(i))
    }
    out[len(out)-1] = reflect.ValueOf(err)
    return out
}