package rpc

import (
	"bytes"
	"encoding/gob"
)

type RPCdata struct {
	Name 	string
	Args	[]any
	Err 	string
}


func Encode(data RPCdata) ([]byte, error) {
	var buf bytes.Buffer;
	encoder := gob.NewEncoder(&buf);
	if err := encoder.Encode(data); err != nil {
		return nil, err;
	}
	return buf.Bytes(), nil;
}


func Decode(b []byte) (RPCdata, error) {
	buf := bytes.NewBuffer(b);
	decoder := gob.NewDecoder(buf);
	var data RPCdata;
	if err := decoder.Decode(&data); err != nil {
		return RPCdata{}, err
	}
	return data, nil
}