package main

import (
	_ "embed"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/cognusion/go-lines"
	"github.com/cognusion/go-memoryguard"
	"github.com/spf13/pflag"
	"golang.org/x/term"
)

//go:embed en_full.txt
var textData []byte

var (
	verbose bool
	file    string
	must    string
	may     string
	minSize int
	maxSize int
	wordle  bool
	clean   bool
	csv     bool
	debug   bool
	linemax int

	runlock  sync.Mutex
	debugOut = log.New(io.Discard, "", 0)
)

func main() {
	pflag.StringVar(&file, "file", "", "Use a different dictionary source")
	pflag.StringVar(&must, "must", "", "List of letters that MUST be in the output")
	pflag.StringVar(&may, "may", "", "List of NON-MUST letters that may also be in the output")
	pflag.IntVar(&minSize, "size", 6, "Minimum length a word must be to be output")
	pflag.IntVar(&maxSize, "max", 0, "Maximum length a word can be to be output")
	pflag.BoolVar(&wordle, "wordle", false, "Presets size=5 max=5")
	pflag.BoolVar(&verbose, "verbose", false, "Toggle to lose your mind with bad results")
	pflag.BoolVar(&csv, "csv", false, "Output in csv format (CLI only)")
	pflag.BoolVar(&debug, "debug", false, "Enable debug output to stderr")
	pflag.IntVar(&linemax, "linemax", 100, "Sets the maximum number of lines outputted in GUI or --csv modes. Zero disables.")

	pflag.BoolVar(&clean, "clean", false, "Only clean the input")
	pflag.CommandLine.MarkHidden("clean")
	pflag.Parse()

	if wordle {
		minSize = 5
		maxSize = 5
	}

	if debug {
		debugOut = log.New(os.Stderr, "", 0)
	}

	if clean {
		printCleanFile(file)
		return
	}

	if len(must) == 0 && len(may) == 0 {
		gmain()
		return
	}

	pChan := phraseMe()

	if csv {
		width := 128

		fd := int(os.Stdin.Fd())
		if term.IsTerminal(fd) {
			tw, _, err := term.GetSize(fd)
			if err == nil {
				//we don't care about errors
				width = tw - 1
			}
		}

		lines.LinifyStreamSeparatorLineMax(pChan, os.Stdout, width, linemax, ",")
	} else {
		for l := range pChan {
			fmt.Println(l)
		}
	}
}

func phraseMe() <-chan string {

	pChan := make(chan string)

	go func() {
		defer close(pChan)
		var (
			mustB []byte
			canB  []byte
			lex   *Lexicon
		)

		mustB = []byte(strings.ToLower(must))
		canB = []byte(strings.ToLower(may))
		if len(mustB) > 0 {
			canB = append(canB, mustB...) // put musts on cans
		}

		LexHook = scour(mustB, canB)
		if file != "" {
			// File
			lex = NewLexiconFromFile(file, minSize, maxSize)
		} else {
			// bytes
			lex = NewLexiconFromBytes(textData, minSize, maxSize, modeLinear)
		}

		for _, s := range lex.Phrases {
			if s == nil {
				break
			}
			pChan <- s.Display
		}
	}()

	return pChan
}

func gmain() {
	// Get a handle on our process
	us, _ := os.FindProcess(os.Getpid())

	// Create a new MemoryGuard around the process
	mg := memoryguard.New(us)
	mg.Limit(512 * 1024 * 1024)
	defer mg.Cancel()

	if debug {
		k := make(chan struct{})
		defer close(k)
		go func() {
			for {
				select {
				case <-k:
					return
				case <-time.After(10 * time.Second):
					debugOut.Printf("PSS: %d\n", mg.PSS())
				}
			}
		}()
	}

	a := app.New()
	loadTheme(a)

	g := newGUI()
	w := g.makeWindow(a)

	g.win.SetTitle("Lexitrash")

	g.setupActions()
	w.ShowAndRun()
}

// here you can add some button / callbacks code using widget IDs
func (g *gui) setupActions() {
	g.win.Canvas().Focus(g.mustText)

	// hit the button!
	g.mustText.OnSubmitted = func(c string) {
		g.trashTap()
	}

	g.mayText.OnSubmitted = func(c string) {
		g.trashTap()
	}
}

func (g *gui) trashTap() {
	if !runlock.TryLock() {
		// Don't need more than one running ever.
		return
	}
	defer runlock.Unlock()

	g.trashButton.Disable()
	defer g.trashButton.Enable()

	must = g.mustText.Text
	may = g.mayText.Text
	textBuffer := &strings.Builder{}

	pChan := phraseMe() // get the phrases!

	// calculate how wide our lines should be
	w := g.win.Canvas().Size().Width
	fs := fyne.MeasureText("a", theme.TextSize(), fyne.TextStyle{})
	charWidth := int(w / fs.Width)

	// Read from the chan and write into our buffer
	// NOTE: We are only allowing 100 lines here. Tops.
	// We can overflow RAM with this puppy.
	// No joke.
	// Use the CLI if you need zillions of results.
	//
	// Also, the width is padded, so we take 10 chars off the width for that. Could be wrong,
	// but I can't figure out how to detect the size of padding in Fyne.
	lines.LinifyStreamSeparatorLineMax(pChan, textBuffer, charWidth-3, linemax, ",")

	// show the results
	scr := widget.NewEntry()
	scr.Text = textBuffer.String()

	d := dialog.NewCustom("Results!", "Ok!", scr, g.win)
	d.Resize(g.win.Canvas().Size())
	d.Show()
}

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
