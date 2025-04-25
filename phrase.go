package main

import (
	"strings"
	"sync"
)

var (
	// ppool is a sync.Pool handing out *Phrases. In the new style, they are oft-created but seldom retained.
	ppool PhrasePool
)

func init() {
	// define the ppool
	ppool.pool = sync.Pool{
		New: func() any {
			return &Phrase{"", make([]int, LetterCount), true}
		},
	}
}

// Phrase is a struct to hold a collection of letters and count of those letters.
type Phrase struct {
	Display     string
	LetterCount []int
	zeroed      bool
}

// Set initiialized the Phrase, compulsively Resetting it first.
func (p *Phrase) Set(inPhrase string) {
	if !p.zeroed {
		p.Reset()
	}

	p.zeroed = false
	p.Display = inPhrase
	for _, l := range inPhrase {
		c := strings.Count(inPhrase, string(l))
		if c > 0 {
			p.LetterCount[aIndex[l]] += c
		}
	}
}

// Reset zeroes-out the attributes of the Phrase.
func (p *Phrase) Reset() {
	p.Display = ""
	for i := range p.LetterCount {
		p.LetterCount[i] = 0
	}
	p.zeroed = true
}

// PhrasePool contains a sync.Pool to dole out previously-created Phrases.
type PhrasePool struct {
	pool sync.Pool
}

// Get will return an existing Phrase or a new one if the pool is empty.
func (p *PhrasePool) Get() *Phrase {
	return p.pool.Get().(*Phrase)
}

// Put returns a Phrase to the pool.
func (p *PhrasePool) Put(b *Phrase) {
	p.pool.Put(b)
}

// NewPhrase returns an initialized reference to a Phrase.
func NewPhrase(inPhrase string) *Phrase {
	p := ppool.Get()
	p.Set(inPhrase)
	return p
}
