package main

import (
	"bytes"
	"io"
)

func algoOne(data []byte, find []byte, repl []byte, output *bytes.Buffer) {
	// use a bytes buffer to provide a stream to process
	input := bytes.NewBuffer(data)

	// the number of bytes we are looking for
	size := len(find)

	// declare the buffers we need to process the stream.
	buf := make([]byte, size)
	end := size - 1

	// Read in an initial number of bytes we need to get started
	if n, err := io.ReadFull(input, buf[:end]); err != nil {
		output.Write(buf[:n])
		return
	}

	for {
		// read in one byte from the input stream
		if _, err := io.ReadFull(input, buf[end:]); err != nil {
			output.Write(buf[:end])
			return
		}

		// if we have a match, replace the bytes
		if bytes.Equal(buf, find) {
			output.Write(repl)
			// read a new initial number of bytes
			if n, err := io.ReadFull(input, buf[:end]); err != nil {
				output.Write(buf[:n])
				return
			}
			continue
		}

		// write the front byte since it has been compared
		output.WriteByte(buf[0])
		// slice that front byte out
		copy(buf, buf[1:])
	}
}
