package main

import (
	"bufio"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"sync"

	"github.com/periaate/clmux"
)

var Resp = func() *resp {
	return &resp{
		val:   "Hello, World!",
		mutex: sync.Mutex{},
	}
}()

type resp struct {
	val   string
	mutex sync.Mutex
}

func (r *resp) SetVal(val string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.val = val
}
func (r *resp) GetVal() string {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return r.val
}

func GetHandler(logger *slog.Logger) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Info("Received request", "from", r.RemoteAddr)
		fmt.Fprint(w, Resp.GetVal())
	}
}

var mux *clmux.Mux

func main() {
	logs := clmux.MakeView("logs", 100)
	cmd := clmux.MakeView("cmd", 0)

	handler := GetHandler(logs.Slogger())
	startCLI(cmd)

	mux = &clmux.Mux{
		Output: os.Stdout,
		Input:  os.Stdin,

		Views: map[string]clmux.Source{},
	}

	mux.Src = cmd
	mux.Register(logs, cmd)

	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe("localhost:8080", nil))
}

func startCLI(clog clmux.Logger) {
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for {
			clog.Log("Enter command ('logs' for logs, 'exit' to exit):\n")
			fmt.Println()
			if !scanner.Scan() {
				continue
			}

			command := scanner.Text()
			switch command {
			case "set":
				clog.Log("Enter new value:\n")
				if !scanner.Scan() {
					continue
				}
				Resp.SetVal(scanner.Text())
			case "logs":
				mux.SetView("logs")
			case "exit":
				os.Exit(0)
			default:
				mux.SetView("cmd")
			}
		}
	}()
}
