package main

import (
	"io"
	"os"

	"github.com/tneuqole/texteditor/internal/editor"
	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

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
	e := editor.New(os.Stdin, os.Stdout)

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
