package rcon

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

const (
	packetTypeAuth    int32 = 3
	packetTypeCommand int32 = 2
	packetTypeResp    int32 = 0

	// minBodyLen = id(4) + kind(4) + empty payload + null(1) + null(1)
	minBodyLen int32 = 10
	// maxPayload is the documented RCON payload limit (bytes, payload only).
	maxPayload int32 = 1446
	// maxBodyLen = id(4) + kind(4) + maxPayload + null(1) + null(1)
	maxBodyLen int32 = 8 + maxPayload + 2
)

type packet struct {
	id      int32
	kind    int32
	payload string
}

func writePacket(w io.Writer, p packet) error {
	payload := []byte(p.payload)
	if int32(len(payload)) > maxPayload {
		return fmt.Errorf("rcon: payload too large (%d bytes, max %d)", len(payload), maxPayload)
	}

	// bodyLen = id(4) + kind(4) + payload + null(1) + null(1)
	bodyLen := int32(4 + 4 + len(payload) + 2)

	buf := new(bytes.Buffer)
	// bytes.Buffer.Write never fails; errors are explicitly ignored.
	binary.Write(buf, binary.LittleEndian, bodyLen) //nolint:errcheck
	binary.Write(buf, binary.LittleEndian, p.id)    //nolint:errcheck
	binary.Write(buf, binary.LittleEndian, p.kind)  //nolint:errcheck
	buf.Write(payload)
	buf.WriteByte(0)
	buf.WriteByte(0)

	_, err := w.Write(buf.Bytes())
	return err
}

func readPacket(r io.Reader) (packet, error) {
	var bodyLen int32
	if err := binary.Read(r, binary.LittleEndian, &bodyLen); err != nil {
		return packet{}, fmt.Errorf("rcon: read length: %w", err)
	}
	if bodyLen < minBodyLen || bodyLen > maxBodyLen {
		return packet{}, fmt.Errorf("rcon: invalid packet length %d", bodyLen)
	}

	body := make([]byte, bodyLen)
	if _, err := io.ReadFull(r, body); err != nil {
		return packet{}, fmt.Errorf("rcon: read body: %w", err)
	}

	return packet{
		id:      int32(binary.LittleEndian.Uint32(body[0:4])),
		kind:    int32(binary.LittleEndian.Uint32(body[4:8])),
		payload: string(body[8 : len(body)-2]),
	}, nil
}
