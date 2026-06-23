package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/flipperdevices/flipctl/internal/viewdoc"
)

func TestInteractiveTUISourceRejectsLegacyPresentationEndpoints(t *testing.T) {
	b, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, bad := range []string{"/api/v1/apps", "/api/v1/jobs", "/api/v1/actions", "/api/v1/views", "/cancel", "renderPing", "renderNmap", "Result struct", "type App struct", "type Job struct"} {
		if strings.Contains(s, bad) {
			t.Fatalf("interactive TUI source still contains legacy pattern %q", bad)
		}
	}
	for _, want := range []string{"/api/v1/sessions/", "/actions", "viewId", "revision", "/api/v1/events"} {
		if !strings.Contains(s, want) {
			t.Fatalf("interactive TUI source missing canonical pattern %q", want)
		}
	}
}

func TestRefreshCreatesSessionAndFetchesCanonicalViewOnly(t *testing.T) {
	var hits []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits = append(hits, r.URL.Path)
		switch r.URL.Path {
		case "/api/v1/sessions":
			_ = json.NewEncoder(w).Encode(map[string]string{"id": "s-1"})
		case "/api/v1/sessions/s-1/view":
			_ = json.NewEncoder(w).Encode(doc("s-1", "home-app-list", 1, nil))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer ts.Close()
	msg := model{base: ts.URL, editBuffers: map[string]string{}}.refresh()
	loaded, ok := msg.(loaded)
	if !ok {
		t.Fatalf("refresh returned %T", msg)
	}
	if loaded.sessionID != "s-1" || loaded.viewDoc.ViewID != "home-app-list" {
		t.Fatalf("bad load: %#v", loaded)
	}
	if strings.Join(hits, ",") != "/api/v1/sessions,/api/v1/sessions/s-1/view" {
		t.Fatalf("unexpected hits %v", hits)
	}
}

func TestCanonicalActionDispatchUsesSelectedActionParamsNotListPresentation(t *testing.T) {
	var got map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/sessions/s-1/actions" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		_ = json.NewEncoder(w).Encode(doc("s-1", "ping-form", 2, nil))
	}))
	defer ts.Close()
	m := model{base: ts.URL, sessionID: "s-1", viewDoc: misleadingHomeDoc(), editBuffers: map[string]string{}}
	msg := m.invokeSelectedAction()
	if _, ok := msg.(loaded); !ok {
		t.Fatalf("invoke returned %T", msg)
	}
	params := requireParams(t, got)
	if got["actionId"] != "app.open" || got["viewId"] != "home-app-list" || got["revision"].(float64) != 7 || params["appId"] != "network.ping" {
		t.Fatalf("bad canonical action request %#v", got)
	}
	if got["appId"] != nil {
		t.Fatalf("app.open should not derive or flatten appId from presentation: %#v", got)
	}
}

func TestSelectedListItemActionIsOfferedAndInvoked(t *testing.T) {
	var got map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/sessions/s-1/actions" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		_ = json.NewEncoder(w).Encode(doc("s-1", "nmap-form", 2, nil))
	}))
	defer ts.Close()
	m := model{base: ts.URL, sessionID: "s-1", viewDoc: homeDoc(), itemCursor: 1, editBuffers: map[string]string{}}
	out := renderActions(m, 80, 10)
	if !strings.Contains(out, "Open Nmap [app.open]") {
		t.Fatalf("selected list action not offered in actions pane:\n%s", out)
	}
	msg := m.invokeSelectedAction()
	if _, ok := msg.(loaded); !ok {
		t.Fatalf("invoke returned %T", msg)
	}
	params := requireParams(t, got)
	if got["actionId"] != "app.open" || params["appId"] != "network.nmap" {
		t.Fatalf("bad selected item action request %#v", got)
	}
	if got["appId"] != nil {
		t.Fatalf("selected item action should not flatten appId from params: %#v", got)
	}
}

func TestFormUpdateAndJobStartDispatchCanonicalEnvelope(t *testing.T) {
	var bodies []map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/sessions/s-1/actions" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		var got map[string]any
		_ = json.NewDecoder(r.Body).Decode(&got)
		bodies = append(bodies, got)
		_ = json.NewEncoder(w).Encode(doc("s-1", "ping-form", 4, nil))
	}))
	defer ts.Close()
	m := model{base: ts.URL, sessionID: "s-1", viewDoc: misleadingFormDoc(), editBuffers: map[string]string{"target": "example.test", "count": "2"}}
	_ = m.invokeAction("form.update")
	_ = m.invokeAction("job.start")
	if len(bodies) != 2 {
		t.Fatalf("got %d bodies", len(bodies))
	}
	if vals := bodies[0]["values"].(map[string]any); vals["target"] != "example.test" || vals["count"] != "2" {
		t.Fatalf("bad values %#v", vals)
	}
	params := requireParams(t, bodies[1])
	if bodies[1]["actionId"] != "job.start" || params["appId"] != "network.ping" || bodies[1]["viewId"] != "nmap-form" || bodies[1]["revision"].(float64) != 3 {
		t.Fatalf("bad start %#v", bodies[1])
	}
	if bodies[1]["appId"] != nil {
		t.Fatalf("job.start should not derive or flatten appId from view id/title: %#v", bodies[1])
	}
}

func TestJobActionsUseParamsNotJobPresentation(t *testing.T) {
	var bodies []map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/sessions/s-1/actions" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		var got map[string]any
		_ = json.NewDecoder(r.Body).Decode(&got)
		bodies = append(bodies, got)
		_ = json.NewEncoder(w).Encode(doc("s-1", "ping-running", 6, nil))
	}))
	defer ts.Close()
	m := model{base: ts.URL, sessionID: "s-1", viewDoc: misleadingJobDoc(), editBuffers: map[string]string{}}
	if msg := m.invokeAction("job.cancel"); msg == nil {
		t.Fatalf("cancel returned nil")
	}
	if msg := m.invokeAction("job.run_again"); msg == nil {
		t.Fatalf("run again returned nil")
	}
	if len(bodies) != 2 {
		t.Fatalf("got %d bodies", len(bodies))
	}
	cancelParams := requireParams(t, bodies[0])
	if bodies[0]["actionId"] != "job.cancel" || cancelParams["jobId"] != "j-canonical" || bodies[0]["viewId"] != "nmap-running" || bodies[0]["revision"].(float64) != 5 {
		t.Fatalf("bad cancel request %#v", bodies[0])
	}
	runAgainParams := requireParams(t, bodies[1])
	if bodies[1]["actionId"] != "job.run_again" || runAgainParams["jobId"] != "j-canonical" || runAgainParams["appId"] != "network.ping" {
		t.Fatalf("bad run-again request %#v", bodies[1])
	}
	for _, got := range bodies {
		if got["jobId"] != nil || got["appId"] != nil {
			t.Fatalf("job action should not derive or flatten ids from presentation: %#v", got)
		}
	}
}

func TestNavigationBackDispatchesCanonicalEnvelopeWithoutPresentationContext(t *testing.T) {
	var got map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/sessions/s-1/actions" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		_ = json.NewEncoder(w).Encode(doc("s-1", "home-app-list", 6, nil))
	}))
	defer ts.Close()
	m := model{base: ts.URL, sessionID: "s-1", viewDoc: misleadingJobDoc(), editBuffers: map[string]string{}}
	msg := m.invokeAction("navigation.back")
	if _, ok := msg.(loaded); !ok {
		t.Fatalf("invoke returned %T", msg)
	}
	if got["actionId"] != "navigation.back" || got["viewId"] != "nmap-running" || got["revision"].(float64) != 5 {
		t.Fatalf("bad back request %#v", got)
	}
	if got["params"] != nil || got["jobId"] != nil || got["appId"] != nil || got["values"] != nil {
		t.Fatalf("back should not synthesize presentation context: %#v", got)
	}
}

func TestStaleRevisionRefetchesCurrentView(t *testing.T) {
	var paths []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if r.URL.Path == "/api/v1/sessions/s-1/actions" {
			http.Error(w, `{"code":"stale_view"}`, http.StatusConflict)
			return
		}
		if r.URL.Path == "/api/v1/sessions/s-1/view" {
			_ = json.NewEncoder(w).Encode(doc("s-1", "ping-form", 9, nil))
			return
		}
		t.Fatalf("unexpected path %s", r.URL.Path)
	}))
	defer ts.Close()
	m := model{base: ts.URL, sessionID: "s-1", viewDoc: homeDoc(), editBuffers: map[string]string{}}
	msg := m.invokeAction("app.open")
	loaded, ok := msg.(loaded)
	if !ok {
		t.Fatalf("invoke returned %T", msg)
	}
	if loaded.viewDoc.Revision != 9 {
		t.Fatalf("did not refetch current view: %#v", loaded.viewDoc)
	}
	if strings.Join(paths, ",") != "/api/v1/sessions/s-1/actions,/api/v1/sessions/s-1/view" {
		t.Fatalf("unexpected paths %v", paths)
	}
}

func TestWatchEventsReturnsChangedOnLiveSSEViewChanged(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/events" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"type\":\"view.changed\",\"sessionId\":\"s-1\",\"revision\":2}\n\n"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}))
	defer ts.Close()
	msg := model{base: ts.URL, sessionID: "s-1"}.watchEvents()
	if _, ok := msg.(sseChangedMsg); !ok {
		t.Fatalf("watchEvents returned %T", msg)
	}
}

func TestViewRendersGenericCanonicalDocumentAndEditor(t *testing.T) {
	m := model{sessionID: "s-1", viewDoc: formDoc(), connected: true, editBuffers: map[string]string{"target": "ya.ru", "count": "3"}, width: 100, height: 24}
	out := m.View()
	for _, want := range []string{"view=ping-form", "Canonical ViewDocument", "form", "Target", "Editable form fields", "Start job [job.start]"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in\n%s", want, out)
		}
	}
}

func TestViewRendersSelectableListCursorAndContextualAction(t *testing.T) {
	m := model{sessionID: "s-1", viewDoc: homeDoc(), connected: true, focus: focusDocument, width: 100, height: 24}
	out := m.View()
	for _, want := range []string{"Selectable list items", "> Ping [network.ping]", "enter Open Ping", "Open Ping [app.open]"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in\n%s", want, out)
		}
	}
}

func TestRenderFixtureUsesCanonicalFixtureCorpus(t *testing.T) {
	if err := renderFixture("../../schemas/view-document-fixtures/ping-form.json", 80, 20, "plain"); err != nil {
		t.Fatal(err)
	}
}

func doc(session, view string, rev int, blocks []viewdoc.Block) viewdoc.Document {
	if blocks == nil {
		blocks = []viewdoc.Block{{ID: "notice", Kind: "notice", Severity: "info", Text: "ok"}}
	}
	return viewdoc.Document{APIVersion: viewdoc.CanonicalAPIVersion, SessionID: session, ViewID: view, Revision: rev, Title: "FlipCTL", Blocks: blocks, Actions: []viewdoc.Action{{ID: "navigation.back", Label: "Back", Enabled: true}}}
}
func homeDoc() *viewdoc.Document {
	d := doc("s-1", "home-app-list", 7, []viewdoc.Block{{ID: "apps", Kind: "list", Title: "Apps", Items: []viewdoc.ListItem{
		{Label: "Ping", Value: "network.ping", Action: &viewdoc.Action{ID: "app.open", Label: "Open Ping", Enabled: true, Params: map[string]string{"appId": "network.ping"}}},
		{Label: "Nmap", Value: "network.nmap", Action: &viewdoc.Action{ID: "app.open", Label: "Open Nmap", Enabled: true, Params: map[string]string{"appId": "network.nmap"}}},
	}}})
	d.Actions = nil
	return &d
}
func misleadingHomeDoc() *viewdoc.Document {
	d := doc("s-1", "home-app-list", 7, []viewdoc.Block{{ID: "apps", Kind: "list", Title: "Apps", Items: []viewdoc.ListItem{
		{Label: "Actually Nmap", Value: "network.nmap", Action: &viewdoc.Action{ID: "app.open", Label: "Open misleading label", Enabled: true, Params: map[string]string{"appId": "network.ping"}}},
	}}})
	d.Actions = nil
	return &d
}
func formDoc() *viewdoc.Document {
	d := doc("s-1", "ping-form", 3, []viewdoc.Block{{ID: "form", Kind: "form", Title: "Ping options", Fields: []viewdoc.FormField{{ID: "target", Label: "Target", Type: "string", Default: "ya.ru"}, {ID: "count", Label: "Count", Type: "int", Default: "3"}}}})
	d.Actions = []viewdoc.Action{{ID: "form.update", Label: "Update form", Enabled: true}, {ID: "job.start", Label: "Start job", Enabled: true, Params: map[string]string{"appId": "network.ping"}}}
	return &d
}
func misleadingFormDoc() *viewdoc.Document {
	d := formDoc()
	d.ViewID = "nmap-form"
	d.Title = "Nmap"
	return d
}
func jobDoc() *viewdoc.Document {
	d := doc("s-1", "ping-running", 5, []viewdoc.Block{{ID: "meta", Kind: "key_value", Pairs: []viewdoc.KeyValuePair{{Key: "jobId", Value: "j-9"}, {Key: "appId", Value: "network.ping"}, {Key: "state", Value: "running"}}}, {ID: "progress", Kind: "progress", Progress: &viewdoc.Progress{Label: "Running", Text: "Job is running"}}})
	d.Actions = []viewdoc.Action{{ID: "navigation.back", Label: "Back", Enabled: true}, {ID: "job.run_again", Label: "Run again", Enabled: true, Params: map[string]string{"jobId": "j-9", "appId": "network.ping"}}, {ID: "job.cancel", Label: "Cancel", Enabled: true, Params: map[string]string{"jobId": "j-9"}}}
	return &d
}
func misleadingJobDoc() *viewdoc.Document {
	d := doc("s-1", "nmap-running", 5, []viewdoc.Block{{ID: "meta", Kind: "key_value", Pairs: []viewdoc.KeyValuePair{{Key: "jobId", Value: "j-from-presentation"}, {Key: "appId", Value: "network.nmap"}, {Key: "state", Value: "running"}}}, {ID: "progress", Kind: "progress", Progress: &viewdoc.Progress{Label: "Running Nmap", Text: "Job is running"}}})
	d.Title = "Job j-from-title"
	d.Actions = []viewdoc.Action{{ID: "navigation.back", Label: "Back", Enabled: true}, {ID: "job.run_again", Label: "Run again", Enabled: true, Params: map[string]string{"jobId": "j-canonical", "appId": "network.ping"}}, {ID: "job.cancel", Label: "Cancel", Enabled: true, Params: map[string]string{"jobId": "j-canonical"}}}
	return &d
}
func requireParams(t *testing.T, got map[string]any) map[string]any {
	t.Helper()
	params, ok := got["params"].(map[string]any)
	if !ok {
		t.Fatalf("missing params in request %#v", got)
	}
	return params
}
