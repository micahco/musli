package progress

import (
	"github.com/micahco/musli/cmd/musli/term"
)

const defaultWidth = 10

func Bar(cur, total int) string {
	w, _, err := term.GetSize()
	if err != nil {
		w = defaultWidth
	} else {
		w = w / 2
	}
	p := cur * w / total
	b := make([]rune, w)
	for i := range b {
		if i < p {
			b[i] = '='
		} else {
			b[i] = '_'
		}
	}
	return "[" + string(b) + "]"
}