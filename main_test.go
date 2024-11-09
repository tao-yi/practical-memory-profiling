package main

import (
	"bytes"
	"testing"
)

var output bytes.Buffer
var in = assembleInputStream()
var find = []byte("elvis")
var repl = []byte("Elvis")

func BenchmarkAlgorithmOne(b *testing.B) {
	for range b.N {
		output.Reset()
		algoOne(in, find, repl, &output)
	}
}

func BenchmarkAlgorithmTwo(b *testing.B) {
	for range b.N {
		output.Reset()
		algoTwo(in, find, repl, &output)
	}
}
