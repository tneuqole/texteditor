package vt100

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)

const EscSeq = "\x1b["

// TODO: if performance is a concern, use a string builder
func write(buf *bytes.Buffer, format string, args ...any) {
	buf.WriteString(EscSeq)
	fmt.Fprintf(buf, format, args...)
}

func read(r io.Reader, format string, args ...any) error {
	var buf [32]byte

	reader := bufio.NewReader(r)
	for i := range len(buf) - 1 {
		b, err := reader.ReadByte()
		if err != nil {
			return err
		}

		buf[i] = b
		if b == 'R' {
			break
		}

	}

	br := bytes.NewReader(buf[:])
	_, err := fmt.Fscanf(br, EscSeq+format, args...)
	return err
}

type VT100WriteCommand interface {
	Write(buf *bytes.Buffer)
}

type VT100ReadCommand interface {
	Read(r io.Reader) error
}

type DSRArg int

const (
	DSRStatus   = 5
	DSRPosition = 6
)

type DeviceStatusReport struct {
	Arg DSRArg
}

func (cmd *DeviceStatusReport) Write(buf *bytes.Buffer) {
	write(buf, "%dn", cmd.Arg)
}

type CursorPositionReport struct {
	Row    int
	Column int
}

func (cmd *CursorPositionReport) Read(r io.Reader) error {
	return read(r, "%d;%dR", &cmd.Row, &cmd.Column)
}

type EIDArg int

const (
	EIDPosToEnd   = 0
	EIDStartToPos = 1
	EIDAll        = 2
)

type EraseInDisplay struct {
	Arg EIDArg
}

func (cmd *EraseInDisplay) Write(buf *bytes.Buffer) {
	write(buf, "%dJ", cmd.Arg)
}

type CursorPosition struct {
	Row    int
	Column int
}

func (cmd *CursorPosition) Write(buf *bytes.Buffer) {
	write(buf, "%d;%dH", cmd.Row, cmd.Column)
}
