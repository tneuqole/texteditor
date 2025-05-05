package rope

type Node struct {
	Parent    *Node
	Left      *Node
	Right     *Node
	CharsLeft int64
	Text      []string
}

func New(parent *Node, charsLeft int64, text []string) *Node {
	return &Node{
		Parent:    parent,
		CharsLeft: charsLeft,
		Text:      text,
	}
}
