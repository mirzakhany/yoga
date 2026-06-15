package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"testing"
	"time"
)

// fakeServer is a minimal LSP server over an io.ReadWriteCloser, used to drive
// the client in tests without a real language-server binary.
type fakeServer struct {
	rwc      io.ReadWriteCloser
	r        *bufio.Reader
	encoding string // positionEncoding to advertise in the initialize result

	completion []CompletionItem
	hover      string
}

func newFakeServer(rwc io.ReadWriteCloser, encoding string) *fakeServer {
	return &fakeServer{rwc: rwc, r: bufio.NewReader(rwc), encoding: encoding}
}

func (s *fakeServer) write(v any) {
	data, _ := json.Marshal(v)
	fmt.Fprintf(s.rwc, "Content-Length: %d\r\n\r\n", len(data))
	s.rwc.Write(data)
}

func (s *fakeServer) reply(id *json.RawMessage, result any) {
	s.write(map[string]any{"jsonrpc": "2.0", "id": json.RawMessage(*id), "result": result})
}

// notify pushes a server→client notification (e.g. publishDiagnostics).
func (s *fakeServer) notify(method string, params any) {
	s.write(map[string]any{"jsonrpc": "2.0", "method": method, "params": params})
}

// serve runs the request/response loop until the connection closes.
func (s *fakeServer) serve() {
	for {
		n, err := readContentLength(s.r)
		if err != nil {
			return
		}
		body := make([]byte, n)
		if _, err := io.ReadFull(s.r, body); err != nil {
			return
		}
		var msg rpcMessage
		if json.Unmarshal(body, &msg) != nil {
			continue
		}
		switch msg.Method {
		case methodInitialize:
			s.reply(msg.ID, initializeResult{Capabilities: serverCapabilities{PositionEncoding: s.encoding}})
		case methodCompletion:
			s.reply(msg.ID, completionList{Items: s.completion})
		case methodHover:
			s.reply(msg.ID, Hover{Contents: mustJSON(MarkupContent{Kind: "plaintext", Value: s.hover})})
		case methodShutdown:
			s.reply(msg.ID, nil)
		}
	}
}

func mustJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

// newTestClient wires a client to a fakeServer over an in-memory pipe.
func newTestClient(t *testing.T, srv *fakeServer) (*client, net.Conn) {
	t.Helper()
	cConn, sConn := net.Pipe()
	srv.rwc = sConn
	srv.r = bufio.NewReader(sConn)
	go srv.serve()

	c, err := newClient(cConn, "file:///workspace")
	if err != nil {
		t.Fatalf("newClient: %v", err)
	}
	return c, sConn
}

func TestFramingRoundTrip(t *testing.T) {
	a, b := net.Pipe()
	defer a.Close()
	defer b.Close()

	go func() {
		w := bufio.NewWriter(a)
		data := []byte(`{"jsonrpc":"2.0","method":"ping"}`)
		fmt.Fprintf(w, "Content-Length: %d\r\n\r\n", len(data))
		w.Write(data)
		w.Flush()
	}()

	r := bufio.NewReader(b)
	n, err := readContentLength(r)
	if err != nil {
		t.Fatalf("readContentLength: %v", err)
	}
	body := make([]byte, n)
	if _, err := io.ReadFull(r, body); err != nil {
		t.Fatalf("ReadFull: %v", err)
	}
	var msg rpcMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if msg.Method != "ping" {
		t.Fatalf("method = %q, want ping", msg.Method)
	}
}

func TestInitializeNegotiatesEncoding(t *testing.T) {
	srv := newFakeServer(nil, "utf-8")
	c, _ := newTestClient(t, srv)
	defer c.rpc.Close()

	if c.encoding != "utf-8" {
		t.Fatalf("encoding = %q, want utf-8", c.encoding)
	}
}

func TestCompletionResolves(t *testing.T) {
	srv := newFakeServer(nil, "utf-8")
	srv.completion = []CompletionItem{
		{Label: "Println", InsertText: "Println"},
		{Label: "Printf"},
	}
	c, _ := newTestClient(t, srv)
	defer c.rpc.Close()

	items, err := c.completion("file:///workspace/main.go", Position{Line: 0, Character: 0})
	if err != nil {
		t.Fatalf("completion: %v", err)
	}
	if len(items) != 2 || items[0].Label != "Println" {
		t.Fatalf("items = %+v", items)
	}
	if got := items[1].Insert(); got != "Printf" {
		t.Fatalf("Insert() fallback = %q, want Printf", got)
	}
}

func TestHoverResolves(t *testing.T) {
	srv := newFakeServer(nil, "utf-8")
	srv.hover = "func fmt.Println(a ...any) (n int, err error)"
	c, _ := newTestClient(t, srv)
	defer c.rpc.Close()

	h, err := c.hover("file:///workspace/main.go", Position{Line: 1, Character: 2})
	if err != nil {
		t.Fatalf("hover: %v", err)
	}
	if h.Text() != srv.hover {
		t.Fatalf("hover text = %q", h.Text())
	}
}

func TestPublishDiagnosticsSurfaces(t *testing.T) {
	srv := newFakeServer(nil, "utf-8")
	c, _ := newTestClient(t, srv)
	defer c.rpc.Close()

	uri := "file:///workspace/main.go"
	srv.notify(methodPublishDiagnostics, PublishDiagnosticsParams{
		URI: uri,
		Diagnostics: []Diagnostic{
			{Message: "undeclared name: foo", Severity: SeverityError,
				Range: Range{Start: Position{Line: 3, Character: 4}, End: Position{Line: 3, Character: 7}}},
		},
	})

	// The notification crosses the read goroutine; poll briefly for it.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(c.diagnostics(uri)) > 0 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	diags := c.diagnostics(uri)
	if len(diags) != 1 || diags[0].Severity != SeverityError {
		t.Fatalf("diagnostics = %+v", diags)
	}
	evs := c.drain(uri)
	if len(evs) != 1 || evs[0].Kind != EventDiagnostics {
		t.Fatalf("events = %+v", evs)
	}
	if rest := c.drain("file:///other.go"); rest != nil {
		t.Fatalf("drain leaked events for another uri: %+v", rest)
	}
}
