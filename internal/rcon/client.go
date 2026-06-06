package rcon

import (
	"errors"
	"fmt"
	"net"
	"sync"
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
	seq      int32 // guarded by mu; no atomic needed since all callers hold mu
}

func NewRCONClient(addr, password string, dialer Dialer) *Client {
	return &Client{addr: addr, password: password, dialer: dialer}
}

// Connect dials the server and authenticates. If a connection is already open
// it is closed first. Must be called before Exec.
func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		c.conn.Close() //nolint:errcheck
		c.conn = nil
	}

	conn, err := c.dialer.Dial(c.addr, dialTimeout)
	if err != nil {
		return fmt.Errorf("rcon: dial %s: %w", c.addr, err)
	}
	c.conn = conn

	if err := c.authenticate(); err != nil {
		c.conn.Close() //nolint:errcheck
		c.conn = nil
		return err
	}
	return nil
}

func (c *Client) authenticate() error {
	if err := c.conn.SetDeadline(time.Now().Add(rwTimeout)); err != nil {
		return fmt.Errorf("rcon: set auth deadline: %w", err)
	}
	defer c.conn.SetDeadline(time.Time{}) //nolint:errcheck

	id := c.nextID()
	if err := writePacket(c.conn, packet{id: id, kind: packetTypeAuth, payload: c.password}); err != nil {
		return fmt.Errorf("rcon: send auth: %w", err)
	}
	resp, err := readPacket(c.conn)
	if err != nil {
		return fmt.Errorf("rcon: read auth response: %w", err)
	}
	// The server replies with the same ID on success or -1 on failure.
	if resp.id == -1 || resp.id != id {
		return ErrAuthFailed
	}
	return nil
}

// Exec sends a command to the server and returns its response.
// On any I/O error the connection is closed and invalidated; call Connect again.
func (c *Client) Exec(cmd string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return "", ErrNotConnected
	}

	if err := c.conn.SetDeadline(time.Now().Add(rwTimeout)); err != nil {
		c.closeConn()
		return "", fmt.Errorf("rcon: set deadline: %w", err)
	}
	defer c.conn.SetDeadline(time.Time{}) //nolint:errcheck

	id := c.nextID()
	if err := writePacket(c.conn, packet{id: id, kind: packetTypeCommand, payload: cmd}); err != nil {
		c.closeConn()
		return "", fmt.Errorf("rcon: send command: %w", err)
	}
	resp, err := readPacket(c.conn)
	if err != nil {
		c.closeConn()
		return "", fmt.Errorf("rcon: read response: %w", err)
	}
	if resp.id != id {
		c.closeConn()
		return "", fmt.Errorf("rcon: response id mismatch: got %d, want %d", resp.id, id)
	}
	return resp.payload, nil
}

// Close closes the underlying TCP connection.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closeConn()
}

// closeConn closes c.conn and nils it out. Caller must hold c.mu.
func (c *Client) closeConn() error {
	if c.conn == nil {
		return nil
	}
	err := c.conn.Close()
	c.conn = nil
	return err
}

func (c *Client) nextID() int32 {
	c.seq++
	return c.seq
}
