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
	"github.com/mirzakhany/yoga/highlight"
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
	err        error
}

// APITestApp is an HTTP request tester implementing yoga.App.
type APITestApp struct {
	splitDir ui.Axis
	url      string

	method      *ui.Select
	splitToggle *ui.Segmented
	bodyType    *ui.Select

	bodyEditor        *ui.Editor
	headersEditor     *ui.Editor
	respEditor        *ui.Editor
	respHeadersEditor *ui.Editor

	reqTabs  *ui.TabBar
	respTabs *ui.TabBar

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
}

var _ yoga.App = (*APITestApp)(nil)

// BuildAPITestApp assembles the API test scene.
func BuildAPITestApp() *APITestApp {
	theme.Use("yoga-midnight")
	th := theme.Current()
	app := &APITestApp{
		resultCh: make(chan httpResult, 1),
		splitDir: ui.Horizontal,
		url:      defaultURL,
	}

	app.method = ui.NewSelect(100, []ui.SelectOption{
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

	app.splitToggle = ui.NewSegmented([]ui.SegmentItem{
		{Icon: "split_horizontal", Value: "h"},
		{Icon: "split_vertical", Value: "v"},
	}).ChangedValue(func(v string) {
		if v == "h" {
			app.setSplit(ui.Horizontal)
		} else {
			app.setSplit(ui.Vertical)
		}
	})

	app.bodyType = ui.NewSelect(100, []ui.SelectOption{
		{Label: "None", Value: "none"},
		{Label: "Text", Value: "text"},
		{Label: "JSON", Value: "json"},
	}).Changed(app.setBodyType)

	app.reqTabs = ui.NewTabBar()
	app.reqTabs.Bg = th.Background
	app.reqTabs.Tabs = []ui.TabModel{
		{Title: "Body"},
		{Title: "Headers"},
	}
	app.reqTabs.OnActivate = app.setReqTab

	app.respTabs = ui.NewTabBar()
	app.respTabs.Bg = th.Background
	app.respTabs.Tabs = []ui.TabModel{
		{Title: "Response"},
		{Title: "Headers"},
	}
	app.respTabs.OnActivate = app.setRespTab

	app.bodyEditor = ui.NewEditor([]byte("{\n  \n}"), highlight.NewJSON())
	app.headersEditor = ui.NewEditor([]byte(defaultReqHeader), highlight.Noop{})
	app.respEditor = ui.NewEditor(nil, highlight.Noop{})
	app.respHeadersEditor = ui.NewEditor(nil, highlight.Noop{})

	app.focusedEditor = app.bodyEditor
	app.statusText = "Ready"
	return app
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

	if kb := c.Keyboard(); kb != nil {
		for _, ev := range kb.Keys {
			if ev.Mods.Primary() && ev.Key == input.KeyEnter {
				app.doRequest()
			}
		}
	}
	if app.pending {
		c.Animate(pendingPoll)
	}

	return ui.Column(
		ui.Row(
			ui.ViewOf(app.method),
			ui.TextField("url", app.url).
				Placeholder("https://api.example.com/...").
				IconStart("terminal").
				OnChange(func(s string) { app.url = s }).
				DefaultFocus().
				Grow(1),
			ui.ViewOf(app.splitToggle),
			ui.Button("send", ui.Text("Send")).Primary().Hint("⌘↵").OnClick(app.doRequest),
		).Gap(th.Spacing.S).PaddingXY(th.Spacing.M, th.Spacing.M),
		ui.HLine(th.Stroke.Thin, th.Border),
		ui.Splitter("api-split", app.splitDir, app.reqPane(th), app.respPane(th)).
			Sizes(0, splitFixedSize).
			Grow(1),
	).Grow(1).Background(ui.TokenSurface)
}

func (app *APITestApp) reqPane(th *theme.Theme) ui.View {
	rows := []ui.View{ui.ViewOf(app.reqTabs)}
	if app.reqTabs.Active == 0 {
		rows = append(rows, app.bodyTypeRow(th))
	}
	rows = append(rows,
		ui.HLine(th.Stroke.Thin, th.Border),
		ui.ViewOf(app.activeReqEditor()).Grow(1),
	)
	return ui.Column(rows...).Gap(th.Spacing.M).PaddingXY(th.Spacing.M, th.Spacing.M).Grow(1)
}

func (app *APITestApp) respPane(th *theme.Theme) ui.View {
	return ui.Column(
		app.statusLine(th),
		ui.ViewOf(app.respTabs),
		ui.HLine(th.Stroke.Thin, th.Border),
		ui.ViewOf(app.activeRespEditor()).Grow(1),
	).Gap(th.Spacing.M).PaddingXY(th.Spacing.M, th.Spacing.M).Grow(1)
}

func (app *APITestApp) bodyTypeRow(th *theme.Theme) ui.View {
	return ui.Row(
		ui.Caption("Body type"),
		ui.ViewOf(app.bodyType),
		ui.Spacer(),
		ui.Button("beautify", ui.Text("Beautify")).Subtle().IconStart("menu").OnClick(app.beautifyBody),
	).Gap(th.Spacing.S).Height(th.Metrics.ControlHeight)
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
	if app.reqTabs.Active == 1 {
		return app.headersEditor
	}
	return app.bodyEditor
}

func (app *APITestApp) activeRespEditor() *ui.Editor {
	if app.respTabs.Active == 1 {
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
	if dir == app.splitDir {
		return
	}
	app.splitDir = dir
	if dir == ui.Horizontal {
		app.splitToggle.Select(0)
	} else {
		app.splitToggle.Select(1)
	}
}

func (app *APITestApp) setReqTab(i int) {
	if i < 0 || i > 1 {
		return
	}
	app.reqTabs.Active = i
	app.focusedEditor = app.activeReqEditor()
}

func (app *APITestApp) setRespTab(i int) {
	if i < 0 || i > 1 {
		return
	}
	app.respTabs.Active = i
	app.focusedEditor = app.activeRespEditor()
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
	app.bodyEditor = ui.NewEditor(content, hl)
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
	app.bodyEditor = ui.NewEditor(pretty.Bytes(), highlight.NewJSON())
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
	app.respEditor = ui.NewEditor(body, hl)
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
	app.respHeadersEditor = ui.NewEditor([]byte(headers), highlight.Noop{})
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

func (app *APITestApp) ClearColor() render.Color { return theme.Current().Background }

func (app *APITestApp) Close() {
	app.bodyEditor.Close()
	app.headersEditor.Close()
	app.respEditor.Close()
	app.respHeadersEditor.Close()
}
