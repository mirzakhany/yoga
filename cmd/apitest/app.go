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

// APITestApp is an HTTP request tester implementing yoga.Scene. It uses the Body
// pattern (layout.Host): persistent state lives in these fields and body()
// derives the element tree from them. State changes call host.Invalidate() —
// there is no manual .Children surgery or relayout().
//
// A few elements are persistent because they own interactive state body() must
// not discard: the splitter (drag-resized pane sizes, axis) and its two section
// containers, plus the status bar (its Paint hook reads its own Frame). body()
// fills the section containers' children each rebuild; the splitter keeps owning
// their size and axis.
type APITestApp struct {
	host        *layout.Host
	reqSection  *layout.Element
	respSection *layout.Element
	statusBar   *layout.Element
	splitDir    components.Axis

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
	app.url.El.Grow(1)

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

	app.bodyType = components.NewSelect(100, []components.SelectOption{
		{Label: "Text", Value: "text"},
		{Label: "JSON", Value: "json"},
	}).Changed(app.setBodyType)
	app.beautify = components.NewButton("Beautify").Subtle().IconStart("menu").Action(app.beautifyBody)

	app.reqTabs = components.NewTabBar()
	app.reqTabs.Bg = th.Background
	app.reqTabs.Tabs = []components.TabModel{
		{Title: "Body"},
		{Title: "Headers"},
	}
	app.reqTabs.OnActivate = app.setReqTab

	app.respTabs = components.NewTabBar()
	app.respTabs.Bg = th.Background
	app.respTabs.Tabs = []components.TabModel{
		{Title: "Response"},
		{Title: "Headers"},
	}
	app.respTabs.OnActivate = app.setRespTab

	app.bodyEditor = components.NewEditor([]byte("{\n  \n}"), highlight.NewJSON())
	app.headersEditor = components.NewEditor([]byte(defaultReqHeader), highlight.Noop{})
	app.respEditor = components.NewEditor(nil, highlight.Noop{})
	app.respHeadersEditor = components.NewEditor(nil, highlight.Noop{})

	app.statusBar = layout.New(layout.Box().H(th.Metrics.ControlHeight))
	app.statusBar.Paint = app.paintStatus

	// Persistent splitter and its section containers: the splitter owns the
	// drag-resized pane sizes and axis, so it must survive body() rebuilds.
	// NewSplitter overwrites each section's Style; body() only fills .Children.
	app.reqSection = layout.New(layout.Box().Direction(layout.Row))
	app.respSection = layout.New(layout.Box().Direction(layout.Row))
	app.buildSplitter()

	app.host = layout.NewHost(app.body)

	app.focus = components.NewFocusManager()
	app.focusedEditor = app.bodyEditor
	app.focus.Add(app.url, app.method, app.bodyType, app.reqTabs, app.respTabs, app.send, editorFocusProxy{app: app})
	app.focus.Focus(app.url)

	app.statusText = "Ready"
	return app
}

// body derives the whole element tree from current state. The runtime calls it
// (via host) on the first frame and after every Invalidate — never directly.
func (app *APITestApp) body() *layout.Element {
	th := theme.Current()

	toolbar := layout.HStack(th.Spacing.S, app.method.El, app.url.El, app.splitToggle.El, app.send.El)
	toolbarWrap := layout.New(layout.Box().PaddingXY(th.Spacing.M, th.Spacing.M), toolbar)

	// Fill the persistent splitter sections from current tab/editor state.
	app.reqSection.Children = []*layout.Element{app.reqPane(th)}
	app.respSection.Children = []*layout.Element{app.respPane(th)}

	splitHost := layout.New(layout.Box().FlexGrow(1), app.splitter.El)

	// Full-bleed horizontal rule under the toolbar; the vertical splitter line
	// meets it to form the section dividers (mockup structure).
	return layout.New(layout.Box().Direction(layout.Column).FlexGrow(1),
		toolbarWrap,
		rule(th),
		splitHost,
		app.method.MenuEl(),
		app.bodyType.MenuEl(),
	).BgPtr(&th.Background)
}

// reqPane builds the request column: tabs, an optional body-type row (Body tab
// only), a divider rule, and the active editor.
func (app *APITestApp) reqPane(th *theme.Theme) *layout.Element {
	rows := []*layout.Element{app.reqTabs.El}
	if app.reqTabs.Active == 0 {
		rows = append(rows, app.bodyTypeRow(th))
	}
	rows = append(rows, rule(th), editorHost(app.activeReqEditor().El))
	return layout.New(layout.Box().Direction(layout.Column).FlexGrow(1).Gap(th.Spacing.M).PaddingXY(th.Spacing.M, th.Spacing.M), rows...)
}

// respPane builds the response column: status bar, tabs, a divider rule, and the
// active editor. Both editors align at the same Y so the per-pane rules read as
// one continuous line beneath the header chrome.
func (app *APITestApp) respPane(th *theme.Theme) *layout.Element {
	return layout.New(layout.Box().Direction(layout.Column).FlexGrow(1).Gap(th.Spacing.M).PaddingXY(th.Spacing.M, th.Spacing.M),
		app.statusBar,
		app.respTabs.El,
		rule(th),
		editorHost(app.activeRespEditor().El),
	)
}

// bodyTypeRow is the "Body type" selector row. Its fixed height and horizontal
// pad align the request column's rows with the response column's status/tabs
// rows so both editors start at the same Y.
func (app *APITestApp) bodyTypeRow(th *theme.Theme) *layout.Element {
	label := components.NewLabel("Body type", components.LabelCaption)
	return layout.HStack(th.Spacing.S, label.El, app.bodyType.El, layout.Spacer(), app.beautify.El).
		Height(th.Metrics.ControlHeight).
		PaddingXY(th.Spacing.M, 0)
}

func (app *APITestApp) activeReqEditor() *components.Editor {
	if app.reqTabs.Active == 1 {
		return app.headersEditor
	}
	return app.bodyEditor
}

func (app *APITestApp) activeRespEditor() *components.Editor {
	if app.respTabs.Active == 1 {
		return app.respHeadersEditor
	}
	return app.respEditor
}

// editorHost wraps an editor element in a clipped, flex-filling viewport.
func editorHost(ed *layout.Element) *layout.Element {
	h := layout.New(layout.Box().FlexGrow(1), ed)
	h.Clip = true
	return h
}

// rule is a thin full-width horizontal divider in the theme border color.
func rule(th *theme.Theme) *layout.Element {
	return layout.New(layout.Box().H(th.Stroke.Thin)).BgPtr(&th.Border)
}

// StatusText returns the latest request status line (for headless tests).
func (app *APITestApp) StatusText() string { return app.statusText }

func (app *APITestApp) paintStatus(dl *render.DrawList, text *shape.Engine) {
	th := theme.Current()
	f := app.statusBar.Frame
	// Transparent row (no fill): the status sits in the response column's header
	// band, level with the request tabs, like the mockup. The left pad matches
	// the tab bar's internal text inset so "200 OK" aligns under the tab labels.
	pad := th.Spacing.M
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
		components.SplitSection{El: app.reqSection, Size: 0},
		components.SplitSection{El: app.respSection, Size: splitFixedSize},
	)
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
	app.host.Invalidate()
}

func (app *APITestApp) setReqTab(i int) {
	if i < 0 || i > 1 {
		return
	}
	app.reqTabs.Active = i
	app.focusedEditor = app.activeReqEditor()
	app.host.Invalidate()
}

func (app *APITestApp) setRespTab(i int) {
	if i < 0 || i > 1 {
		return
	}
	app.respTabs.Active = i
	app.focusedEditor = app.activeRespEditor()
	app.host.Invalidate()
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
	app.host.Invalidate()
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
	app.host.Invalidate()
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
	app.host.Invalidate()
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

func (app *APITestApp) Root() *layout.Element { return app.host.Root() }

func (app *APITestApp) ClearColor() render.Color { return theme.Current().Background }

func (app *APITestApp) Layout(w, h float32) {
	components.SetViewport(w, h)
	app.host.Layout(w, h)
}

func (app *APITestApp) Update(m *input.Mouse, kb *input.Keyboard) {
	layout.Dispatch(app.host.Root(), m)
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
