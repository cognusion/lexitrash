package main

import (
	"fmt"
	"os"
	"testing"
	"time"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/dialog"
	"github.com/cognusion/go-humanity"
	"github.com/cognusion/go-memoryguard"
)

var (
	testFile = "en_full.txt"
	testSize = 6
)

func Benchmark_NewLexiconBytes(b *testing.B) {
	LexHook = func(s *Phrase) bool {
		return false
	}

	verbose = false
	for b.Loop() {
		_ = NewLexiconFromBytes(textData, testSize, 0, modeStandard)
	}
}

func Benchmark_NewLexiconBytesLinear(b *testing.B) {
	LexHook = func(s *Phrase) bool {
		return false
	}

	verbose = false
	for b.Loop() {
		_ = NewLexiconFromBytes(textData, testSize, 0, modeLinear)
	}
}

func Benchmark_NewLexicon(b *testing.B) {
	LexHook = func(s *Phrase) bool {
		return false
	}

	verbose = false
	for b.Loop() {
		_ = NewLexiconFromFile(testFile, testSize, 0)
	}
}

func Benchmark_NewLexiconVerbose(b *testing.B) {
	LexHook = func(s *Phrase) bool {
		return false
	}

	verbose = true
	for b.Loop() {
		_ = NewLexiconFromFile(testFile, testSize, 0)
	}
}

func Test_Dialogtext(t *testing.T) {
	t.Skip("I believe there is a defect in dialog creation. WIP.")

	us, _ := os.FindProcess(os.Getpid())
	mg := memoryguard.New(us)
	mg.Limit(1024 * 1024 * 1024)
	defer mg.Cancel()

	a := app.New()
	g := newGUI()
	w := g.makeWindow(a)
	sz := 100000

	var (
		s  string
		td time.Duration
	)

	for range sz {
		s += "this is some more text"
	}

	start := time.Now()
	d := dialog.NewInformation("", s, w)
	//l := &widget.Label{Text: s, Alignment: fyne.TextAlignCenter, Wrapping: fyne.TextWrapWord}
	td = time.Since(start)
	fmt.Printf("%d\t%s\tPSS: %s\n", len(s), td.String(), humanity.ByteFormat(mg.PSS()))
	//io.Discard.Write([]byte(l.Text))
	d.Dismiss()

	// writing that string to a widget.Label takes 200ns and ~200MB RAM.
	// writing that string to a dialog.NewInformation takes 2s and ~600MB RAM

}
