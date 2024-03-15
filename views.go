package clmux

import (
	"bufio"
	"io"
	"log/slog"
	"sync"
)

const (
	// DefaultMaxEntries is the default maximum number of entries in a cache.
	DefaultMaxEntries = 256
)

type Source interface {
	// Name returns the name of the source.
	Name() string
	// Cached returns the cached entries of the source.
	Cached() []string
	// Start starts the source with the listener.
	Start(Listener)
}

type Logger interface {
	io.Writer
	Log(string) (int, error)
}

// View is a reader source.
type View struct {
	// name is the name of the view.
	name string

	Mutex sync.Mutex
	r     io.Reader
	w     io.Writer

	// Listener is the broadcaster for the view.
	Listener Listener

	maxEntries int
	entries    []string
}

// MakeView returns a new view with the arguments.
func MakeView(name string, maxEntries int) *View {
	if maxEntries < 0 {
		maxEntries = DefaultMaxEntries
	}
	r, w := io.Pipe()
	v := &View{
		name:  name,
		Mutex: sync.Mutex{},

		w: w,
		r: r,

		maxEntries: maxEntries,
		entries:    make([]string, 0, maxEntries),
	}

	return v
}

func (v *View) listen() {
	go func() {
		scanner := bufio.NewScanner(v.r)
		for scanner.Scan() {
			v.Mutex.Lock()
			text := string(scanner.Bytes())
			if text == "" {
				v.Mutex.Unlock()
				continue
			}

			v.entries = append(v.entries, text)
			if len(v.entries) > v.maxEntries {
				v.entries = v.entries[1:]
			}
			v.Listener.Broadcast(v.name, text)
			v.Mutex.Unlock()
		}
	}()
}

// Start starts the view with the listener.
func (v *View) Start(listener Listener) {
	v.Listener = listener
	v.listen()
}

// Name returns the name of the view.
func (v *View) Name() string { return v.name }

// Cached returns the cached entries of the view.
func (v *View) Cached() []string {
	v.Mutex.Lock()
	defer v.Mutex.Unlock()
	return v.entries
}

// Write writes the byte input to the views io.Writer.
func (v *View) Write(input []byte) (n int, err error) { return v.w.Write(input) }

// Log writes the string input to the views io.Writer.
func (v *View) Log(msg string) (n int, err error) { return v.w.Write([]byte(msg)) }

// Slogger returns a new slog.Logger with the views io.Writer.
func (v *View) Slogger(opts ...slog.HandlerOptions) *slog.Logger {
	options := &slog.HandlerOptions{Level: new(slog.LevelVar)}
	if len(opts) > 0 {
		options = &opts[0]
	}

	logh := slog.NewTextHandler(v.w, options)
	logger := slog.New(logh)
	return logger
}
