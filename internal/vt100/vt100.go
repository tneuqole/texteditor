package vt100

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)

type VT100Command interface {
	Write(buf *bytes.Buffer)
	Read(r io.Reader) error
}

type DSRArg int

const (
	ArgDSRStatus   = 5
	ArgDSRPosition = 6
)

type DeviceStatusReport struct {
	Arg DSRArg
}

func (cmd *DeviceStatusReport) Write(buf *bytes.Buffer) {
	fmt.Fprintf(buf, "\x1b[%dn", cmd.Arg)
}

func (cmd *DeviceStatusReport) Read(r io.Reader) error {
	return nil
}

type CursorPosition struct {
	Row    int
	Column int
}

func (cmd *CursorPosition) Write(buf *bytes.Buffer) {}

func (cmd *CursorPosition) Read(r io.Reader) error {
	var bytes [32]byte

	reader := bufio.NewReader(r)
	for i := 0; i < len(bytes)-1; i++ {
		b, err := reader.ReadByte()
		if err != nil {
			return err
		}

		bytes[i] = b
		if b == 'R' {
			break
		}

	}

	_, err := fmt.Sscanf(string(bytes[:]), "\x1b[%d;%dR", &cmd.Row, &cmd.Column)
	return err
}
