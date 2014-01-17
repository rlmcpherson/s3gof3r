package s3gof3r

import (
	"net"
	"net/http"
	"time"
)

type deadlineConn struct {
	Timeout time.Duration
	net.Conn
}

func (c *deadlineConn) Read(b []byte) (n int, err error) {
	if err = c.Conn.SetDeadline(time.Now().Add(c.Timeout)); err != nil {
		return
	}
	return c.Conn.Read(b)
}

func (c *deadlineConn) Write(b []byte) (n int, err error) {
	if err = c.Conn.SetDeadline(time.Now().Add(c.Timeout)); err != nil {
		return
	}
	return c.Conn.Write(b)
}

func ClientWithTimeout(timeout time.Duration) *http.Client {
	transport := &http.Transport{
		Dial: func(netw, addr string) (net.Conn, error) {
			c, err := net.DialTimeout(netw, addr, timeout)
			if err != nil {
				return nil, err
			}
			return &deadlineConn{timeout, c}, nil
		},
		ResponseHeaderTimeout: timeout,
	}
	return &http.Client{Transport: transport}
}
