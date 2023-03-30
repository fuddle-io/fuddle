// Copyright (C) 2023 Andrew Dunstall
//
// Fuddle is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Fuddle is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package cluster

import (
	"fmt"
	"io"
	"net"
	"sync"

	"go.uber.org/atomic"
)

type Proxy struct {
	conns map[*proxyConn]interface{}

	// mu is a mutex protecting the fields above.
	mu sync.Mutex

	target string
	ln     net.Listener
}

func NewProxy(target string) (*Proxy, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("proxy: listen: %w", err)
	}

	proxy := &Proxy{
		conns:  make(map[*proxyConn]interface{}),
		target: target,
		ln:     ln,
	}
	go proxy.acceptLoop()
	return proxy, nil
}

func (p *Proxy) NumConns() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	return len(p.conns)
}

func (p *Proxy) BlockActiveConns() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for c := range p.conns {
		c.SetBlock(true)
	}
}

// Drop drops all existing connections.
func (p *Proxy) Drop() {
	p.mu.Lock()
	for c := range p.conns {
		c.Close()
	}
	p.conns = make(map[*proxyConn]interface{})
	p.mu.Unlock()
}

func (p *Proxy) Addr() string {
	return p.ln.Addr().String()
}

func (p *Proxy) CloseIfActive() bool {
	p.mu.Lock()
	isActive := len(p.conns) > 0
	p.mu.Unlock()

	if isActive {
		p.Close()
	}

	return isActive
}

func (p *Proxy) Close() {
	for c := range p.conns {
		c.Close()
	}
	p.ln.Close()
}

func (p *Proxy) acceptLoop() {
	for {
		downstream, err := p.ln.Accept()
		if err != nil {
			return
		}

		upstream, err := net.Dial("tcp", p.target)
		if err != nil {
			return
		}

		conn := newProxyConn(downstream, upstream)
		p.mu.Lock()
		p.conns[conn] = struct{}{}
		p.mu.Unlock()
	}
}

type proxyConn struct {
	upstream   net.Conn
	downstream net.Conn
	blocked    *atomic.Bool
}

func newProxyConn(upstream net.Conn, downstream net.Conn) *proxyConn {
	conn := &proxyConn{
		upstream:   upstream,
		downstream: downstream,
		blocked:    atomic.NewBool(false),
	}
	go conn.forward(downstream, upstream)
	go conn.forward(upstream, downstream)
	return conn
}

func (c *proxyConn) SetBlock(blocked bool) {
	c.blocked.Store(blocked)
}

func (c *proxyConn) Close() {
	c.upstream.Close()
	c.downstream.Close()
}

func (c *proxyConn) forward(dst io.Writer, src io.Reader) {
	buf := make([]byte, 1024)
	for {
		n, err := src.Read(buf)
		if err != nil {
			return
		}

		if c.blocked.Load() {
			continue
		}

		_, err = dst.Write(buf[0:n])
		if err != nil {
			return
		}
	}
}
