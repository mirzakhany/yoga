package lsp

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// The manager tests run this test binary as the language server: with
// LSP_FAKE_SERVER set, TestMain serves LSP over stdin/stdout instead of running
// tests. The variable's value picks a behavior:
//
//	ok     answer requests until stdin closes
//	crash  write to stderr and exit 3 right after the handshake
//	early  write to stderr and exit 5 before answering initialize
//	config ask the client for workspace/configuration and exit 0 if the reply
//	       is an array with one entry per item, 4 otherwise
func TestMain(m *testing.M) {
	if mode := os.Getenv("LSP_FAKE_SERVER"); mode != "" {
		os.Exit(runFakeProcess(mode))
	}
	os.Exit(m.Run())
}

type stdioRWC struct {
	io.Reader
	io.Writer
}

func (stdioRWC) Close() error { return nil }

func runFakeProcess(mode string) int {
	srv := newFakeServer(stdioRWC{os.Stdin, os.Stdout}, "utf-8")
	for {
		n, err := readContentLength(srv.r)
		if err != nil {
			return 0
		}
		body := make([]byte, n)
		if _, err := io.ReadFull(srv.r, body); err != nil {
			return 0
		}
		var msg rpcMessage
		if json.Unmarshal(body, &msg) != nil {
			continue
		}
		switch {
		case msg.Method == methodInitialize && mode == "early":
			fmt.Fprintln(os.Stderr, "fatal: cannot load typeshed")
			return 5
		case msg.Method == methodInitialize:
			srv.reply(msg.ID, initializeResult{Capabilities: serverCapabilities{PositionEncoding: "utf-8"}})
		case msg.Method == methodInitialized && mode == "crash":
			fmt.Fprintln(os.Stderr, "fatal: fake server crashed")
			return 3
		case msg.Method == methodInitialized && mode == "config":
			srv.write(map[string]any{"jsonrpc": "2.0", "id": 99, "method": "workspace/configuration",
				"params": map[string]any{"items": []any{map[string]any{"section": "a"}, map[string]any{"section": "b"}}}})
		case msg.Method == "" && mode == "config":
			var res []json.RawMessage
			if json.Unmarshal(msg.Result, &res) == nil && len(res) == 2 {
				return 0
			}
			return 4
		case msg.Method == methodShutdown:
			srv.reply(msg.ID, nil)
		case msg.Method == methodExit:
			return 0
		}
	}
}

// errSink collects ServerErrors reported from background goroutines.
type errSink struct {
	mu   sync.Mutex
	errs []*ServerError
}

func (s *errSink) add(e *ServerError) {
	s.mu.Lock()
	s.errs = append(s.errs, e)
	s.mu.Unlock()
}

func (s *errSink) wait(t *testing.T, kind ErrorKind) *ServerError {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		s.mu.Lock()
		for _, e := range s.errs {
			if e.Kind == kind {
				s.mu.Unlock()
				return e
			}
		}
		s.mu.Unlock()
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("no %v error reported; got %v", kind, s.all())
	return nil
}

func (s *errSink) all() []*ServerError {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]*ServerError(nil), s.errs...)
}

// registerFake maps ext to this test binary running in the given mode.
func registerFake(t *testing.T, ext, lang, mode string) {
	t.Helper()
	t.Setenv("LSP_FAKE_SERVER", mode)
	Register(ext, ServerConfig{LanguageID: lang, Command: os.Args[0]})
	t.Cleanup(func() { Unregister(ext) })
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func content(s string) func() []byte { return func() []byte { return []byte(s) } }

func TestManagerReportsMissingServerOnce(t *testing.T) {
	Register(".nope", ServerConfig{LanguageID: "nope", Command: "yoga-no-such-language-server"})
	t.Cleanup(func() { Unregister(".nope") })
	m := NewManager()
	var sink errSink
	m.OnError(sink.add)

	a := m.Open(filepath.Join(t.TempDir(), "a.nope"), content(""))
	b := m.Open(filepath.Join(t.TempDir(), "b.nope"), content(""))
	defer a.Close()
	defer b.Close()

	if errs := sink.all(); len(errs) != 1 || errs[0].Kind != ErrNotFound {
		t.Fatalf("errors = %v, want one ErrNotFound", errs)
	}
	m.Restart("nope")
	if errs := sink.all(); len(errs) != 2 {
		t.Fatalf("Restart should report the still-missing server again; errors = %v", errs)
	}
}

func TestManagerReportsCrashWithStderr(t *testing.T) {
	registerFake(t, ".crash", "crash", "crash")
	m := NewManager()
	var sink errSink
	m.OnError(sink.add)

	d := m.Open(filepath.Join(t.TempDir(), "x.crash"), content("x"))
	defer d.Close()

	e := sink.wait(t, ErrExited)
	if !strings.Contains(e.Stderr, "fake server crashed") {
		t.Fatalf("stderr = %q, want the server's output", e.Stderr)
	}
	if !strings.Contains(e.Err.Error(), "exit status 3") {
		t.Fatalf("err = %v, want exit status 3", e.Err)
	}
}

func TestManagerReportsFailedStartWithExitStatus(t *testing.T) {
	registerFake(t, ".early", "early", "early")
	m := NewManager()
	var sink errSink
	m.OnError(sink.add)

	d := m.Open(filepath.Join(t.TempDir(), "x.early"), content("x"))
	defer d.Close()

	e := sink.wait(t, ErrStart)
	if !strings.Contains(e.Err.Error(), "exit status 5") || !strings.Contains(e.Stderr, "typeshed") {
		t.Fatalf("err = %v, stderr = %q; want exit status 5 and the server's output", e.Err, e.Stderr)
	}
}

func TestManagerAnswersWorkspaceConfiguration(t *testing.T) {
	registerFake(t, ".cfg", "cfg", "config")
	m := NewManager()
	var sink errSink
	m.OnError(sink.add)

	d := m.Open(filepath.Join(t.TempDir(), "x.cfg"), content(""))
	defer d.Close()

	// The fake exits 0 on a well-formed reply and 4 otherwise.
	e := sink.wait(t, ErrExited)
	if !strings.Contains(e.Err.Error(), "exit status 0") {
		t.Fatalf("configuration reply rejected: %v", e.Err)
	}
}

func TestManagerRestartAttachesAndDetaches(t *testing.T) {
	registerFake(t, ".rst", "rst", "ok")
	m := NewManager()
	var sink errSink
	m.OnError(sink.add)

	d := m.Open(filepath.Join(t.TempDir(), "x.rst"), content("hello")).(*doc)
	defer d.Close()
	first := d.session()
	if first == nil {
		t.Fatal("document did not attach")
	}
	waitFor(t, "handshake", func() bool { return first.getClient() != nil })

	m.Restart("rst")
	second := d.session()
	if second == nil || second == first {
		t.Fatalf("Restart did not start a new server: first=%p second=%p", first, second)
	}
	if first.alive() {
		t.Fatal("old server still marked alive after Restart")
	}
	if evs, _ := d.Poll(); len(evs) == 0 || evs[0].Kind != EventDiagnostics {
		t.Fatalf("Poll after Restart = %+v, want a diagnostics refresh", evs)
	}
	waitFor(t, "new handshake", func() bool { return second.getClient() != nil })

	// Disabling the language detaches the document.
	Unregister(".rst")
	m.Restart("rst")
	if d.session() != nil {
		t.Fatal("document still attached after its server was unregistered")
	}
	if got := m.Running(); len(got) != 0 {
		t.Fatalf("Running = %v, want none", got)
	}
	for _, e := range sink.all() {
		if !errors.Is(e, io.ErrClosedPipe) {
			t.Errorf("unexpected error during intentional restarts: %v", e)
		}
	}
}
