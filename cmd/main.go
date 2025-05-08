package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"

	"github.com/tneuqole/texteditor/internal/vt100"
	"golang.org/x/term"
)

const Version = "1.0.0"

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
	Rows int // Window Size
	Cols int
	Cx   int // Cursor Position
	Cy   int
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

	// TODO: read 2 more chars and set deadline to 100 ms
	// handle timeout error --> return Esc
	if c == vt100.Esc {
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
	case 'h':
		e.Cx--
	case 'j':
		e.Cy++
	case 'k':
		e.Cy--
	case 'l':
		e.Cx++
	default:
		fmt.Printf("%d: %c\n\r", c, c)
	}

	return nil
}

func (e *Editor) ClearScreen() {
	ed := vt100.EraseInDisplay{Arg: vt100.EDAll}
	ed.Write(e.Buf)
}

func (e *Editor) MoveCursorTopLeft() {
	cp := vt100.CursorPosition{Row: 1, Column: 1}
	cp.Write(e.Buf)
}

func (e *Editor) HideCursor() {
	rm := vt100.ResetMode{Arg: vt100.ModeCursorVisible}
	rm.Write(e.Buf)
}

func (e *Editor) ShowCursor() {
	sm := vt100.SetMode{Arg: vt100.ModeCursorVisible}
	sm.Write(e.Buf)
}

func (e *Editor) RefreshScreen() {
	e.HideCursor()
	// why do you need to do this to draw the screen?
	// removing it results in weird behavior when moving the cursor
	e.MoveCursorTopLeft()
	e.DrawRows()

	cp := vt100.CursorPosition{Row: e.Cy + 1, Column: e.Cx + 1}
	cp.Write(e.Buf)

	e.ShowCursor()

	e.Flush()
}

func (e *Editor) Die(err error) {
	e.ClearScreen()
	e.Out.WriteString(err.Error()) // TODO: write to stderr?
}

func (e *Editor) DrawRows() {
	el := vt100.EraseInLine{Arg: vt100.ELPosToEnd}
	for i := range e.Rows {
		e.Buf.WriteString("~")

		if i == e.Rows/3 {
			msg := "goeditor -- " + Version
			for range (e.Cols - len(msg)) / 2 {
				e.Buf.WriteString(" ")
			}
			e.Buf.WriteString(msg)
		}

		el.Write(e.Buf)
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

	cols, rows, err := term.GetSize(fd)
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
