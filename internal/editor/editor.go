package editor

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/tneuqole/texteditor/internal/keys"
	"github.com/tneuqole/texteditor/internal/version"
	"github.com/tneuqole/texteditor/internal/vt100"
)

const (
	TabStop = 4
	Tab     = '\t'
)

// store raw text and formatted text.
// see updateLine() for formatting details
type Line struct {
	Raw       []rune
	RSize     int
	Formatted []rune
	FSize     int
}

// TODO: make fields private where applicable
type Editor struct {
	In         *bufio.Reader
	Out        *os.File
	Buf        *bytes.Buffer
	Exit       bool
	ScreenRows int // Window Size
	ScreenCols int
	CursorX    int // Cursor Position in raw line
	CursorXf   int // Cursor Position in formatted line
	CursorY    int
	Lines      []Line
	NumLines   int
	RowOffset  int
	ColOffset  int
	filename   *string
	statusMsg  string
	statusTime time.Time
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

func (e *Editor) readKey() (rune, error) {
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

		switch seq[0] {
		case '[':
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
		case 'O':
			switch seq[1] {
			case 'H':
				return keys.Home, nil
			case 'F':
				return keys.End, nil
			}
		}
	}

	return c, nil
}

func (e *Editor) ProcessKey() error {
	c, err := e.readKey()
	if err != nil {
		return err
	}

	switch c {
	case keys.CtrlKey('q'):
		e.ClearScreen()
		e.flush()
		e.Exit = true
	case keys.ArrowLeft, keys.ArrowRight, keys.ArrowUp, keys.ArrowDown:
		e.moveCursor(c)
	case keys.PageUp:
		e.CursorY = e.RowOffset
		for range e.ScreenRows {
			e.moveCursor(keys.ArrowUp)
		}
	case keys.PageDown:
		e.CursorY = min(e.RowOffset+e.ScreenRows-1, e.NumLines)
		for range e.ScreenRows {
			e.moveCursor(keys.ArrowDown)
		}
	case keys.Home:
		e.CursorX = 0
	case keys.End:
		if e.CursorY < e.NumLines {
			e.CursorX = e.Lines[e.CursorY].FSize
		}
	default:
		fmt.Printf("%d: %c\n\r", c, c)
	}

	return nil
}

/*** Cursor Methods ***/

func (e *Editor) setCursorXf() {
	e.CursorXf = e.CursorX

	line := e.getCurrentLine()

	for i := range line.RSize {
		if line.Raw[i] == Tab {
			e.CursorXf = e.CursorXf + TabStop
		}
	}
}

func (e *Editor) moveCursor(c rune) {
	line := e.getCurrentLine()

	switch c {
	case keys.ArrowLeft:
		if e.CursorX > 0 {
			e.CursorX--
		} else if e.CursorY > 0 {
			// moving left at the beginning of a line
			// goes to end of prev line
			e.CursorY--
			e.CursorX = e.Lines[e.CursorY].RSize
		}
	case keys.ArrowRight:
		if e.CursorX < line.RSize {
			e.CursorX++
		} else if e.CursorX == line.RSize && e.NumLines > e.CursorY {
			// if we're at the end of the line and there is another line
			// below, moving right goes to beginning of next line
			e.CursorY++
			e.CursorX = 0
		}
	case keys.ArrowUp:
		if e.CursorY > 0 {
			e.CursorY--
		}
	case keys.ArrowDown:
		if e.CursorY < e.NumLines {
			e.CursorY++
		}
	}

	// get the new line if we are not at the end
	// of the file
	line = e.getCurrentLine()

	// if we went up or down, cx may be greater than
	// the new line length. in that case snap cx to the
	// end of the new line
	e.CursorX = min(e.CursorX, line.RSize)
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

/*** Screen Methods ***/

func (e *Editor) SetStatusMessage(msg string) {
	e.statusMsg = msg
	e.statusTime = time.Now()
}

func (e *Editor) drawStatusBar() {
	sgr := vt100.SelectGraphicRendition{Arg: vt100.SelectGraphicRenditionNegative}
	sgr.Write(e.Buf)

	var builder strings.Builder

	filename := "[No Name]"
	if e.filename != nil {
		filename = *e.filename
	}

	n := min(len(filename), 20)

	builder.WriteRune(' ')
	builder.WriteString(filename[:n])
	builder.WriteString(fmt.Sprintf(" - %d lines", e.NumLines))

	leftStatus := builder.String()
	e.Buf.WriteString(leftStatus)

	builder.Reset()

	builder.WriteString(fmt.Sprintf(" %d/%d ", e.CursorY+1, e.NumLines))

	rightStatus := builder.String()

	for range e.ScreenCols - (len(leftStatus) + len(rightStatus)) {
		e.Buf.WriteRune(' ')
	}

	e.Buf.WriteString(rightStatus)

	sgr = vt100.SelectGraphicRendition{Arg: vt100.SelectGraphicRenditionOff}
	sgr.Write(e.Buf)

	e.Buf.WriteString("\r\n")

	n = min(len(e.statusMsg), e.ScreenCols-1)
	statusMsg := e.statusMsg[:n]

	eil := vt100.EraseInLine{Arg: vt100.ELPosToEnd}
	eil.Write(e.Buf)

	if time.Since(e.statusTime) <= 5*time.Second {
		e.Buf.WriteRune(' ')
		e.Buf.WriteString(statusMsg)
	}
}

func (e *Editor) ClearScreen() {
	ed := vt100.EraseInDisplay{Arg: vt100.EDAll}
	ed.Write(e.Buf)
}

func (e *Editor) scroll() {
	e.setCursorXf()

	if e.CursorY < e.RowOffset {
		e.RowOffset = e.CursorY
	}

	if e.CursorY >= e.RowOffset+e.ScreenRows {
		e.RowOffset = e.CursorY - e.ScreenRows + 1
	}

	if e.CursorXf < e.ColOffset {
		e.ColOffset = e.CursorXf
	}

	if e.CursorXf >= e.ColOffset+e.ScreenCols {
		e.ColOffset = e.CursorXf - e.ScreenCols + 1
	}
}

func (e *Editor) RefreshScreen() {
	e.scroll()
	e.hideCursor()

	// TODO: why do you need to do this to draw the screen?
	// removing it results in weird behavior when moving the cursor
	e.moveCursorTopLeft()
	e.drawRows()
	e.drawStatusBar()

	cp := vt100.CursorPosition{Row: e.CursorY - e.RowOffset + 1, Column: e.CursorXf - e.ColOffset + 1}
	cp.Write(e.Buf)

	e.showCursor()

	e.flush()
}

func (e *Editor) drawRows() {
	el := vt100.EraseInLine{Arg: vt100.ELPosToEnd}
	for y := range e.ScreenRows {
		filerow := y + e.RowOffset
		if filerow >= e.NumLines {
			e.Buf.WriteString("~")

			if e.NumLines == 0 && y == e.ScreenRows/3 {
				msg := "goeditor -- " + version.Version
				for range (e.ScreenCols - len(msg)) / 2 {
					e.Buf.WriteString(" ")
				}
				e.Buf.WriteString(msg)
			}

		} else {
			l := max(0, e.Lines[filerow].FSize-e.ColOffset)
			l = min(l, e.ScreenCols)

			if e.Lines[filerow].FSize > e.ColOffset {
				e.Buf.WriteString(string(e.Lines[filerow].Formatted[e.ColOffset : e.ColOffset+l]))
			}

		}

		el.Write(e.Buf)

		e.Buf.WriteString("\n\r")
	}
}

/*** line operations ***/

func (e *Editor) getCurrentLine() Line {
	l := Line{}

	if e.CursorY < e.NumLines {
		l = e.Lines[e.CursorY]
	}

	return l
}

func (e *Editor) updateLine(line *Line) {
	for i := range line.RSize {
		c := line.Raw[i]
		if c == Tab {
			for range TabStop {
				line.Formatted = append(line.Formatted, ' ')
			}
		} else {
			line.Formatted = append(line.Formatted, c)
		}
	}

	line.FSize = len(line.Formatted)
}

func (e *Editor) appendLine(text []rune) {
	l := Line{
		Raw:       text,
		RSize:     len(text),
		Formatted: []rune{},
	}

	e.updateLine(&l)

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
		e.appendLine([]rune(text))
	}

	e.filename = &filename

	return nil
}
