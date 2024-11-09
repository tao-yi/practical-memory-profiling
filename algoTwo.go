package main

import (
	"bytes"
)

// algoTwo is a second way to solve the problem
func algoTwo(data []byte, find []byte, repl []byte, output *bytes.Buffer) {
	// Use the bytes Reader to provide a stream to process
	input := bytes.NewReader(data)
	// The number of bytes we are looking for
	size := len(find)
	// create an index variable to match bytes
	idx := 0
	for {
		// read a single byte from out input
		b, err := input.ReadByte()
		if err != nil {
			break
		}
		if b == find[idx] {
			idx++
			if idx == size {
				output.Write(repl)
				idx = 0
			}
			continue
		}

		// did we have any sort of match on any given byte?
		if idx != 0 {
			// write what we;ve matched up to this point
			output.Write(find[:idx])
			// unread the unmatched byte so it can be processed again
			input.UnreadByte()
			// reset the offset to start matching from the beginning
			idx = 0
			continue
		}
		// There was no previous match. Write byte and reset
		output.WriteByte(b)
		idx = 0
	}
}
