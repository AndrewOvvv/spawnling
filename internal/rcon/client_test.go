package rcon_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/AndrewOvvv/spawnling/internal/rcon"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── Dialer stub ───────────────────────────────────────────────────────────────

type pipeDialer struct{ conn net.Conn }

func (d *pipeDialer) Dial(_ string, _ time.Duration) (net.Conn, error) {
	return d.conn, nil
}

type errDialer struct{ err error }

func (d *errDialer) Dial(_ string, _ time.Duration) (net.Conn, error) {
	return nil, d.err
}

// ── Fake RCON server ──────────────────────────────────────────────────────────

// fakeRCON implements a minimal RCON server for testing.
// It validates the password and replies to commands from the responses map.
type fakeRCON struct {
	password  string
	responses map[string]string
}

func (s *fakeRCON) serve(t *testing.T, conn net.Conn) {
	t.Helper()
	defer conn.Close()

	// Auth handshake.
	id, _, pwd, err := recvPacket(conn)
	if err != nil {
		return
	}
	if pwd != s.password {
		sendPacket(conn, -1, 2, "") //nolint:errcheck
		return
	}
	sendPacket(conn, id, 2, "") //nolint:errcheck

	// Command loop.
	for {
		id, _, cmd, err := recvPacket(conn)
		if err != nil {
			return
		}
		resp := s.responses[cmd]
		if err := sendPacket(conn, id, 0, resp); err != nil {
			return
		}
	}
}

// newFakeClient wires a Client to a fakeRCON server over an in-memory pipe.
func newFakeClient(t *testing.T, srv *fakeRCON) *rcon.Client {
	t.Helper()
	clientConn, serverConn := net.Pipe()
	t.Cleanup(func() { clientConn.Close(); serverConn.Close() })
	go srv.serve(t, serverConn)
	return rcon.NewRCONClient("", srv.password, &pipeDialer{conn: clientConn})
}

// ── Protocol helpers for the fake server ─────────────────────────────────────
// These intentionally duplicate the production encoding so that the tests
// remain independent of unexported package internals.

func sendPacket(w io.Writer, id, kind int32, payload string) error {
	b := []byte(payload)
	bodyLen := int32(4 + 4 + len(b) + 2)
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, bodyLen) //nolint:errcheck
	binary.Write(buf, binary.LittleEndian, id)      //nolint:errcheck
	binary.Write(buf, binary.LittleEndian, kind)    //nolint:errcheck
	buf.Write(b)
	buf.WriteByte(0)
	buf.WriteByte(0)
	_, err := w.Write(buf.Bytes())
	return err
}

func recvPacket(r io.Reader) (id, kind int32, payload string, err error) {
	var bodyLen int32
	if err = binary.Read(r, binary.LittleEndian, &bodyLen); err != nil {
		return
	}
	body := make([]byte, bodyLen)
	if _, err = io.ReadFull(r, body); err != nil {
		return
	}
	id = int32(binary.LittleEndian.Uint32(body[0:4]))
	kind = int32(binary.LittleEndian.Uint32(body[4:8]))
	payload = string(body[8 : len(body)-2])
	return
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestConnect_Success(t *testing.T) {
	client := newFakeClient(t, &fakeRCON{password: "secret"})
	require.NoError(t, client.Connect())
	client.Close()
}

func TestConnect_WrongPassword(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	t.Cleanup(func() { clientConn.Close(); serverConn.Close() })

	srv := &fakeRCON{password: "secret"}
	go srv.serve(t, serverConn)

	client := rcon.NewRCONClient("", "wrongpassword", &pipeDialer{conn: clientConn})
	err := client.Connect()

	require.Error(t, err)
	assert.True(t, errors.Is(err, rcon.ErrAuthFailed))
}

func TestConnect_DialError(t *testing.T) {
	dialErr := errors.New("connection refused")
	client := rcon.NewRCONClient("localhost:25575", "secret", &errDialer{err: dialErr})

	err := client.Connect()

	require.Error(t, err)
	assert.ErrorContains(t, err, "connection refused")
}

func TestExec_ReturnsResponse(t *testing.T) {
	srv := &fakeRCON{
		password:  "secret",
		responses: map[string]string{"list": "There are 2 players: Alice, Bob"},
	}
	client := newFakeClient(t, srv)
	require.NoError(t, client.Connect())

	resp, err := client.Exec("list")

	require.NoError(t, err)
	assert.Equal(t, "There are 2 players: Alice, Bob", resp)
}

func TestExec_EmptyResponse(t *testing.T) {
	srv := &fakeRCON{
		password:  "secret",
		responses: map[string]string{"say hello": ""},
	}
	client := newFakeClient(t, srv)
	require.NoError(t, client.Connect())

	resp, err := client.Exec("say hello")

	require.NoError(t, err)
	assert.Empty(t, resp)
}

func TestExec_MultipleSequentialCommands(t *testing.T) {
	srv := &fakeRCON{
		password: "secret",
		responses: map[string]string{
			"time set day":  "",
			"weather clear": "",
			"list":          "There are 0 players",
		},
	}
	client := newFakeClient(t, srv)
	require.NoError(t, client.Connect())

	_, err := client.Exec("time set day")
	require.NoError(t, err)

	_, err = client.Exec("weather clear")
	require.NoError(t, err)

	resp, err := client.Exec("list")
	require.NoError(t, err)
	assert.Equal(t, "There are 0 players", resp)
}

func TestExec_NotConnected(t *testing.T) {
	client := rcon.NewRCONClient("localhost:25575", "secret", &errDialer{})

	_, err := client.Exec("list")

	require.Error(t, err)
	assert.True(t, errors.Is(err, rcon.ErrNotConnected))
}

func TestClose_Idempotent(t *testing.T) {
	client := newFakeClient(t, &fakeRCON{password: "secret"})
	require.NoError(t, client.Connect())

	assert.NoError(t, client.Close())
	assert.NoError(t, client.Close()) // second close must not error
}
