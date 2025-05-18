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

type Line struct {
	Size int
	Text []rune
}

type Editor struct {
	In         *bufio.Reader
	Out        *os.File
	Buf        *bytes.Buffer
	Exit       bool
	ScreenRows int // Window Size
	ScreenCols int
	CursorX    int // Cursor Position
	CursorY    int
	Lines      []Line
	NumLines   int
}

func New(in, out *os.File) *Editor {
	return &Editor{
		In:  bufio.NewReader(in),
		Out: out,
		Buf: &bytes.Buffer{},
	}
}

/*** Core Logic ***/

func (e *Editor) flush() {
	e.Buf.WriteTo(e.Out)
	e.Buf.Reset()
}

func (e *Editor) Die(err error) {
	e.moveCursorTopLeft()
	e.ClearScreen()
	e.flush()
	if err != nil {
		fmt.Fprintf(e.Out, "%T: %s\n", err, err) // TODO: write to stderr?
	}
}

func (e *Editor) ReadKey() (rune, error) {
	c, _, err := e.In.ReadRune()
	if err != nil {
		return 0, err
	}

	var seq [3]rune
	if c == vt100.Esc {
		seq[0], _, err = e.In.ReadRune()
		if err != nil {
			return c, nil
		}
		seq[1], _, err = e.In.ReadRune()
		if err != nil {
			return c, nil
		}

		if seq[0] == '[' {
			if seq[1] >= '0' && seq[1] <= '9' {
				seq[2], _, err = e.In.ReadRune()
				if err != nil {
					return c, nil
				}

				if seq[2] == '~' {
					switch seq[1] {
					case '1':
						return keys.Home, nil
					case '3':
						return keys.Del, nil
					case '4':
						return keys.End, nil
					case '5':
						return keys.PageUp, nil
					case '6':
						return keys.PageDown, nil
					case '7':
						return keys.Home, nil
					case '8':
						return keys.End, nil
					}
				}
			} else {
				switch seq[1] {
				case 'A':
					return keys.ArrowUp, nil
				case 'B':
					return keys.ArrowDown, nil
				case 'C':
					return keys.ArrowRight, nil
				case 'D':
					return keys.ArrowLeft, nil
				case 'H':
					return keys.Home, nil
				case 'F':
					return keys.End, nil
				}
			}
		} else if seq[0] == 'O' {
			switch seq[1] {
			case 'H':
				return keys.Home, nil
			case 'F':
				return keys.End, nil
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
		e.flush()
		e.Exit = true
	case keys.ArrowLeft, keys.ArrowRight, keys.ArrowUp, keys.ArrowDown:
		e.MoveCursor(c)
	case keys.PageUp:
		for range e.ScreenRows {
			e.MoveCursor(keys.ArrowUp)
		}
	case keys.PageDown:
		for range e.ScreenRows {
			e.MoveCursor(keys.ArrowDown)
		}
	case keys.Home:
		e.CursorX = 0
	case keys.End:
		e.CursorX = e.ScreenCols - 1
	default:
		fmt.Printf("%d: %c\n\r", c, c)
	}

	return nil
}

/*** Cursor Methods ***/

func (e *Editor) MoveCursor(c rune) {
	switch c {
	case keys.ArrowLeft:
		if e.CursorX != 0 {
			e.CursorX--
		}
	case keys.ArrowRight:
		if e.CursorX < e.ScreenCols-1 {
			e.CursorX++
		}
	case keys.ArrowUp:
		if e.CursorY > 0 {
			e.CursorY--
		}
	case keys.ArrowDown:
		if e.CursorY < e.ScreenRows-1 {
			e.CursorY++
		}
	}
}

func (e *Editor) moveCursorTopLeft() {
	cp := vt100.CursorPosition{Row: 1, Column: 1}
	cp.Write(e.Buf)
}

func (e *Editor) hideCursor() {
	rm := vt100.ResetMode{Arg: vt100.ModeCursorVisible}
	rm.Write(e.Buf)
}

func (e *Editor) showCursor() {
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
	e.hideCursor()
	// why do you need to do this to draw the screen?
	// removing it results in weird behavior when moving the cursor
	e.moveCursorTopLeft()
	e.drawRows()

	cp := vt100.CursorPosition{Row: e.CursorY + 1, Column: e.CursorX + 1}
	cp.Write(e.Buf)

	e.showCursor()

	e.flush()
}

func (e *Editor) drawRows() {
	el := vt100.EraseInLine{Arg: vt100.ELPosToEnd}
	for y := range e.ScreenRows {
		if y >= e.NumLines {
			e.Buf.WriteString("~")

			if e.NumLines == 0 && y == e.ScreenRows/3 {
				msg := "goeditor -- " + version.Version
				for range (e.ScreenCols - len(msg)) / 2 {
					e.Buf.WriteString(" ")
				}
				e.Buf.WriteString(msg)
			}

		} else {
			// TODO: handle overflow
			e.Buf.WriteString(string(e.Lines[y].Text))
		}

		el.Write(e.Buf)

		if y < e.ScreenRows-1 {
			e.Buf.WriteString("\n\r")
		}
	}
}

/*** line operations ***/

func (e *Editor) AppendLine(text []rune) {
	l := Line{
		Text: text,
		Size: len(text),
	}
	e.Lines = append(e.Lines, l)
	e.NumLines++
}

/*** file i/o ***/

func (e *Editor) Open(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		text := scanner.Text()
		e.AppendLine([]rune(text))
	}

	return nil
}
