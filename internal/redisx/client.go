package redisx

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	Addr    string
	Timeout time.Duration
}

func New(addr string) *Client { return &Client{Addr: addr, Timeout: 5 * time.Second} }

func (c *Client) Do(ctx context.Context, args ...string) (any, error) {
	d := net.Dialer{Timeout: c.Timeout}
	conn, err := d.DialContext(ctx, "tcp", c.Addr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	if dl, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(dl)
	} else if len(args) >= 3 && strings.EqualFold(args[0], "BLPOP") && args[len(args)-1] == "0" {
		_ = conn.SetDeadline(time.Now().Add(24 * time.Hour))
	} else {
		_ = conn.SetDeadline(time.Now().Add(c.Timeout))
	}
	var b strings.Builder
	b.WriteString("*" + strconv.Itoa(len(args)) + "\r\n")
	for _, a := range args {
		b.WriteString("$" + strconv.Itoa(len([]byte(a))) + "\r\n")
		b.WriteString(a)
		b.WriteString("\r\n")
	}
	if _, err = conn.Write([]byte(b.String())); err != nil {
		return nil, err
	}
	return readReply(bufio.NewReader(conn))
}

func readReply(r *bufio.Reader) (any, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
	if line == "" {
		return nil, errors.New("redis: empty response")
	}
	switch line[0] {
	case '+':
		return line[1:], nil
	case '-':
		return nil, errors.New(line[1:])
	case ':':
		return strconv.ParseInt(line[1:], 10, 64)
	case '$':
		n, err := strconv.Atoi(line[1:])
		if err != nil {
			return nil, err
		}
		if n < 0 {
			return nil, nil
		}
		buf := make([]byte, n+2)
		if _, err = readFull(r, buf); err != nil {
			return nil, err
		}
		return string(buf[:n]), nil
	case '*':
		n, err := strconv.Atoi(line[1:])
		if err != nil {
			return nil, err
		}
		if n < 0 {
			return nil, nil
		}
		out := make([]any, n)
		for i := range out {
			out[i], err = readReply(r)
			if err != nil {
				return nil, err
			}
		}
		return out, nil
	default:
		return nil, fmt.Errorf("redis: unknown response %q", line)
	}
}
func readFull(r *bufio.Reader, b []byte) (int, error) {
	n := 0
	for n < len(b) {
		k, e := r.Read(b[n:])
		n += k
		if e != nil {
			return n, e
		}
	}
	return n, nil
}
