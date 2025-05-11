package editor

import (
	"bufio"
	"bytes"
	"fmt"
	"os"

	"github.com/tneuqole/texteditor/internal/keys"
	"github.com/tneuqole/texteditor/internal/version"
	"github.com/tneuqole/texteditor/internal/vt100"
)

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

func New(in, out *os.File) *Editor {
	return &Editor{
		In:  bufio.NewReader(in),
		Out: out,
		Buf: &bytes.Buffer{},
	}
}

/*** Core Logic ***/

func (e *Editor) Flush() {
	e.Buf.WriteTo(e.Out)
	e.Buf.Reset()
}

func (e *Editor) Die(err error) {
	e.MoveCursorTopLeft()
	e.ClearScreen()
	e.Flush()
	if err != nil {
		fmt.Fprintf(e.Out, "%T: %s\n", err, err) // TODO: write to stderr?
	}
}

func (e *Editor) ReadKey() (rune, error) {
	c, _, err := e.In.ReadRune()
	if err != nil {
		return 0, err
	}

	if c == vt100.Esc {
		c2, _, err := e.In.ReadRune()
		if err != nil {
			return c, nil
		}
		c3, _, err := e.In.ReadRune()
		if err != nil {
			return c, nil
		}

		if c2 == '[' {
			if c3 >= '0' && c3 <= '9' {
				c4, _, err := e.In.ReadRune()
				if err != nil {
					return c, nil
				}

				if c4 == '~' {
					switch c3 {
					case '5':
						return keys.PageUp, nil
					case '6':
						return keys.PageDown, nil
					}
				}
			} else {
				switch c3 {
				case 'A':
					return keys.ArrowUp, nil
				case 'B':
					return keys.ArrowDown, nil
				case 'C':
					return keys.ArrowRight, nil
				case 'D':
					return keys.ArrowLeft, nil
				}
			}
		}
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
	case keys.CtrlKey('q'):
		e.ClearScreen()
		e.Flush()
		e.Exit = true
	case keys.ArrowLeft:
		e.MoveCursor(c)
	case keys.ArrowRight:
		e.MoveCursor(c)
	case keys.ArrowUp:
		e.MoveCursor(c)
	case keys.ArrowDown:
		e.MoveCursor(c)
	case keys.PageUp:
		for range e.Rows {
			e.MoveCursor(keys.ArrowUp)
		}
	case keys.PageDown:
		for range e.Rows {
			e.MoveCursor(keys.ArrowDown)
		}
	default:
		fmt.Printf("%d: %c\n\r", c, c)
	}

	return nil
}

/*** Cursor Methods ***/

func (e *Editor) MoveCursor(c rune) {
	switch c {
	case keys.ArrowLeft:
		if e.Cx != 0 {
			e.Cx--
		}
	case keys.ArrowRight:
		if e.Cx < e.Cols-1 {
			e.Cx++
		}
	case keys.ArrowUp:
		if e.Cy > 0 {
			e.Cy--
		}
	case keys.ArrowDown:
		if e.Cy < e.Rows-1 {
			e.Cy++
		}
	}
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

/*** Screen Methods ***/

func (e *Editor) ClearScreen() {
	ed := vt100.EraseInDisplay{Arg: vt100.EDAll}
	ed.Write(e.Buf)
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

func (e *Editor) DrawRows() {
	el := vt100.EraseInLine{Arg: vt100.ELPosToEnd}
	for i := range e.Rows {
		e.Buf.WriteString("~")

		if i == e.Rows/3 {
			msg := "goeditor -- " + version.Version
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
