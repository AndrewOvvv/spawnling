package rcon

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

const (
	dialTimeout = 5 * time.Second
	rwTimeout   = 10 * time.Second
)

var (
	ErrAuthFailed   = errors.New("rcon: authentication failed")
	ErrNotConnected = errors.New("rcon: not connected")
)

type Client struct {
	mu       sync.Mutex
	conn     net.Conn
	dialer   Dialer
	addr     string
	password string
	seq      atomic.Int32
}

func NewRCONClient(addr, password string, dialer Dialer) *Client {
	return &Client{addr: addr, password: password, dialer: dialer}
}

// Connect dials the server and authenticates. Must be called before Exec.
func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	conn, err := c.dialer.Dial(c.addr, dialTimeout)
	if err != nil {
		return fmt.Errorf("rcon: dial %s: %w", c.addr, err)
	}
	c.conn = conn
	return c.authenticate()
}

func (c *Client) authenticate() error {
	id := c.nextID()
	if err := writePacket(c.conn, packet{id: id, kind: packetTypeAuth, payload: c.password}); err != nil {
		return fmt.Errorf("rcon: send auth: %w", err)
	}
	resp, err := readPacket(c.conn)
	if err != nil {
		return fmt.Errorf("rcon: read auth response: %w", err)
	}
	if resp.id == -1 {
		return ErrAuthFailed
	}
	return nil
}

// Exec sends a command to the server and returns its response.
func (c *Client) Exec(cmd string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return "", ErrNotConnected
	}

	id := c.nextID()
	c.conn.SetDeadline(time.Now().Add(rwTimeout))
	defer c.conn.SetDeadline(time.Time{})

	if err := writePacket(c.conn, packet{id: id, kind: packetTypeCommand, payload: cmd}); err != nil {
		return "", fmt.Errorf("rcon: send command: %w", err)
	}
	resp, err := readPacket(c.conn)
	if err != nil {
		return "", fmt.Errorf("rcon: read response: %w", err)
	}
	if resp.id != id {
		return "", fmt.Errorf("rcon: response id mismatch: got %d, want %d", resp.id, id)
	}
	return resp.payload, nil
}

// Close closes the underlying TCP connection.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return nil
	}
	err := c.conn.Close()
	c.conn = nil
	return err
}

func (c *Client) nextID() int32 {
	return c.seq.Add(1)
}
