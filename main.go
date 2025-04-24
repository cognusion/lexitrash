package main

import (
	"fmt"

	"github.com/spf13/pflag"
)

var (
	disableSanity bool
)

func scour(lex *Lexicon, must []byte, can []byte) <-chan (string) {
	sChan := make(chan string)

	musti := l2i(must)
	cani := letterFrequency(string(can))

	go func() {
		for _, w := range lex.Phrases {
			if w == nil {
				continue
			}

			ok := true
			//fmt.Println(w.Display)
			for _, i := range musti {
				if w.LetterCount[i] == 0 {
					ok = false
					break
				}
			}
			if !ok {
				//fmt.Println("\tNope on musts")
				continue
			}
			// POST: we have the musts

			for i, c := range cani {
				//fmt.Printf("\t\t%d == %d. %d >=0?\n", i, c, w.LetterCount[i])
				if c == 0 && w.LetterCount[i] > 0 {
					//fmt.Println("\tNope on cants")
					ok = false
					break
				}
			}
			if !ok {
				continue
			}
			// POST: We don't have the can'ts
			//fmt.Println("\tYES!")

			sChan <- w.Display
		}
		close(sChan)
	}()
	return sChan
}

func l2i(ls []byte) []int {
	out := make([]int, len(ls))
	letters := "abcdefghijklmnopqrstuvwxyz"

	for i := range ls {
		l := rune(ls[i])
		for a, c := range letters {
			if l == c {
				out[i] = a
				break
			}
		}
	}
	return out
}

func main() {
	var file *string = pflag.String("file", "./en_full.txt", "Use a different dictionary source")
	var must *string = pflag.String("must", "", "List of letters that MUST be in the output")
	var can *string = pflag.String("can", "", "List of NON-MUST letters that may also be in the output")
	var minSize *int = pflag.Int("size", 6, "Minimum length a word must be to be output")
	pflag.BoolVar(&disableSanity, "verbose", false, "Toggle to lose your mind with bad results")
	pflag.Parse()

	if len(*must) == 0 && len(*can) == 0 {
		pflag.PrintDefaults()
		return
	}

	mustB := []byte(*must)
	canB := []byte(*can)
	canB = append(canB, mustB...) // put musts on cans
	lex := NewLexiconFromFile(*file, *minSize)

	sChan := scour(lex, mustB, canB)
	for s := range sChan {
		fmt.Println(s)
	}

}
