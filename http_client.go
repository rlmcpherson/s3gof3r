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

// ClientWithTimeout is an http client optimized for high throughput
// to S3, It times out more agressively than the default
// http client in net/http as well as setting deadlines on the TCP connection
func ClientWithTimeout(timeout time.Duration) *http.Client {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: func(netw, addr string) (net.Conn, error) {
			c, err := net.DialTimeout(netw, addr, timeout)
			if err != nil {
				return nil, err
			}
			if tc, ok := c.(*net.TCPConn); ok {
				tc.SetKeepAlive(true)
				tc.SetKeepAlivePeriod(timeout)
			}
			return &deadlineConn{timeout, c}, nil
		},
		ResponseHeaderTimeout: timeout,
		MaxIdleConnsPerHost:   10,
	}
	return &http.Client{Transport: transport}
}
