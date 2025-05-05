package rpc

import (
	"bufio"
	"encoding/binary"
	"io"
	"net"
)


const HEADER_SIZE = 4

func WriteFrame(conn net.Conn, payload []byte) error {
	w := bufio.NewWriter(conn);
	// write length of payload first
	if err := binary.Write(w, binary.BigEndian, uint32(len(payload))); err != nil {
		return err;
	}
	
	_, err := w.Write(payload)
	if  err != nil {
		return err;
	}

	return w.Flush();
	
}


func ReadFrame(conn net.Conn) ([]byte, error) {
	r := bufio.NewReader(conn);

	var length uint32;
	if err := binary.Read(r, binary.BigEndian, &length); err != nil {
		return nil, err
	}

	payload := make([]byte, length);
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, err;
	}
	return payload, nil;
	
}