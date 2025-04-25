package main

import (
	"testing"
)

var (
	testFile = "en_full.txt"
	testSize = 6
)

func Benchmark_NewLexiconOld(b *testing.B) {
	LexHook = func(s *Phrase) bool {
		return false
	}

	verbose = false
	for i := 0; i < b.N; i++ {
		_ = newLexiconFromFileOld(testFile, testSize)
	}
}

func Benchmark_NewLexicon(b *testing.B) {
	LexHook = func(s *Phrase) bool {
		return false
	}

	verbose = false
	for i := 0; i < b.N; i++ {
		_ = NewLexiconFromFile(testFile, testSize)
	}
}

func Benchmark_NewLexiconOldVerbose(b *testing.B) {
	LexHook = func(s *Phrase) bool {
		return false
	}

	verbose = true

	for i := 0; i < b.N; i++ {
		_ = newLexiconFromFileOld(testFile, testSize)
	}
}

func Benchmark_NewLexiconVerbose(b *testing.B) {
	LexHook = func(s *Phrase) bool {
		return false
	}

	verbose = true
	for i := 0; i < b.N; i++ {
		_ = NewLexiconFromFile(testFile, testSize)
	}
}
