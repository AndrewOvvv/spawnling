package rcon

import "fmt"

// Client is a Minecraft RCON client (TCP, port 25575 by default).
type Client struct {
	host     string
	port     int
	password string
}

func NewRCONClient(host string, port int, password string) *Client {
	return &Client{host: host, port: port, password: password}
}

func (c *Client) Exec(cmd string) (string, error) {
	// TODO: implement RCON protocol (https://wiki.vg/RCON)
	panic(fmt.Sprintf("rcon: not implemented (cmd=%q)", cmd))
}

func (c *Client) Close() error {
	return nil
}
