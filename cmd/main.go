package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"

	"github.com/tneuqole/texteditor/internal/vt100"
	"golang.org/x/term"
)

// Ctrl+c sets bits 5 & 6 of c to 0
// Use & to convert c to Ctrl-c
func CtrlKey(c rune) rune {
	return c & 0b0011111
}

type Editor struct {
	In   *bufio.Reader
	Out  *os.File
	Buf  *bytes.Buffer
	Exit bool
	Rows int
	Cols int
}

func (e *Editor) Flush() {
	e.Buf.WriteTo(e.Out)
	e.Buf.Reset()
}

func (e *Editor) ReadKey() (rune, error) {
	c, _, err := e.In.ReadRune()
	if err != nil {
		return 0, err
	}

	// fmt.Printf("%d: %c\n\r", c, c)

	return c, nil
}

func (e *Editor) ProcessKey() error {
	c, err := e.ReadKey()
	if err != nil {
		return err
	}

	switch c {
	case CtrlKey('q'):
		e.ClearScreen()
		e.Flush()
		e.Exit = true
	default:
		fmt.Printf("%d: %c\n\r", c, c)
	}

	return nil
}

func (e *Editor) ClearScreen() {
	eid := vt100.EraseInDisplay{Arg: vt100.EIDAll}
	eid.Write(e.Buf)
}

func (e *Editor) MoveCursorTopLeft() {
	cp := vt100.CursorPosition{Row: 1, Column: 1}
	cp.Write(e.Buf)
}

func (e *Editor) RefreshScreen() {
	e.ClearScreen()
	e.DrawRows()
	e.MoveCursorTopLeft()

	e.Flush()
}

func (e *Editor) Die(err error) {
	e.ClearScreen()
	e.Out.WriteString(err.Error()) // TODO: write to stderr?
}

func (e *Editor) DrawRows() {
	for i := range e.Rows {
		e.Buf.WriteString("~")
		if i < e.Rows-1 {
			e.Buf.WriteString("\n\r")
		}
	}
}

func (e *Editor) GetCursorPosition() (*vt100.CursorPositionReport, error) {
	var buf bytes.Buffer
	dsr := vt100.DeviceStatusReport{
		Arg: vt100.DSRPosition,
	}
	dsr.Write(&buf)
	fmt.Print(buf.String())

	var cpr vt100.CursorPositionReport
	cpr.Read(e.In)

	// fmt.Printf("row=%d, col=%d\n\r", cpr.Row, cpr.Column)

	return &cpr, nil
}

func main() {
	e := Editor{
		In:  bufio.NewReader(os.Stdin),
		Out: os.Stdout,
		Buf: &bytes.Buffer{},
	}

	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		e.Die(err)
		return
	}
	defer term.Restore(fd, oldState)

	rows, cols, err := term.GetSize(fd)
	if err != nil {
		e.Die(err)
		return
	}

	e.Rows = rows
	e.Cols = cols

	for !e.Exit {
		e.RefreshScreen()
		err = e.ProcessKey()
		if err != nil {
			e.Die(err)
			return
		}

	}
}
