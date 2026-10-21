// Command apitest is a minimal HTTP request tester demo built on the Yoga UI framework.
package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"strconv"
	"strings"
	"time"

	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/icons"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/theme"
	"github.com/mirzakhany/yoga/ui"
)

const (
	pendingPoll      = 30 * time.Millisecond
	splitFixedSize   = 320
	defaultURL       = "https://httpbin.org/get"
	defaultReqHeader = "Accept: */*\n"
)

type httpResult struct {
	statusCode int
	statusLine string
	headers    string
	body       []byte
	duration   time.Duration
	timing     []timingPhase
	err        error
}

// timingPhase is one step of a request, as offsets from its start.
type timingPhase struct {
	name     string
	from, to time.Duration
}

// historyEntry is a sent request, listed in the history palette.
type historyEntry struct {
	method, url string
	status      int // 0 while pending or after a transport error
	duration    time.Duration
}

const maxHistory = 20

// APITestApp is an HTTP request tester implementing yoga.App.
type APITestApp struct {
	splitDir ui.Axis
	url      string

	method   string
	bodyType string

	bodyEditor        *ui.Editor
	headersEditor     *ui.Editor
	respEditor        *ui.Editor
	respHeadersEditor *ui.Editor

	reqTabs    []ui.TabModel
	respTabs   []ui.TabModel
	reqActive  int
	respActive int

	focusedEditor *ui.Editor

	pending    bool
	resultCh   chan httpResult
	statusText string

	haveResult   bool
	respCode     int
	respStatus   string
	respDuration time.Duration
	respSize     int
	respErr      bool
	respTiming   []timingPhase

	history []historyEntry // newest first
}

var _ yoga.App = (*APITestApp)(nil)

// BuildAPITestApp assembles the API test scene.
func BuildAPITestApp() *APITestApp {
	theme.Use("yoga-midnight")
	// Thin scrollbars that only show while scrolling keep the editors calm.
	theme.UseScrollbarMode(theme.ScrollOverlay)
	app := &APITestApp{
		resultCh: make(chan httpResult, 1),
		splitDir: ui.Horizontal,
		url:      defaultURL,
		method:   "GET",
		bodyType: "json",
		reqTabs:  []ui.TabModel{{Title: "Body"}, {Title: "Headers"}},
		respTabs: []ui.TabModel{{Title: "Response"}, {Title: "Headers"}, {Title: "Timing"}},
	}

	app.bodyEditor = newEditor([]byte("{\n  \n}"), highlight.NewJSON())
	app.headersEditor = newEditor([]byte(defaultReqHeader), highlight.Noop{})
	app.respEditor = newEditor(nil, highlight.Noop{})
	app.respHeadersEditor = newEditor(nil, highlight.Noop{})

	app.focusedEditor = app.bodyEditor
	app.statusText = "Ready"
	return app
}

// newEditor builds the app's editors: long lines wrap, with wrapped rows
// hanging past their indent so nested JSON keeps its shape, and the gutter
// is only as wide as the line numbers.
func newEditor(content []byte, hl highlight.Highlighter) *ui.Editor {
	return ui.NewEditor(content, hl,
		ui.WithSoftWrap(true),
		ui.WithWrapIndent(ui.WrapHanging),
		ui.WithGutter(ui.GutterSpec{MinDigits: 2}),
	)
}

// Body derives the whole element tree from current state, registering focus,
// overlays, and animation through the frame context. The runtime calls it every
// frame; persistent state lives in the app fields.
func (app *APITestApp) Body(c *ui.Ctx) ui.View {
	th := c.Theme()
	m := c.Mouse()

	select {
	case r := <-app.resultCh:
		app.handleResult(r)
	default:
	}

	app.pickEditorFocus(m)
	app.registerCommands(c)
	if app.pending {
		c.Animate(pendingPoll)
	}

	methods := []ui.SelectOption{
		{Label: "GET", Value: "GET"},
		{Label: "POST", Value: "POST"},
		{Label: "PUT", Value: "PUT"},
		{Label: "PATCH", Value: "PATCH"},
		{Label: "DELETE", Value: "DELETE"},
	}
	splitIdx := 0
	if app.splitDir == ui.Vertical {
		splitIdx = 1
	}
	return ui.Column(
		ui.Row(
			// Method, URL and Send read as one control.
			ui.InputGroup("request",
				ui.Select("method", methods).
					FitContent().
					Selected(optionIndex(app.method, methods)).
					OptionColors(func(o ui.SelectOption) render.Color { return th.Categorical[methodHue(o.Value)] }).
					OnChange(func(v string) { app.method = v }),
				ui.TextField("url", app.url).
					Placeholder("https://api.example.com/...").
					OnChange(func(s string) { app.url = s }).
					DefaultFocus().
					Grow(1),
				ui.Button("send", ui.Text("Send")).Primary().Hint("⌘↵").OnClick(app.doRequest),
			).Grow(1),
			ui.IconButton("history", icons.Clock).Tooltip("Request history ("+app.historyChord().Label()+")").
				OnClick(func() { app.showHistory(c) }),
			ui.Segmented("split",
				ui.SegmentItem{Icon: icons.LayoutPanelLeft, Value: "h"},
				ui.SegmentItem{Icon: icons.LayoutPanelTop, Value: "v"},
			).Selected(splitIdx).OnChange(func(v string) {
				if v == "h" {
					app.setSplit(ui.Horizontal)
				} else {
					app.setSplit(ui.Vertical)
				}
			}),
		).Gap(th.Spacing.S).PaddingXY(th.Spacing.M, th.Spacing.M),
		ui.HLine(th.Stroke.Thin, th.Border),
		ui.Splitter("api-split", app.splitDir, app.reqPane(th), app.respPane(th)).
			Sizes(0, splitFixedSize).
			Grow(1),
	).Grow(1).Background(ui.TokenSurface)
}

func (app *APITestApp) reqPane(th *theme.Theme) ui.View {
	rows := []ui.View{
		ui.Tabs("req-tabs", app.reqTabs).
			Selected(app.reqActive).
			OnSelectItem(func(i int, _ string) { app.setReqTab(i) }).
			TabBackground(th.Background).
			Closable(false).
			Gap(th.Spacing.S),
	}
	if app.reqActive == 0 {
		rows = append(rows, app.bodyTypeRow(th))
	}
	rows = append(rows,
		ui.HLine(th.Stroke.Thin, th.Border).
			MarginTop(th.Spacing.S),
		ui.ViewOf(app.activeReqEditor()).Grow(1),
	)
	return ui.Column(rows...).Gap(th.Spacing.M).PaddingXY(th.Spacing.M, th.Spacing.M).Grow(1)
}

func (app *APITestApp) respPane(th *theme.Theme) ui.View {
	return ui.Column(
		app.statusLine(th),
		ui.Tabs("resp-tabs", app.respTabs).
			Selected(app.respActive).
			OnSelectItem(func(i int, _ string) { app.setRespTab(i) }).
			TabBackground(th.Background).
			Closable(false),
		ui.HLine(th.Stroke.Thin, th.Border),
		app.respBody(th),
	).Gap(th.Spacing.M).PaddingXY(th.Spacing.M, th.Spacing.M).Grow(1)
}

// respBody is the active response tab's content.
func (app *APITestApp) respBody(th *theme.Theme) ui.View {
	if app.respActive == 2 {
		return app.timingView(th)
	}
	return ui.ViewOf(app.activeRespEditor()).Grow(1)
}

// timingView draws the last request's phases as a waterfall on one scale.
func (app *APITestApp) timingView(th *theme.Theme) ui.View {
	if len(app.respTiming) == 0 {
		return ui.Column(ui.Muted("Send a request to see where its time went.")).Grow(1)
	}
	total := app.respTiming[len(app.respTiming)-1].to
	for _, p := range app.respTiming {
		total = max(total, p.to)
	}
	rows := make([]ui.View, 0, len(app.respTiming)+1)
	for i, p := range app.respTiming {
		d := p.to - p.from
		rows = append(rows, ui.Row(
			ui.Caption(p.name).Width(110),
			ui.RangeBar(fmt.Sprintf("phase-%d", i), float32(p.from), float32(p.to), float32(total)).
				Hue(phaseHue(p.name)).
				Tooltip(p.name+" "+fmtDuration(d)).
				Grow(1),
			ui.Caption(fmtDuration(d)).Width(64),
		).Gap(th.Spacing.M).Align(ui.AlignCenter))
	}
	rows = append(rows, ui.Row(
		ui.Strong("Total"), ui.Spacer(), ui.Caption(fmtDuration(total)),
	).MarginTop(th.Spacing.S))
	return ui.Scroll("timing", ui.Column(rows...).Gap(th.Spacing.S).PaddingXY(th.Spacing.S, th.Spacing.S)).Grow(1)
}

// fmtDuration shows d in milliseconds, with decimals below 10 ms.
func fmtDuration(d time.Duration) string {
	ms := float64(d) / float64(time.Millisecond)
	if ms < 10 {
		return fmt.Sprintf("%.2f ms", ms)
	}
	return fmt.Sprintf("%.0f ms", ms)
}

// phaseHue colors a timing phase: setup in blues, waiting in orange, the
// transfer in green.
func phaseHue(name string) ui.Hue {
	switch name {
	case "DNS lookup":
		return ui.HueTeal
	case "TCP connect":
		return ui.HueBlue
	case "TLS handshake":
		return ui.HuePurple
	case "Waiting (TTFB)":
		return ui.HueOrange
	}
	return ui.HueGreen
}

// methodHue is the categorical hue an HTTP method is shown in.
func methodHue(method string) ui.Hue {
	switch method {
	case "POST":
		return ui.HueBlue
	case "PUT":
		return ui.HueOrange
	case "PATCH":
		return ui.HuePurple
	case "DELETE":
		return ui.HueRed
	}
	return ui.HueGreen
}

func (app *APITestApp) historyChord() ui.Chord { return ui.MustParseChord("Mod+P") }

// registerCommands registers Send, the history palette and its entries for
// this frame.
func (app *APITestApp) registerCommands(c *ui.Ctx) {
	h := c.Commands()
	// The history list is a compact switcher; this app has no other palette.
	h.Density(ui.DensityCompact).HintFooter(true)
	h.Register(
		ui.Cmd("request.send").Title("Send request").Shortcut("Mod+Enter").Run(app.doRequest),
		ui.Cmd("request.history").Title("Request history").ShortcutChord(app.historyChord()).
			Run(func() { app.showHistory(c) }),
	)
	for i, e := range app.history {
		entry := e
		trailing := "…"
		switch {
		case entry.status > 0:
			trailing = fmt.Sprintf("%d · %v", entry.status, entry.duration.Round(time.Millisecond))
		case entry.duration > 0:
			trailing = "error"
		}
		h.Register(ui.Item(fmt.Sprintf("history.%d", i)).Scope("history").
			Title(entry.url).
			Leading(ui.Row(ui.Badge(entry.method).Hue(methodHue(entry.method)).Mono()).Width(64)).
			Trailing(trailing).
			Run(func() { app.load(entry) }).
			Alt("⌘↵", func() {
				app.load(entry)
				app.doRequest()
			}).
			AltLabel("Load and send"))
	}
}

func (app *APITestApp) showHistory(c *ui.Ctx) {
	c.Commands().ShowScope("history", "Search sent requests…")
}

// load puts a history entry's method and URL back in the request bar.
func (app *APITestApp) load(e historyEntry) {
	app.method, app.url = e.method, e.url
}

func (app *APITestApp) bodyTypeRow(th *theme.Theme) ui.View {
	opts := []ui.SelectOption{
		{Label: "None", Value: "none"},
		{Label: "Text", Value: "text"},
		{Label: "JSON", Value: "json"},
	}
	return ui.Row(
		ui.Caption("Body type"),
		ui.Select("body-type", opts).
			Width(100).
			Selected(optionIndex(app.bodyType, opts)).
			OnChange(app.setBodyType),
		ui.Spacer(),
		ui.Button("beautify", ui.Text("Beautify")).Subtle().IconStart(icons.Menu).OnClick(app.beautifyBody),
	).
		MarginLeft(th.Spacing.S).
		MarginTop(th.Spacing.S).
		Gap(th.Spacing.S).Height(th.Metrics.ControlHeight)
}

func (app *APITestApp) statusLine(th *theme.Theme) ui.View {
	if !app.haveResult {
		return ui.Row(
			ui.Text(app.statusText).Style(ui.Spec{}.TextColor(ui.TokenForegroundMuted)),
		).Height(th.Metrics.ControlHeight)
	}
	col := ui.TokenSuccess
	switch {
	case app.respErr || app.respCode >= 400:
		col = ui.TokenError
	case app.respCode >= 300:
		col = ui.TokenWarning
	}
	label := app.respStatus
	if label == "" {
		label = fmt.Sprintf("%d", app.respCode)
	}
	row := []ui.View{ui.Text(label).Style(ui.Spec{}.TextColor(col))}
	if !app.respErr {
		meta := fmt.Sprintf("%d ms  ·  %s", app.respDuration.Milliseconds(), humanSize(app.respSize))
		row = append(row, ui.Spacer(), ui.Text(meta).Style(ui.Spec{}.TextColor(ui.TokenForegroundMuted)))
	}
	return ui.Row(row...).Height(th.Metrics.ControlHeight)
}

func (app *APITestApp) activeReqEditor() *ui.Editor {
	if app.reqActive == 1 {
		return app.headersEditor
	}
	return app.bodyEditor
}

func (app *APITestApp) activeRespEditor() *ui.Editor {
	if app.respActive != 0 {
		return app.respHeadersEditor
	}
	return app.respEditor
}

// StatusText returns the latest request status line (for headless tests).
func (app *APITestApp) StatusText() string { return app.statusText }

// humanSize formats a byte count as a compact human-readable string.
func humanSize(n int) string {
	switch {
	case n < 1024:
		return fmt.Sprintf("%d B", n)
	case n < 1024*1024:
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	default:
		return fmt.Sprintf("%.1f MB", float64(n)/(1024*1024))
	}
}

func (app *APITestApp) setSplit(dir ui.Axis) {
	app.splitDir = dir
}

func (app *APITestApp) setReqTab(i int) {
	if i < 0 || i > 1 {
		return
	}
	app.reqActive = i
	app.focusedEditor = app.activeReqEditor()
}

func (app *APITestApp) setRespTab(i int) {
	if i < 0 || i > 2 {
		return
	}
	app.respActive = i
	app.focusedEditor = app.activeRespEditor()
}

func (app *APITestApp) setBodyType(value string) {
	if app.bodyType == value {
		return
	}
	app.bodyType = value
	content := app.bodyEditor.Bytes()
	wasFocused := app.focusedEditor == app.bodyEditor
	app.bodyEditor.Close()
	var hl highlight.Highlighter
	if value == "json" {
		hl = highlight.NewJSON()
	} else {
		hl = highlight.Noop{}
	}
	app.bodyEditor = newEditor(content, hl)
	if wasFocused {
		app.focusedEditor = app.bodyEditor
		app.bodyEditor.Focus()
	}
}

// beautifyBody pretty-prints the request body when it is valid JSON, replacing
// the editor content in place.
func (app *APITestApp) beautifyBody() {
	src := app.bodyEditor.Bytes()
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, src, "", "  "); err != nil {
		app.statusText = "Beautify: invalid JSON"
		return
	}
	wasFocused := app.focusedEditor == app.bodyEditor
	app.bodyEditor.Close()
	app.bodyEditor = newEditor(pretty.Bytes(), highlight.NewJSON())
	if wasFocused {
		app.focusedEditor = app.bodyEditor
		app.bodyEditor.Focus()
	}
}

func (app *APITestApp) pickEditorFocus(m *input.Mouse) {
	if m == nil || !m.Pressed {
		return
	}
	for _, ed := range []*ui.Editor{
		app.respHeadersEditor, app.respEditor, app.headersEditor, app.bodyEditor,
	} {
		if ed != nil && ed.Contains(m.X, m.Y) {
			app.focusedEditor = ed
			return
		}
	}
}

func (app *APITestApp) setResponseBody(body []byte, asJSON bool) {
	if asJSON {
		var pretty bytes.Buffer
		if err := json.Indent(&pretty, body, "", "  "); err == nil {
			body = pretty.Bytes()
		}
	}
	wasFocused := app.focusedEditor == app.respEditor
	app.respEditor.Close()
	var hl highlight.Highlighter
	if asJSON {
		hl = highlight.NewJSON()
	} else {
		hl = highlight.Noop{}
	}
	app.respEditor = newEditor(body, hl)
	if wasFocused {
		app.focusedEditor = app.respEditor
		app.respEditor.Focus()
	}
}

func (app *APITestApp) setResponseHeaders(headers string) {
	// Reflect the header count on the Headers tab badge.
	n := 0
	for _, line := range strings.Split(headers, "\n") {
		if strings.TrimSpace(line) != "" {
			n++
		}
	}
	if n > 0 {
		app.respTabs[1].Badge = strconv.Itoa(n)
	} else {
		app.respTabs[1].Badge = ""
	}

	wasFocused := app.focusedEditor == app.respHeadersEditor
	app.respHeadersEditor.Close()
	app.respHeadersEditor = newEditor([]byte(headers), highlight.Noop{})
	if wasFocused {
		app.focusedEditor = app.respHeadersEditor
		app.respHeadersEditor.Focus()
	}
}

func (app *APITestApp) doRequest() {
	if app.pending {
		return
	}
	url := strings.TrimSpace(app.url)
	if url == "" {
		app.statusText = "Enter a URL"
		return
	}
	app.pending = true
	app.statusText = "Sending..."
	app.history = append([]historyEntry{{method: app.method, url: url}}, app.history...)
	if len(app.history) > maxHistory {
		app.history = app.history[:maxHistory]
	}
	go app.executeRequest(url)
}

func (app *APITestApp) executeRequest(rawURL string) {
	start := time.Now()
	method := app.method

	headers, err := parseHeaders(string(app.headersEditor.Bytes()))
	if err != nil {
		app.resultCh <- httpResult{err: err, duration: time.Since(start)}
		return
	}

	var body io.Reader
	if methodHasBody(method) {
		bodyBytes := app.bodyEditor.Bytes()
		if len(bytes.TrimSpace(bodyBytes)) > 0 {
			body = bytes.NewReader(bodyBytes)
			if headers.Get("Content-Type") == "" {
				if app.bodyType == "json" {
					headers.Set("Content-Type", "application/json")
				} else {
					headers.Set("Content-Type", "text/plain; charset=utf-8")
				}
			}
		}
	}

	req, err := http.NewRequest(method, rawURL, body)
	if err != nil {
		app.resultCh <- httpResult{err: err, duration: time.Since(start)}
		return
	}
	req.Header = headers
	tr := &timingTrace{start: start}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), tr.trace()))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		app.resultCh <- httpResult{err: err, duration: time.Since(start)}
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		app.resultCh <- httpResult{err: err, duration: time.Since(start)}
		return
	}

	app.resultCh <- httpResult{
		statusCode: resp.StatusCode,
		statusLine: resp.Status,
		headers:    formatResponseHeaders(resp.Header),
		body:       respBody,
		duration:   time.Since(start),
		timing:     tr.phases(time.Since(start)),
	}
}

// timingTrace records when each step of a request happened. A reused
// connection skips DNS, connect and TLS, so those phases may be missing.
type timingTrace struct {
	start                time.Time
	dnsStart, dnsDone    time.Time
	connStart, connDone  time.Time
	tlsStart, tlsDone    time.Time
	wroteRequest, gotFst time.Time
}

func (t *timingTrace) trace() *httptrace.ClientTrace {
	now := func(p *time.Time) { *p = time.Now() }
	return &httptrace.ClientTrace{
		DNSStart:             func(httptrace.DNSStartInfo) { now(&t.dnsStart) },
		DNSDone:              func(httptrace.DNSDoneInfo) { now(&t.dnsDone) },
		ConnectStart:         func(string, string) { now(&t.connStart) },
		ConnectDone:          func(string, string, error) { now(&t.connDone) },
		TLSHandshakeStart:    func() { now(&t.tlsStart) },
		TLSHandshakeDone:     func(tls.ConnectionState, error) { now(&t.tlsDone) },
		WroteRequest:         func(httptrace.WroteRequestInfo) { now(&t.wroteRequest) },
		GotFirstResponseByte: func() { now(&t.gotFst) },
	}
}

// phases turns the recorded times into offsets from the start; total is
// when the body finished downloading.
func (t *timingTrace) phases(total time.Duration) []timingPhase {
	var out []timingPhase
	add := func(name string, from, to time.Time) {
		if from.IsZero() || to.IsZero() {
			return
		}
		out = append(out, timingPhase{name, from.Sub(t.start), to.Sub(t.start)})
	}
	add("DNS lookup", t.dnsStart, t.dnsDone)
	add("TCP connect", t.connStart, t.connDone)
	add("TLS handshake", t.tlsStart, t.tlsDone)
	add("Waiting (TTFB)", t.wroteRequest, t.gotFst)
	if !t.gotFst.IsZero() {
		out = append(out, timingPhase{"Download", t.gotFst.Sub(t.start), total})
	}
	return out
}

func (app *APITestApp) handleResult(r httpResult) {
	app.pending = false
	app.haveResult = true
	app.respDuration = r.duration
	app.respTiming = r.timing
	if len(app.history) > 0 {
		app.history[0].status, app.history[0].duration = r.statusCode, r.duration
	}
	if r.err != nil {
		app.respErr = true
		app.respCode = 0
		app.respStatus = "Error"
		app.respSize = 0
		app.statusText = fmt.Sprintf("Error — %s", r.duration.Round(time.Millisecond))
		app.setResponseBody([]byte(r.err.Error()), false)
		app.setResponseHeaders("")
		return
	}
	app.respErr = false
	app.respCode = r.statusCode
	app.respStatus = r.statusLine
	app.respSize = len(r.body)
	app.statusText = fmt.Sprintf("%s — %s", r.statusLine, r.duration.Round(time.Millisecond))
	app.setResponseBody(r.body, isJSONResponse(r.headers, r.body))
	app.setResponseHeaders(r.headers)
}

func methodHasBody(method string) bool {
	switch method {
	case "POST", "PUT", "PATCH":
		return true
	default:
		return false
	}
}

func parseHeaders(text string) (http.Header, error) {
	h := make(http.Header)
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			return nil, fmt.Errorf("invalid header line: %q", line)
		}
		h.Add(strings.TrimSpace(key), strings.TrimSpace(val))
	}
	return h, nil
}

func formatResponseHeaders(h http.Header) string {
	var b strings.Builder
	for key, vals := range h {
		for _, v := range vals {
			fmt.Fprintf(&b, "%s: %s\n", key, v)
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func isJSONResponse(headerText string, body []byte) bool {
	lower := strings.ToLower(headerText)
	if strings.Contains(lower, "application/json") || strings.Contains(lower, "application/ld+json") {
		return true
	}
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return false
	}
	var js json.RawMessage
	return json.Unmarshal(trimmed, &js) == nil
}

func optionIndex(v string, opts []ui.SelectOption) int {
	for i, o := range opts {
		if o.Value == v {
			return i
		}
	}
	return 0
}

func (app *APITestApp) Close() {
	app.bodyEditor.Close()
	app.headersEditor.Close()
	app.respEditor.Close()
	app.respHeadersEditor.Close()
}
