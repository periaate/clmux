package clmux

import (
	"io"
	"strings"
	"sync"
)

type Listener interface{ Broadcast(src string, msg string) }

// Mux is a io.Reader multiplexer with caching and real-time updates.
// It is effectively tmux, but for the CLI.
// E.g., multiplexing between an interactive mode, and a web servers logs.
type Mux struct {
	Output io.Writer
	Input  io.Reader

	// Views are the sources of the mux.
	Views map[string]Source

	Src   Source
	mutex sync.RWMutex
}

// Registers the sources to the mux and starts them.
func (m *Mux) Register(srcs ...Source) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	for _, src := range srcs {
		m.Views[src.Name()] = src
		src.Start(m)
	}
}

// Broadcasts the message to the mux.
func (m *Mux) Broadcast(src, msg string) {
	m.mutex.RLock()
	if m.Src.Name() != src {
		m.mutex.RUnlock()
		return
	}
	m.mutex.RUnlock()
	m.broadcast(msg)
}

func (m *Mux) broadcast(msg string) {
	msg = strings.TrimSpace(msg)
	if msg == "" || msg == "\n" {
		return
	}
	m.Output.Write([]byte(msg + "\n"))
}

func (m *Mux) SetView(name string) {
	m.mutex.Lock()
	view, ok := m.Views[name]
	if !ok {
		m.mutex.Unlock()
		return
	}

	print("\r\n\033[H\033[2J") // Clear screen

	for _, entry := range view.Cached() {
		m.broadcast(entry)
	}
	m.Src = view
	m.mutex.Unlock()
}
