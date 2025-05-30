package vt100

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)

const (
	Esc    = '\x1b'
	EscSeq = "\x1b["
)

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

type EDArg int

const (
	EDPosToEnd   = 0
	EDStartToPos = 1
	EDAll        = 2
)

type EraseInDisplay struct {
	Arg EDArg
}

func (cmd *EraseInDisplay) Write(buf *bytes.Buffer) {
	write(buf, "%dJ", cmd.Arg)
}

type ELArg int

const (
	ELPosToEnd   = 0
	ELStartToPos = 1
	ELAll        = 2
)

type EraseInLine struct {
	Arg ELArg
}

func (cmd *EraseInLine) Write(buf *bytes.Buffer) {
	write(buf, "%dK", cmd.Arg)
}

type CursorPosition struct {
	Row    int
	Column int
}

func (cmd *CursorPosition) Write(buf *bytes.Buffer) {
	write(buf, "%d;%dH", cmd.Row, cmd.Column)
}

type ModeArg string

const ModeCursorVisible = "?25"

type SetMode struct {
	Arg ModeArg
}

func (cmd *SetMode) Write(buf *bytes.Buffer) {
	write(buf, "%sh", cmd.Arg)
}

type ResetMode struct {
	Arg ModeArg
}

func (cmd *ResetMode) Write(buf *bytes.Buffer) {
	write(buf, "%sl", cmd.Arg)
}
