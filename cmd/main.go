package main

import (
	"fmt"
	"log/slog"
	"os"

	"golang.org/x/term"
)

type Editor struct {
	Logger *slog.Logger
	Exit   bool
}

// Ctrl+c sets bits 5 & 6 of c to 0
// Use & to convert c to Ctrl-c
func CtrlKey(c rune) rune {
	return c & 0b0011111
}

func (e *Editor) ReadKey() rune {
	var c rune
	_, err := fmt.Scanf("%c", &c)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%d: %c\n\r", c, c)

	return c
}

func (e *Editor) ProcessKey(c rune) {
	switch c {
	case CtrlKey('q'):
		e.Exit = true
	case CtrlKey('r'):
		e.RefreshScreen()
	}
}

func (e *Editor) RefreshScreen() {
	fmt.Print("\x1b[2J")
	fmt.Print("\x1b[H")
}

func (e *Editor) Die(err error) {
	e.RefreshScreen()
	fmt.Println(err.Error())
	e.Exit = true
}

func (e *Editor) DrawRows() {
	for range 24 {
		fmt.Print("~\r\n")
	}
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	e := Editor{
		Logger: logger,
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		e.Die(err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	e.RefreshScreen()
	e.DrawRows()

	for !e.Exit {
		key := e.ReadKey()
		e.ProcessKey(key)

	}
}
