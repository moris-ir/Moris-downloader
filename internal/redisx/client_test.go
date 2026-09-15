package redisx

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"testing"
	"time"
)

func TestClientRESP(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		c, _ := ln.Accept()
		defer c.Close()
		r := bufio.NewReader(c)
		line, _ := r.ReadString('\n')
		_ = line
		fmt.Fprint(c, "+PONG\r\n")
	}()
	c := New(ln.Addr().String())
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	v, err := c.Do(ctx, "PING")
	if err != nil {
		t.Fatal(err)
	}
	if v != "PONG" {
		t.Fatalf("%v", v)
	}
}
