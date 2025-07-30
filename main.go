package main

import (
	"fmt"
	"strings"

	"github.com/spf13/pflag"
)

var (
	verbose bool
)

func scour(must []byte, can []byte) func(*Phrase) bool {

	var (
		musti []int = l2i(must)
		cani  []int = letterFrequency(string(can))
	)

	return func(w *Phrase) bool {
		if w == nil {
			return false
		}

		for _, i := range musti {
			if w.LetterCount[i] == 0 {
				return false
			}
		}
		// POST: we have the musts

		for i, c := range cani {
			if c == 0 && w.LetterCount[i] > 0 {
				return false
			}
		}
		// POST: We don't have the can'ts

		return true
	}
}

func l2i(ls []byte) []int {
	out := make([]int, len(ls))

	for i, b := range ls {
		l := rune(b)
		out[i] = aIndex[l]
	}
	return out
}

func main() {
	var (
		file    *string = pflag.String("file", "./en_full.txt", "Use a different dictionary source")
		must    *string = pflag.String("must", "", "List of letters that MUST be in the output")
		can     *string = pflag.String("may", "", "List of NON-MUST letters that may also be in the output")
		minSize *int    = pflag.Int("size", 6, "Minimum length a word must be to be output")
		maxSize *int    = pflag.Int("max", 0, "Maximum length a word can be to be output")
		wordle  *bool   = pflag.Bool("wordle", false, "Presets size=5 max=5")

		mustB []byte
		canB  []byte
	)
	pflag.BoolVar(&verbose, "verbose", false, "Toggle to lose your mind with bad results")
	pflag.Parse()

	if len(*must) == 0 && len(*can) == 0 {
		pflag.PrintDefaults()
		return
	}

	if *wordle {
		*minSize = 5
		*maxSize = 5
	}

	mustB = []byte(strings.ToLower(*must))
	canB = []byte(strings.ToLower(*can))
	if len(mustB) > 0 {
		canB = append(canB, mustB...) // put musts on cans
	}

	LexHook = scour(mustB, canB)
	lex := NewLexiconFromFile(*file, *minSize, *maxSize)

	for _, s := range lex.Phrases {
		if s == nil {
			break
		}
		fmt.Println(s.Display)
	}
}
