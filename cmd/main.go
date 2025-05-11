package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/tneuqole/texteditor/internal/vt100"
	"golang.org/x/sys/unix"
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
			switch c3 {
			case 'A':
				return 'k', nil
			case 'B':
				return 'j', nil
			case 'C':
				return 'l', nil
			case 'D':
				return 'h', nil
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
	e.MoveCursorTopLeft()
	e.ClearScreen()
	e.Flush()
	if err != nil {
		fmt.Fprintf(e.Out, "%T: %s\n", err, err) // TODO: write to stderr?
	}
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

// based on term.MakeRaw
func makeRaw(fd int) (*unix.Termios, error) {
	oldTermios, err := unix.IoctlGetTermios(fd, unix.TIOCGETA)
	if err != nil {
		return nil, err
	}

	rawTermios := *oldTermios

	rawTermios.Iflag &^= unix.IGNBRK | unix.BRKINT | unix.PARMRK | unix.ISTRIP | unix.INLCR | unix.IGNCR | unix.ICRNL | unix.IXON
	rawTermios.Oflag &^= unix.OPOST
	rawTermios.Lflag &^= unix.ECHO | unix.ECHONL | unix.ICANON | unix.ISIG | unix.IEXTEN
	rawTermios.Cflag &^= unix.CSIZE | unix.PARENB
	rawTermios.Cflag |= unix.CS8

	// set min bytes to 0 and timeout to 100ms so reading doesn't block
	rawTermios.Cc[unix.VMIN] = 0
	rawTermios.Cc[unix.VTIME] = 1

	if err := unix.IoctlSetTermios(fd, unix.TIOCSETA, &rawTermios); err != nil {
		return nil, err
	}

	return oldTermios, nil
}

func main() {
	e := Editor{
		In:  bufio.NewReader(os.Stdin),
		Out: os.Stdout,
		Buf: &bytes.Buffer{},
	}

	fd := int(os.Stdin.Fd())
	oldTermios, err := makeRaw(fd)
	if err != nil {
		e.Die(err)
		return
	}

	defer unix.IoctlSetTermios(fd, unix.TIOCSETA, oldTermios)

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
		if err != nil && err != io.EOF {
			e.Die(err)
			return
		}

	}
	e.Die(nil)
}
