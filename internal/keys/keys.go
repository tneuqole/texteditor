package keys

const (
	ArrowLeft = iota + 1000
	ArrowRight
	ArrowUp
	ArrowDown
	PageUp
	PageDown
	Home
	End
	Del
)

// Ctrl+c sets bits 5 & 6 of c to 0
// Use & to convert c to Ctrl-c
func CtrlKey(c rune) rune {
	return c & 0b0011111
}
