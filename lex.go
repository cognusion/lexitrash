package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/dlclark/regexp2" // because backreferences
)

const (
	// MaxPhrase is the longest a phrase can be.
	MaxPhrase = 32
	// LetterCount is the number of letter frequency counters
	LetterCount = 27
	// LetterTotal is the index of a phrase's letter count total
	LetterTotal = 26

	// Comment is the byte representing the character to treat as a comment sigil.
	Comment = byte(35)
	// RuneA is a const representing the rune 'a'.
	RuneA = rune('a')
	// RuneZ is a const representing the rune 'z'.
	RuneZ = rune('z')
)

var (
	// Garbage is any letter repeated more than twice.
	Garbage = regexp2.MustCompile(`^.*(\S)\1{2,}.*$`, 0)
	// Garbage2 is any two-letter sequence repeated more than thrice.
	Garbage2 = regexp2.MustCompile(`^.*(\S\S)\1{3,}.*$`, 0)
	// Garbage3 is any three-letter sequence repeated more than thrice.
	Garbage3 = regexp2.MustCompile(`^.*(\S\S\S)\1{3,}.*$`, 0)
)

// Phrase is a struct to hold a collection of letters and count of those letters.
type Phrase struct {
	Display     string
	LetterCount []int
}

// NewPhrase returns an initialized reference to a Phrase.
func NewPhrase(inPhrase string) *Phrase {
	return &Phrase{inPhrase, letterFrequency(inPhrase)}
}

// letterFrequency takes a string, and returns a frequency array.
func letterFrequency(instr string) []int {
	// last cell in letter frequency list is sum of whole list
	out := make([]int, LetterCount)
	letters := "abcdefghijklmnopqrstuvwxyz"
	for i := 0; i < len(letters); i++ {
		c := strings.Count(instr, string(letters[i]))
		if c > 0 {
			out[i] += c
			out[LetterTotal] += c
		}
	}
	return out
}

// Lexicon is a collection of Phrase.
type Lexicon struct {
	Phrases []*Phrase
	Length  int
}

// Append adds a Phrase to a Lexicon.
func (l *Lexicon) Append(w *Phrase) {
	if l.Length < len(l.Phrases) {
		// we have at least one spare Phrase
		l.Phrases[l.Length] = w
	} else {
		// we are out of Phrases
		l.Phrases = append(l.Phrases, w)
		//fmt.Fprintf(os.Stderr, "[WARN] Out of phrases %d of %d\n", l.Length, len(l.Phrases))
	}
	l.Length++
}

// NewLexicon returns an initialized, empty Lexicon.
func NewLexicon() *Lexicon {
	return &Lexicon{Phrases: make([]*Phrase, 0)}
}

// NewLexiconFromFile returns a Lexicon initialized from the provided file.
func NewLexiconFromFile(phraseFile string, minPhraseLen int) *Lexicon {
	// read
	file, err := os.Open(phraseFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "unable to open file:", err)
		os.Exit(1)
	}
	defer file.Close()

	count, _ := lineCounter(file)
	file.Seek(0, 0)

	lexicon := &Lexicon{Phrases: make([]*Phrase, count)}

	scanner := bufio.NewScanner(file)

	scanChan := make(chan string)
	blockChan := make(chan struct{})
	var wg sync.WaitGroup

	go func() {
		defer close(blockChan)
		for s := range scanChan {
			lexicon.Append(NewPhrase(s))
		}
	}()

	for scanner.Scan() {
		s := scanner.Text()
		if s[0] == Comment {
			// skip comments
			continue
		}

		s = strings.Split(s, " ")[0] // discard suffix data
		if len(s) < minPhraseLen {
			// skip too-short phrases
			continue
		}
		s = strings.ToLower(s) // ensure lc

		if !verbose {
			// Remove garbage. Preserve sanity.
			wg.Add(1)
			go func(str string) {
				defer wg.Done()

				if crap, _ := Garbage.MatchString(str); crap {
					// skip garbage
					return
				}
				if crap, _ := Garbage2.MatchString(str); crap {
					// skip garbage
					return
				}
				if crap, _ := Garbage3.MatchString(str); crap {
					// skip garbage
					return
				}

				for _, b := range str {
					if b < RuneA || b > RuneZ {
						return
					}
				}
				// POST: Not so garbagey.
				scanChan <- str
			}(s)

		} else {
			scanChan <- s
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

// lineCounter is a hacky counter of return characters.
func lineCounter(r io.Reader) (int, error) {
	buf := make([]byte, 8196)
	count := 0
	lineSep := []byte{'\n'}

	for {
		c, err := r.Read(buf)
		if err != nil && err != io.EOF {
			return count, err
		}

		count += bytes.Count(buf[:c], lineSep)

		if err == io.EOF {
			break
		}
	}

	return count, nil
}
