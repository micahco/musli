package spinner

import (
	"time"

	"github.com/micahco/musli/cmd/musli/term"
)

var Dots = []string{"[.  ]", "[.. ]", "[...]", "[ ..]", "[  .]", "[   ]", "[  .]", "[ ..]", "[...]", "[.. ]", "[.  ]", "[   ]"}

func Spin(ch chan struct{}, delay int64, caption string, frames []string) {
	for {
		for _, f := range frames {
			select {
			case <-ch:
				return
			default:
				term.ClearLine(f, caption)
				time.Sleep(time.Duration(delay) * time.Millisecond)
			}
		}
	}
}