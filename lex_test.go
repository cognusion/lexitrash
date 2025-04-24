package main

import (
	"testing"
)

var testFile = "en_full.txt"

func Benchmark_NewLexiconOld(b *testing.B) {
	verbose = false
	for i := 0; i < b.N; i++ {
		_ = newLexiconFromFileOld(testFile, 6)
	}
}

func Benchmark_NewLexicon(b *testing.B) {
	verbose = false
	for i := 0; i < b.N; i++ {
		_ = NewLexiconFromFile(testFile, 6)
	}
}

func Benchmark_NewLexiconOldVerbose(b *testing.B) {
	verbose = true
	for i := 0; i < b.N; i++ {
		_ = newLexiconFromFileOld(testFile, 6)
	}
}

func Benchmark_NewLexiconVerbose(b *testing.B) {
	verbose = true
	for i := 0; i < b.N; i++ {
		_ = NewLexiconFromFile(testFile, 6)
	}
}
