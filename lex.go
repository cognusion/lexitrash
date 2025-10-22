package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/dlclark/regexp2" // because backreferences
)

const (
	// MaxPhrase is the longest a phrase can be.
	MaxPhrase = 32
	// Letters is the universe of characters we count.
	Letters = "abcdefghijklmnopqrstuvwxyz"
	// LetterCount is the number of letter frequency counters (len(Letters))
	LetterCount = 26

	// Comment is the byte representing the character to treat as a comment sigil.
	Comment = byte(35)
	// RuneA is a const representing the rune 'a'.
	RuneA = rune('a')
	// RuneZ is a const representing the rune 'z'.
	RuneZ = rune('z')
)

const (
	modeStandard mode = iota // standard uses goros because of garbage
	modeLinear               // linear uses no goros so WASM doesn't choke
)

type mode int

var (
	// Garbage is any letter repeated more than twice.
	Garbage = regexp2.MustCompile(`^.*(\S)\1{2,}.*$`, 0)
	// Garbage2 is any two-letter sequence repeated more than thrice.
	Garbage2 = regexp2.MustCompile(`^.*(\S\S)\1{3,}.*$`, 0)
	// Garbage3 is any three-letter sequence repeated more than thrice.
	Garbage3 = regexp2.MustCompile(`^.*(\S\S\S)\1{3,}.*$`, 0)

	// aIndex is a map of each lower-case alphabet rune onto its corresponding index slot.
	aIndex map[rune]int
)

// LexHook is a function to filter results when building a Lexicon
var LexHook func(*Phrase) bool

func init() {
	LexHook = func(s *Phrase) bool {
		return true
	}

	// build aindex
	aIndex = make(map[rune]int)
	for i, l := range Letters {
		aIndex[l] = i
	}
}

// letterFrequency takes a string, and returns a frequency array.
func letterFrequency(instr string) []int {
	// last cell in letter frequency list is sum of whole list
	out := make([]int, LetterCount)
	for _, l := range instr {
		c := strings.Count(instr, string(l))
		if c > 0 {
			out[aIndex[l]] += c
		}
	}
	return out
}

// Lexicon is a collection of Phrase.
type Lexicon struct {
	Phrases []*Phrase
}

// Append adds a Phrase to a Lexicon.
func (l *Lexicon) Append(w *Phrase) {
	l.Phrases = append(l.Phrases, w)
}

// NewLexicon returns an initialized, empty Lexicon.
func NewLexicon() *Lexicon {
	return &Lexicon{Phrases: make([]*Phrase, 0)}
}

// NewLexiconFromBytes returns a Lexicon initialized from the provided byte slice.
func NewLexiconFromBytes(b []byte, minPhraseLen, maxPhraseLen int, m mode) *Lexicon {
	if maxPhraseLen < 1 {
		// sanity
		maxPhraseLen = 999
	}

	scanner := bufio.NewScanner(bytes.NewBuffer(b))

	switch m {
	case modeLinear:
		return newLexiconLinear(scanner, minPhraseLen, maxPhraseLen, false)
	case modeStandard:
		fallthrough
	default:
		return newLexicon(scanner, minPhraseLen, maxPhraseLen, false) // assume bytes are clean
	}
}

// NewLexiconFromFile returns a Lexicon initialized from the provided file.
func NewLexiconFromFile(phraseFile string, minPhraseLen, maxPhraseLen int) *Lexicon {
	if maxPhraseLen < 1 {
		// sanity
		maxPhraseLen = 999
	}

	// read
	file, err := os.Open(path.Clean(phraseFile))
	if err != nil {
		fmt.Fprintln(os.Stderr, "unable to open file:", err)
		os.Exit(1)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	return newLexicon(scanner, minPhraseLen, maxPhraseLen, true) // assume file is dirty
}

func newLexicon(scanner *bufio.Scanner, minPhraseLen, maxPhraseLen int, dirty bool) *Lexicon {
	lexicon := NewLexicon()

	scanChan := make(chan *Phrase)
	blockChan := make(chan struct{})
	var wg sync.WaitGroup

	go func() {
		defer close(blockChan)
		for s := range scanChan {
			lexicon.Append(s)
		}
	}()

	for scanner.Scan() {
		s := scanner.Text()
		if s[0] == Comment {
			// skip comments
			continue
		}

		s = strings.Split(s, " ")[0] // discard suffix data
		if len(s) < minPhraseLen || len(s) > maxPhraseLen {
			// skip too-short/too-long phrases
			continue
		}
		s = strings.ToLower(s) // ensure lc

		if !verbose {
			// Remove garbage. Preserve sanity.
			wg.Add(1)
			go func(str string) {
				defer wg.Done()

				if dirty && isGarbage(str) {
					return
				}
				// POST: Not so garbagey.

				p := ppool.Get()
				p.Set(str)
				if LexHook(p) {
					scanChan <- p
				} else {
					ppool.Put(p)
				}
			}(s)
		} else {
			// verbose

			p := ppool.Get()
			p.Set(s)
			if LexHook(p) {
				scanChan <- p
			} else {
				ppool.Put(p)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "error reading standard input:", err)
	}

	wg.Wait() // Wait for the goros to finish
	close(scanChan)

	<-blockChan // wait for the output
	return lexicon
}

func newLexiconLinear(scanner *bufio.Scanner, minPhraseLen, maxPhraseLen int, dirty bool) *Lexicon {
	lexicon := NewLexicon()

	for scanner.Scan() {
		s := scanner.Text()
		if s[0] == Comment {
			// skip comments
			continue
		}

		s = strings.Split(s, " ")[0] // discard suffix data
		if len(s) < minPhraseLen || len(s) > maxPhraseLen {
			// skip too-short/too-long phrases
			continue
		}
		s = strings.ToLower(s) // ensure lc

		if !verbose {
			// Remove garbage. Preserve sanity.
			if dirty && isGarbage(s) {
				continue
			}
			// POST: Not so garbagey.

			p := ppool.Get()
			p.Set(s)
			if LexHook(p) {
				lexicon.Append(p)
			} else {
				ppool.Put(p)
			}
		} else {
			// verbose

			p := ppool.Get()
			p.Set(s)
			if LexHook(p) {
				lexicon.Append(p)
			} else {
				ppool.Put(p)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "error reading standard input:", err)
	}

	return lexicon
}

func printCleanFile(fileName string) {
	// read
	file, err := os.Open(path.Clean(fileName))
	if err != nil {
		fmt.Fprintln(os.Stderr, "unable to open file:", err)
		os.Exit(1)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		s := scanner.Text()
		if s[0] == Comment {
			// skip comments
			continue
		}
		s = strings.ToLower(strings.Split(s, " ")[0]) // discard suffix data
		if isGarbage(s) {
			continue
		}
		fmt.Println(s)
	}
}

func isGarbage(str string) bool {
	if crap, _ := Garbage.MatchString(str); crap {
		// skip garbage
		return true
	}
	if crap, _ := Garbage2.MatchString(str); crap {
		// skip garbage
		return true
	}
	if crap, _ := Garbage3.MatchString(str); crap {
		// skip garbage
		return true
	}
	for _, b := range str {
		if b < RuneA || b > RuneZ {
			return true
		}
	}
	return false
}
