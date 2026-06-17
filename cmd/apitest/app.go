// Command apitest is a minimal HTTP request tester demo built on the Yoga UI framework.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mirzakhany/yoga"
	"github.com/mirzakhany/yoga/components"
	"github.com/mirzakhany/yoga/highlight"
	"github.com/mirzakhany/yoga/input"
	"github.com/mirzakhany/yoga/layout"
	"github.com/mirzakhany/yoga/render"
	"github.com/mirzakhany/yoga/shape"
	"github.com/mirzakhany/yoga/theme"
)

const (
	textFieldBlink   = 500 * time.Millisecond
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
	err        error
}

// APITestApp is an HTTP request tester implementing yoga.Scene.
type APITestApp struct {
	root            *layout.Element
	toolbar         *layout.Element
	splitHost       *layout.Element
	reqPane         *layout.Element
	respPane        *layout.Element
	reqHost         *layout.Element
	respHost        *layout.Element
	reqEditorPanel  *layout.Element
	respEditorPanel *layout.Element
	bodyTypeRow     *layout.Element
	statusBar       *layout.Element
	splitDir        components.Axis

	method      *components.Select
	url         *components.TextField
	splitToggle *components.Segmented
	send        *components.Button
	beautify    *components.Button
	bodyType    *components.Select

	bodyEditor        *components.Editor
	headersEditor     *components.Editor
	respEditor        *components.Editor
	respHeadersEditor *components.Editor

	reqTabs  *components.TabBar
	respTabs *components.TabBar
	splitter *components.Splitter
	focus    *components.FocusManager

	focusedEditor *components.Editor

	pending    bool
	resultCh   chan httpResult
	statusText string

	// Last-response summary for the status bar.
	haveResult   bool
	respCode     int
	respStatus   string
	respDuration time.Duration
	respSize     int
	respErr      bool

	lastW float32
	lastH float32
}

var _ yoga.Scene = (*APITestApp)(nil)

// BuildAPITestApp assembles the API test scene.
func BuildAPITestApp() *APITestApp {
	theme.Use("yoga-midnight")
	th := theme.Current()
	app := &APITestApp{
		resultCh: make(chan httpResult, 1),
		splitDir: components.Horizontal,
	}

	app.method = components.NewSelect(100, []components.SelectOption{
		{Label: "GET", Value: "GET"},
		{Label: "POST", Value: "POST"},
		{Label: "PUT", Value: "PUT"},
		{Label: "PATCH", Value: "PATCH"},
		{Label: "DELETE", Value: "DELETE"},
	}).
		OptionColor("GET", th.Success).
		OptionColor("POST", th.Accent).
		OptionColor("PUT", th.Warning).
		OptionColor("PATCH", render.RGBA8(187, 154, 247, 255)).
		OptionColor("DELETE", th.Error)

	app.url = components.NewTextField(components.TextFieldConfig{
		Placeholder: "https://api.example.com/...",
		Height:      th.Metrics.ControlHeight,
	}).WithIconStart("terminal")
	app.url.Value = defaultURL
	app.url.El.Style = app.url.El.Style.FlexGrow(1)

	app.splitToggle = components.NewSegmented([]components.SegmentItem{
		{Icon: "split_horizontal", Value: "h"},
		{Icon: "split_vertical", Value: "v"},
	}).ChangedValue(func(v string) {
		if v == "h" {
			app.setSplit(components.Horizontal)
		} else {
			app.setSplit(components.Vertical)
		}
	})

	app.send = components.NewButton("Send").Primary().Hint("⌘↵").Action(app.doRequest)

	app.toolbar = layout.HStack(th.Spacing.S,
		app.method.El,
		app.url.El,
		app.splitToggle.El,
		app.send.El,
	)

	app.bodyType = components.NewSelect(100, []components.SelectOption{
		{Label: "Text", Value: "text"},
		{Label: "JSON", Value: "json"},
	}).Changed(app.setBodyType)

	bodyTypeLabel := components.NewLabel("Body type", components.LabelCaption)
	app.beautify = components.NewButton("Beautify").Subtle().IconStart("menu").Action(app.beautifyBody)
	app.bodyTypeRow = layout.HStack(th.Spacing.S,
		bodyTypeLabel.El,
		app.bodyType.El,
		layout.Spacer(),
		app.beautify.El,
	)

	app.reqTabs = components.NewTabBar()
	app.reqTabs.Tabs = []components.TabModel{
		{Title: "Body"},
		{Title: "Headers"},
	}
	app.reqTabs.OnActivate = app.setReqTab

	app.respTabs = components.NewTabBar()
	app.respTabs.Tabs = []components.TabModel{
		{Title: "Response"},
		{Title: "Headers"},
	}
	app.respTabs.OnActivate = app.setRespTab

	app.bodyEditor = components.NewEditor([]byte("{\n  \n}"), highlight.NewJSON())
	app.headersEditor = components.NewEditor([]byte(defaultReqHeader), highlight.Noop{})
	app.respEditor = components.NewEditor(nil, highlight.Noop{})
	app.respHeadersEditor = components.NewEditor(nil, highlight.Noop{})

	app.reqHost = layout.New(layout.Box().FlexGrow(1))
	app.reqEditorPanel = newEditorPanel(app.reqHost, th)

	app.respHost = layout.New(layout.Box().FlexGrow(1))
	app.respEditorPanel = newEditorPanel(app.respHost, th)

	app.reqPane = layout.New(layout.Box().Direction(layout.Column).FlexGrow(1).Gap(th.Spacing.XS),
		app.reqTabs.El,
		app.bodyTypeRow,
		app.reqEditorPanel,
	)
	app.reqHost.Children = []*layout.Element{app.bodyEditor.El}

	app.statusBar = layout.New(layout.Box().H(24))
	app.statusBar.Paint = app.paintStatus

	app.respPane = layout.New(layout.Box().Direction(layout.Column).FlexGrow(1).Gap(th.Spacing.XS),
		app.statusBar,
		app.respTabs.El,
		app.respEditorPanel,
	)
	app.respHost.Children = []*layout.Element{app.respEditor.El}

	app.splitHost = layout.New(layout.Box().FlexGrow(1))
	app.buildSplitter()

	app.root = layout.New(layout.Box().Direction(layout.Column).FlexGrow(1).Gap(th.Spacing.S).PaddingAll(th.Spacing.M),
		app.toolbar,
		app.splitHost,
		app.method.MenuEl(),
		app.bodyType.MenuEl(),
	).WithBackgroundPtr(&th.Background)

	app.focus = components.NewFocusManager()
	app.focusedEditor = app.bodyEditor
	app.focus.Add(app.url, app.method, app.bodyType, app.reqTabs, app.respTabs, app.send, editorFocusProxy{app: app})
	app.focus.Focus(app.url)

	app.statusText = "Ready"
	return app
}

// StatusText returns the latest request status line (for headless tests).
func (app *APITestApp) StatusText() string { return app.statusText }

func newEditorPanel(host *layout.Element, th *theme.Theme) *layout.Element {
	return layout.New(layout.Box().
		FlexGrow(1).
		PaddingAll(th.Spacing.S).
		Background(th.Surface).
		Border(th.Border, th.Stroke.Thin, th.Radius.Medium),
		host)
}

func (app *APITestApp) paintStatus(dl *render.DrawList, text *shape.Engine) {
	th := theme.Current()
	f := app.statusBar.Frame
	dl.AddRoundedRect(f, th.Radius.Medium, th.Chrome)
	pad := th.Spacing.MNudge
	cy := f.Y + f.H/2

	// No request yet (or in-flight): single muted line, no dot.
	if !app.haveResult {
		_, sh := text.Measure(app.statusText)
		text.DrawStringTop(dl, app.statusText, f.X+pad, cy-sh/2, th.ForegroundMuted)
		return
	}

	statusCol := th.Success
	switch {
	case app.respErr || app.respCode >= 400:
		statusCol = th.Error
	case app.respCode >= 300:
		statusCol = th.Warning
	}

	// Status dot.
	const dotR = 3.5
	dl.AddRoundedRect(render.Rect{X: f.X + pad, Y: cy - dotR, W: 2 * dotR, H: 2 * dotR}, dotR, statusCol)

	// Status code / line, color-coded.
	label := app.respStatus
	if label == "" {
		label = fmt.Sprintf("%d", app.respCode)
	}
	_, sh := text.Measure(label)
	text.DrawStringTop(dl, label, f.X+pad+2*dotR+th.Spacing.S, cy-sh/2, statusCol)

	// Right-aligned mono meta: "142 ms  ·  1.2 KB".
	if !app.respErr {
		meta := fmt.Sprintf("%d ms  ·  %s", app.respDuration.Milliseconds(), humanSize(app.respSize))
		mw, mh := text.MeasureMono(meta)
		text.DrawStringTopMono(dl, meta, f.X+f.W-pad-mw, cy-mh/2, th.ForegroundMuted)
	}
}

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

func (app *APITestApp) buildSplitter() {
	app.splitter = components.NewSplitter(app.splitDir,
		components.SplitSection{El: app.reqPane, Size: 0},
		components.SplitSection{El: app.respPane, Size: splitFixedSize},
	)
	app.splitHost.Children = []*layout.Element{app.splitter.El}
}

func (app *APITestApp) setSplit(dir components.Axis) {
	if dir == app.splitDir {
		return
	}
	app.splitDir = dir
	app.splitter.SetAxis(dir)
	if dir == components.Horizontal {
		app.splitToggle.Select(0)
	} else {
		app.splitToggle.Select(1)
	}
	app.relayout()
}

func (app *APITestApp) setReqTab(i int) {
	if i < 0 || i > 1 {
		return
	}
	app.reqTabs.Active = i
	switch i {
	case 0:
		app.reqPane.Children = []*layout.Element{app.reqTabs.El, app.bodyTypeRow, app.reqEditorPanel}
		app.reqHost.Children = []*layout.Element{app.bodyEditor.El}
		app.focusedEditor = app.bodyEditor
	case 1:
		app.reqPane.Children = []*layout.Element{app.reqTabs.El, app.reqEditorPanel}
		app.reqHost.Children = []*layout.Element{app.headersEditor.El}
		app.focusedEditor = app.headersEditor
	}
	app.relayout()
}

func (app *APITestApp) setRespTab(i int) {
	if i < 0 || i > 1 {
		return
	}
	app.respTabs.Active = i
	switch i {
	case 0:
		app.respHost.Children = []*layout.Element{app.respEditor.El}
		app.focusedEditor = app.respEditor
	case 1:
		app.respHost.Children = []*layout.Element{app.respHeadersEditor.El}
		app.focusedEditor = app.respHeadersEditor
	}
	app.relayout()
}

func (app *APITestApp) setBodyType(value string) {
	content := app.bodyEditor.Bytes()
	wasFocused := app.focusedEditor == app.bodyEditor
	app.bodyEditor.Close()
	var hl highlight.Highlighter
	if value == "json" {
		hl = highlight.NewJSON()
	} else {
		hl = highlight.Noop{}
	}
	app.bodyEditor = components.NewEditor(content, hl)
	if wasFocused {
		app.focusedEditor = app.bodyEditor
		app.bodyEditor.Focus()
	}
	if app.reqTabs.Active == 0 {
		app.reqHost.Children = []*layout.Element{app.bodyEditor.El}
	}
	app.relayout()
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
	app.bodyEditor = components.NewEditor(pretty.Bytes(), highlight.NewJSON())
	if wasFocused {
		app.focusedEditor = app.bodyEditor
		app.bodyEditor.Focus()
	}
	if app.reqTabs.Active == 0 {
		app.reqHost.Children = []*layout.Element{app.bodyEditor.El}
	}
	app.relayout()
}

func (app *APITestApp) pickEditorFocus(m *input.Mouse) {
	if m == nil || !m.Pressed {
		return
	}
	for _, ed := range []*components.Editor{
		app.respHeadersEditor, app.respEditor, app.headersEditor, app.bodyEditor,
	} {
		if ed != nil && ed.El.Frame.Contains(m.X, m.Y) {
			app.focusedEditor = ed
			return
		}
	}
}

func (app *APITestApp) activeEditor() *components.Editor {
	if app.focusedEditor != nil {
		return app.focusedEditor
	}
	return app.bodyEditor
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
	app.respEditor = components.NewEditor(body, hl)
	if wasFocused {
		app.focusedEditor = app.respEditor
		app.respEditor.Focus()
	}
	if app.respTabs.Active == 0 {
		app.respHost.Children = []*layout.Element{app.respEditor.El}
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
		app.respTabs.Tabs[1].Badge = strconv.Itoa(n)
	} else {
		app.respTabs.Tabs[1].Badge = ""
	}

	wasFocused := app.focusedEditor == app.respHeadersEditor
	app.respHeadersEditor.Close()
	app.respHeadersEditor = components.NewEditor([]byte(headers), highlight.Noop{})
	if wasFocused {
		app.focusedEditor = app.respHeadersEditor
		app.respHeadersEditor.Focus()
	}
	if app.respTabs.Active == 1 {
		app.respHost.Children = []*layout.Element{app.respHeadersEditor.El}
	}
}

func (app *APITestApp) doRequest() {
	if app.pending {
		return
	}
	url := strings.TrimSpace(app.url.Value)
	if url == "" {
		app.statusText = "Enter a URL"
		return
	}
	app.pending = true
	app.statusText = "Sending..."
	go app.executeRequest(url)
}

func (app *APITestApp) executeRequest(rawURL string) {
	start := time.Now()
	method := app.method.Options[app.method.Selected].Value

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
				if app.bodyType.Options[app.bodyType.Selected].Value == "json" {
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
	}
}

func (app *APITestApp) handleResult(r httpResult) {
	app.pending = false
	app.haveResult = true
	app.respDuration = r.duration
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
	app.relayout()
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

func (app *APITestApp) relayout() {
	app.root.MarkDirty()
	if app.lastW > 0 && app.lastH > 0 {
		app.root.Calculate(app.lastW, app.lastH)
	}
}

func (app *APITestApp) Root() *layout.Element { return app.root }

func (app *APITestApp) ClearColor() render.Color { return theme.Current().Background }

func (app *APITestApp) Layout(w, h float32) {
	app.lastW, app.lastH = w, h
	components.SetViewport(w, h)
	app.root.Calculate(w, h)
}

func (app *APITestApp) Update(m *input.Mouse, kb *input.Keyboard) {
	layout.Dispatch(app.root, m)
	app.pickEditorFocus(m)

	select {
	case r := <-app.resultCh:
		app.handleResult(r)
	default:
	}

	if kb != nil {
		for _, ev := range kb.Keys {
			if ev.Mods.Primary() && ev.Key == input.KeyEnter {
				app.doRequest()
			}
		}
	}

	if app.focus != nil {
		app.focus.HandleMouse(m)
		if kb != nil {
			app.focus.Route(kb)
		}
	}

	app.url.Update(m)
	app.bodyEditor.Update(m)
	app.headersEditor.Update(m)
	app.respEditor.Update(m)
	app.respHeadersEditor.Update(m)
}

func (app *APITestApp) AnimationWait() (time.Duration, bool) {
	if app.pending {
		return pendingPoll, true
	}
	if app.url.Focused() {
		return textFieldBlink, true
	}
	if app.focusedEditor != nil && app.focusedEditor.Focused() {
		return app.focusedEditor.AnimationWait()
	}
	return 0, false
}

func (app *APITestApp) Close() {
	app.bodyEditor.Close()
	app.headersEditor.Close()
	app.respEditor.Close()
	app.respHeadersEditor.Close()
}

type editorFocusProxy struct{ app *APITestApp }

func (p editorFocusProxy) editor() *components.Editor { return p.app.activeEditor() }

func (p editorFocusProxy) Focus()                        { p.editor().Focus() }
func (p editorFocusProxy) Blur()                         { p.editor().Blur() }
func (p editorFocusProxy) Focused() bool                 { return p.editor().Focused() }
func (p editorFocusProxy) HandleText(r []rune)           { p.editor().HandleText(r) }
func (p editorFocusProxy) HandleKeys(k []input.KeyEvent) { p.editor().HandleKeys(k) }
func (p editorFocusProxy) CapturesTab() bool             { return true }
func (p editorFocusProxy) FocusOnClick() bool            { return true }
func (p editorFocusProxy) FocusEl() *layout.Element      { return p.editor().El }
