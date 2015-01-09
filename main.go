/*
delay takes a stream of data on the standard input and writes it to the
standard output after a specified delay.
*/
package main

import (
	"bufio"
	"flag"
	"io"
	"log"
	"os"
	"time"
)

var delay = time.Minute

type data struct {
	b []byte
	t time.Time
}

func main() {
	log.SetFlags(0)

	flag.DurationVar(&delay, "d", time.Minute, "the amount of time to hold data before re-sending it")
	flag.Parse()

	if delay < 0 {
		log.Fatalln("delay: cannot wait negative time:", delay)
	}

	ch := make(chan data)

	go func() {
		defer close(ch)

		buffer := make(chan data)

		go func() {
			defer close(buffer)

			r := bufio.NewReader(os.Stdin)

			for {
				b := make([]byte, 1<<10)
				n, err := r.Read(b)
				if err == io.EOF {
					return
				}
				if err != nil {
					log.Fatalln("delay: input error:", err)
				}
				buffer <- data{b[:n], time.Now()}
			}
		}()

		// IO can be slow, so we buffer the data in a goroutine that doesn't do IO.
		var queue []data
		for {
			var (
				send data
				out  chan<- data
			)
			if len(queue) != 0 {
				send = queue[0]
				out = ch
			} else if buffer == nil {
				return
			}

			select {
			case d, ok := <-buffer:
				if ok {
					queue = append(queue, d)
				} else {
					buffer = nil
				}

			case out <- send:
				queue = queue[1:]
			}
		}
	}()

	var queue []data
	var next <-chan time.Time
	for {
		if ch == nil && len(queue) == 0 {
			return
		}

		select {
		case d, ok := <-ch:
			if ok {
				queue = append(queue, d)
				if next == nil {
					next = nextDelay(d)
				}
			} else {
				ch = nil
			}

		case <-next:
			d := queue[0]
			queue = queue[1:]

			if len(queue) > 0 {
				next = nextDelay(queue[0])
			} else {
				next = nil
			}

			n, err := os.Stdout.Write(d.b)
			if err != nil {
				log.Fatalln("delay: output error:", err)
			}
			if n != len(d.b) {
				log.Fatalln("delay: short write:", n, "!=", len(d.b))
			}
		}
	}
}

func nextDelay(d data) <-chan time.Time {
	duration := -time.Since(d.t) + delay
	if duration < 0 {
		duration = 0
	}
	return time.After(duration)
}
