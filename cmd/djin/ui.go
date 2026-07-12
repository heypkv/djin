package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"time"

	"github.com/heypkv/djin/internal/webui"
)

// heyContractVersion is the app-contract generation this server implements
// (see github.com/heypkv/hey docs/app-contract-v0.md).
const heyContractVersion = 0

type uiOpts struct {
	port   int
	json   bool
	noOpen bool
}

// cmdUI serves the embedded web UI over loopback. With --json it prints the
// hey handshake line after the listener is bound; otherwise it opens the
// browser. This is the placeholder UI server: the real Vite+React app lands
// in a later task, but the hey app contract is fully implemented here.
func cmdUI(args []string) error {
	o := uiOpts{port: 4181}
	for i := 0; i < len(args); i++ {
		a := args[i]
		next := func() (string, error) {
			if i+1 >= len(args) {
				return "", fmt.Errorf("missing value for %s", a)
			}
			i++
			return args[i], nil
		}
		var err error
		switch a {
		case "--port":
			var v string
			if v, err = next(); err == nil {
				o.port, err = strconv.Atoi(v)
			}
		case "--json":
			o.json = true
		case "--no-open":
			o.noOpen = true
		default:
			return fmt.Errorf("unknown flag %q", a)
		}
		if err != nil {
			return err
		}
	}

	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", o.port))
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	serverURL := fmt.Sprintf("http://127.0.0.1:%d", port)

	mux := http.NewServeMux()
	mux.Handle("/", webui.Handler())
	mux.HandleFunc("GET /api/version", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]string{"name": "djin", "version": version})
	})
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("POST /hey/shutdown", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		go func() {
			time.Sleep(100 * time.Millisecond)
			os.Exit(0)
		}()
	})

	if o.json {
		// hey app contract: exactly one flushed stdout line once bound.
		hs, _ := json.Marshal(map[string]any{
			"hey": 1, "name": "djin", "version": version,
			"url": serverURL, "pid": os.Getpid(), "port": port,
		})
		fmt.Println(string(hs))
	} else {
		fmt.Fprintf(os.Stderr, "djin ui at %s (Ctrl+C to stop)\n", serverURL)
		if !o.noOpen {
			openBrowser(serverURL)
		}
	}

	return http.Serve(ln, originGuard(port, mux))
}

// originGuard rejects cross-origin browser requests. The UI is same-origin, so
// any foreign Origin header means some other website is poking the local
// server — loopback is not a trust boundary (north-star principle 4).
func originGuard(port int, next http.Handler) http.Handler {
	allowed := map[string]bool{
		fmt.Sprintf("http://127.0.0.1:%d", port): true,
		fmt.Sprintf("http://localhost:%d", port): true,
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" && !allowed[origin] {
			http.Error(w, "forbidden origin", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func openBrowser(u string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", u)
	case "darwin":
		cmd = exec.Command("open", u)
	default:
		cmd = exec.Command("xdg-open", u)
	}
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "could not open browser (%v) — open %s yourself\n", err, u)
		return
	}
	go func() { _ = cmd.Wait() }()
}
