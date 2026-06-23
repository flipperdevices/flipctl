package api

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"

	"github.com/flipperdevices/flipctl/internal/plugin"
	"github.com/flipperdevices/flipctl/internal/viewdoc"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func testServer() *httptest.Server {
	apps, _ := loadTestPluginApps("plugins")
	return httptest.NewServer(New(apps, nil).Handler())
}

func loadTestPluginApps(dir string) ([]plugin.App, error) {
	root, err := findRepoRoot()
	if err != nil {
		return nil, err
	}
	apps, err := plugin.LoadDir(filepath.Join(root, dir))
	if err != nil {
		return nil, err
	}
	for i := range apps {
		if strings.HasPrefix(apps[i].Command.Executable, "./fakecmds/") {
			apps[i].Command.Executable = filepath.Join(root, strings.TrimPrefix(apps[i].Command.Executable, "./"))
		}
	}
	return apps, nil
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		next := filepath.Dir(dir)
		if next == dir {
			return "", fmt.Errorf("go.mod not found above %s", dir)
		}
		dir = next
	}
}
func TestContractEndpointsAndValidation(t *testing.T) {
	ts := testServer()
	defer ts.Close()
	created := createSession(t, ts)
	for _, p := range []string{"/api/v1/meta", "/api/v1/capabilities", "/api/v1/apps", "/api/v1/schema/view-document.json", "/api/v1/openapi.json", "/api/v1/sessions/" + created.ID, "/api/v1/sessions/" + created.ID + "/view"} {
		resp, err := http.Get(ts.URL + p)
		if err != nil || resp.StatusCode > 299 {
			t.Fatalf("%s status/err %v %v", p, resp.StatusCode, err)
		}
		resp.Body.Close()
	}
	assertLegacyRouteUnavailable(t, ts, http.MethodGet, "/api/v1/views/network.ping")
	assertLegacyRouteUnavailable(t, ts, http.MethodPost, "/api/v1/actions")
	assertLegacyRouteUnavailable(t, ts, http.MethodPost, "/api/v1/jobs/j-1/cancel")
}

func TestJobsListSortsNewestFirstByNumericJobID(t *testing.T) {
	s := New(nil, nil)
	s.jobs["j-2"] = Job{ID: "j-2", JobID: "j-2", AppID: "network.ping", State: "done", Created: "2026-06-21T00:00:02Z"}
	s.jobs["j-11"] = Job{ID: "j-11", JobID: "j-11", AppID: "network.nmap", State: "done", Created: "2026-06-21T00:00:11Z"}
	s.jobs["j-9"] = Job{ID: "j-9", JobID: "j-9", AppID: "network.ping", State: "done", Created: "2026-06-21T00:00:09Z"}
	s.jobs["j-10"] = Job{ID: "j-10", JobID: "j-10", AppID: "network.nmap", State: "done", Created: "2026-06-21T00:00:10Z"}
	r := httptest.NewRequest(http.MethodGet, "/api/v1/jobs", nil)
	w := httptest.NewRecorder()
	s.jobsList(w, r)
	var jobs []Job
	if err := json.NewDecoder(w.Body).Decode(&jobs); err != nil {
		t.Fatal(err)
	}
	ids := []string{}
	for _, job := range jobs {
		ids = append(ids, job.ID)
	}
	want := []string{"j-11", "j-10", "j-9", "j-2"}
	if strings.Join(ids, ",") != strings.Join(want, ",") {
		t.Fatalf("want %v, got %v", want, ids)
	}
}

func TestRealToolsNmapAcceptsExternalTargetAndCreatesJob(t *testing.T) {
	apps, err := loadTestPluginApps("plugins-real")
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(New(apps, nil).Handler())
	defer ts.Close()
	job := startCanonicalJob(t, ts, "network.nmap", map[string]string{"target": "ya.ru"})
	if job.ID == "" || job.AppID != "network.nmap" {
		t.Fatalf("unexpected job: %#v", job)
	}
}

func TestCancelAndSSEReplay(t *testing.T) {
	ts := testServer()
	defer ts.Close()
	job := startSlow(t, ts)
	session := createSession(t, ts)
	doc := getCanonicalDoc(t, ts, session.ID)
	postSessionAction(t, ts, session.ID, map[string]any{"actionId": "job.select", "viewId": doc.ViewID, "revision": doc.Revision, "jobId": job.ID})
	doc = waitCanonicalJobState(t, ts, session.ID, job.ID, "running")
	postSessionAction(t, ts, session.ID, map[string]any{"actionId": "job.cancel", "viewId": doc.ViewID, "revision": doc.Revision, "jobId": job.ID})
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		r, _ := http.Get(ts.URL + "/api/v1/jobs/" + job.ID)
		json.NewDecoder(r.Body).Decode(&job)
		r.Body.Close()
		if job.State == "canceled" {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if job.State != "canceled" {
		t.Fatalf("want canceled, got %#v", job)
	}
	r, err := http.Get(ts.URL + "/api/v1/events?after=0&replayOnly=1")
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	s := bufio.NewScanner(r.Body)
	sawStart, sawCancel := false, false
	for s.Scan() {
		line := s.Text()
		if strings.Contains(line, "job.started") {
			sawStart = true
		}
		if strings.Contains(line, "job.canceled") || strings.Contains(line, "job.cancel_requested") {
			sawCancel = true
		}
	}
	if !sawStart || !sawCancel {
		t.Fatalf("missing replay events start=%v cancel=%v", sawStart, sawCancel)
	}
}
func TestSSELiveSubscriber(t *testing.T) {
	ts := testServer()
	defer ts.Close()
	client := ts.Client()
	req, _ := http.NewRequest("GET", ts.URL+"/api/v1/events?after=0", nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	lines := make(chan string, 16)
	go func() {
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
	}()
	startSlow(t, ts)
	deadline := time.After(2 * time.Second)
	for {
		select {
		case line := <-lines:
			if strings.Contains(line, "job.started") {
				return
			}
		case <-deadline:
			t.Fatal("live SSE subscriber did not receive job.started")
		}
	}
}
func TestJobEventsStreamingProgressAndReplayOrder(t *testing.T) {
	ts := httptest.NewServer(New(appsWithScriptedPingOutput(t), nil).Handler())
	defer ts.Close()
	startCanonicalJob(t, ts, "network.ping", map[string]string{"target": "127.0.0.1", "count": "1"})
	outputID, progressID, doneID, sawReply := waitReplayOrder(t, ts)
	if outputID == 0 || progressID == 0 || doneID == 0 || !(outputID < doneID && progressID < doneID) {
		t.Fatalf("bad event order output=%d progress=%d done=%d", outputID, progressID, doneID)
	}
	if !sawReply {
		t.Fatalf("missing parser-compatible ping reply in replay output")
	}
}
func startSlow(t *testing.T, ts *httptest.Server) Job {
	t.Helper()
	return startCanonicalJob(t, ts, "network.ping", map[string]string{"target": "slow.test", "count": "1"})
}

func waitReplayOrder(t *testing.T, ts *httptest.Server) (outputID, progressID, doneID int, sawReply bool) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		lastID, outputID, progressID, doneID := 0, 0, 0, 0
		sawReply := false
		r, err := http.Get(ts.URL + "/api/v1/events?after=0&replayOnly=1")
		if err != nil {
			t.Fatal(err)
		}
		s := bufio.NewScanner(r.Body)
		for s.Scan() {
			line := s.Text()
			if strings.HasPrefix(line, "id: ") {
				var id int
				fmt.Sscanf(line, "id: %d", &id)
				if id <= lastID {
					r.Body.Close()
					t.Fatalf("non-monotonic id %d after %d", id, lastID)
				}
				lastID = id
			}
			if strings.Contains(line, "64 bytes from 127.0.0.1: icmp_seq=1 ttl=64 time=0.1 ms") {
				sawReply = true
			}
			if strings.Contains(line, "job.output") && outputID == 0 {
				outputID = lastID
			}
			if strings.Contains(line, "job.progress") && progressID == 0 {
				progressID = lastID
			}
			if strings.Contains(line, "job.done") && doneID == 0 {
				doneID = lastID
			}
		}
		lastErr = s.Err()
		r.Body.Close()
		if lastErr != nil {
			t.Fatal(lastErr)
		}
		if outputID != 0 && progressID != 0 && doneID != 0 && outputID < doneID && progressID < doneID && sawReply {
			return outputID, progressID, doneID, sawReply
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("replay did not contain ordered output/progress/done with parser-compatible reply before deadline: output=%d progress=%d done=%d sawReply=%v err=%v", outputID, progressID, doneID, sawReply, lastErr)
	return 0, 0, 0, false
}

func startCanonicalJob(t *testing.T, ts *httptest.Server, appID string, fields map[string]string) Job {
	t.Helper()
	session := createSession(t, ts)
	doc := getCanonicalDoc(t, ts, session.ID)
	started := postSessionActionDoc(t, ts, session.ID, map[string]any{"actionId": "job.start", "viewId": doc.ViewID, "revision": doc.Revision, "appId": appID, "values": fields})
	jobID := canonicalJobID(t, started)
	if jobID == "" {
		t.Fatalf("canonical job.start response did not project a job: %#v", started)
	}
	resp, err := http.Get(ts.URL + "/api/v1/jobs/" + jobID)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("job lookup status %d body=%s", resp.StatusCode, body)
	}
	var job Job
	if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
		t.Fatal(err)
	}
	if job.State != "running" {
		t.Fatalf("job not running: %#v", job)
	}
	return job
}

func assertLegacyRouteUnavailable(t *testing.T, ts *httptest.Server, method, path string) {
	t.Helper()
	req, err := http.NewRequest(method, ts.URL+path, strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusMethodNotAllowed {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("legacy route %s %s should be unavailable by default, got %d body=%s", method, path, resp.StatusCode, body)
	}
}

func createSession(t *testing.T, ts *httptest.Server) Session {
	t.Helper()
	resp, err := http.Post(ts.URL+"/api/v1/sessions", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create session status %d", resp.StatusCode)
	}
	var s Session
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

func getDoc(t *testing.T, ts *httptest.Server, sessionID string) map[string]any {
	t.Helper()
	resp, err := http.Get(ts.URL + "/api/v1/sessions/" + sessionID + "/view")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("view status %d", resp.StatusCode)
	}
	var doc map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		t.Fatal(err)
	}
	return doc
}

func TestTUIContract_JobStartReturnsCanonicalViewDocument(t *testing.T) {
	ts := testServer()
	defer ts.Close()
	session := createSession(t, ts)
	opened := postSessionActionDoc(t, ts, session.ID, map[string]any{"actionId": "app.open", "viewId": "home-app-list", "revision": 1, "params": map[string]string{"appId": "network.ping"}})
	updated := postSessionActionDoc(t, ts, session.ID, map[string]any{"actionId": "form.update", "viewId": opened.ViewID, "revision": opened.Revision, "values": map[string]string{"target": "127.0.0.1", "count": "1"}})
	started := postSessionActionDoc(t, ts, session.ID, map[string]any{"actionId": "job.start", "viewId": updated.ViewID, "revision": updated.Revision, "appId": "network.ping"})

	jobID := canonicalJobID(t, started)
	if jobID == "" {
		t.Fatalf("job.start response did not include started job projection: %#v", started)
	}
	if got := canonicalJobState(started); got != "running" {
		t.Fatalf("job.start response state = %q, want running: %#v", got, started)
	}
	if !docHasKind(started, "progress") {
		t.Fatalf("job.start response did not include running progress block: %#v", started.Blocks)
	}
	if started.ViewID != "ping-running" {
		t.Fatalf("job.start response viewID = %q, want ping-running", started.ViewID)
	}
}

func TestCanonicalSessionsActionsAndViewChanged(t *testing.T) {
	ts := testServer()
	defer ts.Close()
	s1, s2 := createSession(t, ts), createSession(t, ts)
	doc1 := getDoc(t, ts, s1.ID)
	if doc1["apiVersion"] != "viewdoc.flipctl.dev/v1alpha1" || doc1["sessionId"] != s1.ID || len(doc1["blocks"].([]any)) == 0 {
		t.Fatalf("bad doc %#v", doc1)
	}
	act := fmt.Sprintf(`{"actionId":"form.update","viewId":%q,"revision":%.0f,"values":{"target":"127.0.0.1","count":"1"}}`, doc1["viewId"], doc1["revision"])
	resp, err := http.Post(ts.URL+"/api/v1/sessions/"+s1.ID+"/actions", "application/json", strings.NewReader(act))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("form.update status %d", resp.StatusCode)
	}
	resp.Body.Close()
	stale, _ := http.Post(ts.URL+"/api/v1/sessions/"+s1.ID+"/actions", "application/json", strings.NewReader(act))
	if stale.StatusCode != http.StatusConflict {
		t.Fatalf("want stale conflict got %d", stale.StatusCode)
	}
	stale.Body.Close()
	jdoc := getDoc(t, ts, s1.ID)
	start := fmt.Sprintf(`{"actionId":"job.start","viewId":%q,"revision":%.0f,"appId":"network.ping"}`, jdoc["viewId"], jdoc["revision"])
	resp, err = http.Post(ts.URL+"/api/v1/sessions/"+s1.ID+"/actions", "application/json", strings.NewReader(start))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("job.start status %d", resp.StatusCode)
	}
	var startedDoc viewdoc.Document
	if err := json.NewDecoder(resp.Body).Decode(&startedDoc); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	ValidateViewDocument(t, startedDoc)
	if startedDoc.SessionID != s1.ID || startedDoc.APIVersion != viewdoc.CanonicalAPIVersion || canonicalJobID(t, startedDoc) == "" || canonicalJobState(startedDoc) != "running" {
		t.Fatalf("bad job.start view document %#v", startedDoc)
	}
	other := getDoc(t, ts, s2.ID)
	if other["revision"] != float64(1) {
		t.Fatalf("session 2 changed: %#v", other)
	}
	r, err := http.Get(ts.URL + "/api/v1/events?after=0&replayOnly=1")
	if err != nil {
		t.Fatalf("events replay: %v", err)
	}
	defer r.Body.Close()
	body := new(bytes.Buffer)
	body.ReadFrom(r.Body)
	if !strings.Contains(body.String(), "view.changed") || !strings.Contains(body.String(), s1.ID) {
		t.Fatalf("missing view.changed: %s", body.String())
	}
}

func TestLiveSessionViewEndpointValidatesAgainstViewDocumentSchema(t *testing.T) {
	ts := testServer()
	defer ts.Close()
	session := createSession(t, ts)

	resp, err := http.Get(ts.URL + "/api/v1/sessions/" + session.ID + "/view")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("view status %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	doc := validateHTTPViewDocument(t, raw)
	if doc.ViewID != "home-app-list" || doc.SessionID != session.ID {
		t.Fatalf("unexpected HTTP view document: %#v", doc)
	}
}

func TestProjectorAddsSemanticActionParams(t *testing.T) {
	s := New([]plugin.App{{ID: "network.ping", Title: "Ping", Summary: "Ping target", Fields: []plugin.Field{{ID: "target", Label: "Target", Type: "string"}}}}, nil)
	ss := &Session{ID: "s-1", CurrentAppID: "network.ping", FormValues: map[string]string{}, Revision: 1}

	formDoc := s.projectLocked(ss)
	startParams := requireActionParams(t, formDoc, "job.start")
	if startParams["appId"] != "network.ping" {
		t.Fatalf("job.start params should carry appId, got %#v", startParams)
	}

	ss.SelectedJobID = "j-9"
	s.jobs["j-9"] = Job{ID: "j-9", JobID: "j-9", AppID: "network.ping", SessionID: ss.ID, State: "running"}
	jobDoc := s.projectLocked(ss)
	runAgainParams := requireActionParams(t, jobDoc, "job.run_again")
	if runAgainParams["appId"] != "network.ping" || runAgainParams["jobId"] != "j-9" {
		t.Fatalf("job.run_again params should carry appId/jobId, got %#v", runAgainParams)
	}
	cancelParams := requireActionParams(t, jobDoc, "job.cancel")
	if cancelParams["jobId"] != "j-9" {
		t.Fatalf("job.cancel params should carry jobId, got %#v", cancelParams)
	}
}

func TestArchitecture_OneDaemonTwoSessionsOneSharedJob(t *testing.T) {
	ts := testServer()
	defer ts.Close()

	webSession, tuiSession := createSession(t, ts), createSession(t, ts)
	webHome := getCanonicalDoc(t, ts, webSession.ID)
	tuiHome := getCanonicalDoc(t, ts, tuiSession.ID)
	assertRevisionAdvanced(t, "web starts at initial revision", 0, webHome)
	assertRevisionAdvanced(t, "tui starts at initial revision", 0, tuiHome)
	if webHome.SessionID == tuiHome.SessionID || webHome.ViewID != "home-app-list" || tuiHome.ViewID != "home-app-list" {
		t.Fatalf("expected independent home views: web=%#v tui=%#v", webHome, tuiHome)
	}

	postSessionAction(t, ts, webSession.ID, map[string]any{"actionId": "app.open", "viewId": webHome.ViewID, "revision": webHome.Revision, "appId": "network.ping"})
	webPingForm := getCanonicalDoc(t, ts, webSession.ID)
	assertRevisionAdvanced(t, "web opens Ping", webHome.Revision, webPingForm)
	if webPingForm.ViewID != "ping-form" {
		t.Fatalf("web should be on Ping form, got %#v", webPingForm)
	}
	tuiStillHome := getCanonicalDoc(t, ts, tuiSession.ID)
	if tuiStillHome.ViewID != tuiHome.ViewID || tuiStillHome.Revision != tuiHome.Revision {
		t.Fatalf("TUI-like session should remain on independent home view: before=%#v after=%#v", tuiHome, tuiStillHome)
	}

	postSessionAction(t, ts, webSession.ID, map[string]any{"actionId": "form.update", "viewId": webPingForm.ViewID, "revision": webPingForm.Revision, "values": map[string]string{"target": "slow.test", "count": "1"}})
	webConfigured := getCanonicalDoc(t, ts, webSession.ID)
	assertRevisionAdvanced(t, "web sets slow.test", webPingForm.Revision, webConfigured)
	if !formFieldDefaultEquals(webConfigured, "target", "slow.test") {
		t.Fatalf("web form did not retain canonical slow.test target: %#v", webConfigured.Blocks)
	}

	postSessionAction(t, ts, webSession.ID, map[string]any{"actionId": "job.start", "viewId": webConfigured.ViewID, "revision": webConfigured.Revision, "appId": "network.ping"})
	webRunning := waitCanonicalJobState(t, ts, webSession.ID, "", "running")
	assertRevisionAdvanced(t, "web starts deterministic Ping", webConfigured.Revision, webRunning)
	jobID := canonicalJobID(t, webRunning)
	if jobID == "" || webRunning.ViewID != "ping-running" {
		t.Fatalf("web should observe running Ping job via canonical view: jobID=%q doc=%#v", jobID, webRunning)
	}

	postSessionAction(t, ts, tuiSession.ID, map[string]any{"actionId": "job.select", "viewId": tuiStillHome.ViewID, "revision": tuiStillHome.Revision, "jobId": jobID})
	tuiRunning := waitCanonicalJobState(t, ts, tuiSession.ID, jobID, "running")
	assertRevisionAdvanced(t, "tui observes shared job", tuiStillHome.Revision, tuiRunning)
	if tuiRunning.ViewID != "ping-running" || canonicalJobID(t, tuiRunning) != jobID {
		t.Fatalf("TUI-like session did not observe shared job through canonical ViewDocument: %#v", tuiRunning)
	}

	postSessionAction(t, ts, tuiSession.ID, map[string]any{"actionId": "job.cancel", "viewId": tuiRunning.ViewID, "revision": tuiRunning.Revision, "jobId": jobID})
	tuiCanceled := waitCanonicalJobState(t, ts, tuiSession.ID, jobID, "canceled")
	assertRevisionAdvanced(t, "tui cancels shared job", tuiRunning.Revision, tuiCanceled)
	webCanceled := waitCanonicalJobState(t, ts, webSession.ID, jobID, "canceled")
	assertRevisionAdvanced(t, "web observes canceled shared job", webRunning.Revision, webCanceled)

	r, err := http.Get(ts.URL + "/api/v1/events?after=0&replayOnly=1")
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	assertArchitectureEvents(t, parseSSEEvents(t, r.Body), webSession.ID, tuiSession.ID, jobID)
}

func TestCrossFrontendCanonicalSessionsShareJobsAndKeepIndependentNavigation(t *testing.T) {
	ts := testServer()
	defer ts.Close()
	webSession, tuiSession := createSession(t, ts), createSession(t, ts)
	webDoc := getCanonicalDoc(t, ts, webSession.ID)
	tuiDoc := getCanonicalDoc(t, ts, tuiSession.ID)
	if webDoc.SessionID == tuiDoc.SessionID || webDoc.APIVersion != viewdoc.CanonicalAPIVersion || tuiDoc.APIVersion != viewdoc.CanonicalAPIVersion {
		t.Fatalf("bad session docs web=%#v tui=%#v", webDoc, tuiDoc)
	}

	postSessionAction(t, ts, webSession.ID, map[string]any{"actionId": "form.update", "viewId": webDoc.ViewID, "revision": webDoc.Revision, "values": map[string]string{"target": "slow.test", "count": "1"}})
	webDoc = getCanonicalDoc(t, ts, webSession.ID)
	postSessionAction(t, ts, webSession.ID, map[string]any{"actionId": "job.start", "viewId": webDoc.ViewID, "revision": webDoc.Revision, "appId": "network.ping"})
	jobs := getJobs(t, ts)
	if len(jobs) == 0 || jobs[0].SessionID != webSession.ID || jobs[0].State != "running" {
		t.Fatalf("web-like session did not start shared running job: %#v", jobs)
	}
	jobID := jobs[0].ID

	webDocAtJob := getCanonicalDoc(t, ts, webSession.ID)
	postSessionAction(t, ts, tuiSession.ID, map[string]any{"actionId": "job.select", "viewId": tuiDoc.ViewID, "revision": tuiDoc.Revision, "jobId": jobID})
	tuiDoc = getCanonicalDoc(t, ts, tuiSession.ID)
	if tuiDoc.SessionID != tuiSession.ID || tuiDoc.Revision <= 1 || !docHasKind(tuiDoc, "key_value") {
		t.Fatalf("tui-like session did not independently navigate to job view: %#v", tuiDoc)
	}
	webDocAfter := getCanonicalDoc(t, ts, webSession.ID)
	if webDocAfter.SessionID != webSession.ID || webDocAfter.ViewID != webDocAtJob.ViewID || webDocAfter.Revision != webDocAtJob.Revision {
		t.Fatalf("tui navigation changed web session: before=%#v after=%#v", webDocAtJob, webDocAfter)
	}

	postSessionAction(t, ts, tuiSession.ID, map[string]any{"actionId": "job.cancel", "viewId": tuiDoc.ViewID, "revision": tuiDoc.Revision, "jobId": jobID})
	jobs = waitJobState(t, ts, jobID, "canceled")

	r, err := http.Get(ts.URL + "/api/v1/events?after=0&replayOnly=1")
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	events := parseSSEEvents(t, r.Body)
	last := 0
	seenViewChanged, seenCancel := false, false
	for _, ev := range events {
		if ev.ID <= last {
			t.Fatalf("events not monotonic: %#v", events)
		}
		last = ev.ID
		if ev.Type == "view.changed" && (ev.SessionID == webSession.ID || ev.SessionID == tuiSession.ID) {
			seenViewChanged = true
		}
		if ev.Type == "job.cancel_requested" || ev.Type == "job.canceled" {
			seenCancel = true
		}
	}
	if !seenViewChanged || !seenCancel {
		t.Fatalf("missing expected events view.changed=%v cancel=%v events=%#v", seenViewChanged, seenCancel, events)
	}
}

func assertRevisionAdvanced(t *testing.T, label string, previous int, doc viewdoc.Document) {
	t.Helper()
	ValidateViewDocument(t, doc)
	if doc.Revision <= previous {
		t.Fatalf("%s: revision did not advance beyond %d: %#v", label, previous, doc)
	}
}

func formFieldDefaultEquals(doc viewdoc.Document, fieldID, want string) bool {
	for _, block := range doc.Blocks {
		for _, field := range block.Fields {
			if field.ID == fieldID && field.Default == want {
				return true
			}
		}
	}
	return false
}

func requireActionParams(t *testing.T, doc viewdoc.Document, actionID string) map[string]string {
	t.Helper()
	for _, action := range doc.Actions {
		if action.ID == actionID {
			if len(action.Params) == 0 {
				t.Fatalf("action %s missing params in doc %#v", actionID, doc)
			}
			return action.Params
		}
	}
	t.Fatalf("action %s not found in doc %#v", actionID, doc)
	return nil
}

func waitCanonicalJobState(t *testing.T, ts *httptest.Server, sessionID, jobID, want string) viewdoc.Document {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	var last viewdoc.Document
	for time.Now().Before(deadline) {
		last = getCanonicalDoc(t, ts, sessionID)
		if (jobID == "" || canonicalJobID(t, last) == jobID) && canonicalJobState(last) == want {
			return last
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("session %s did not observe job %q state %q, last doc=%#v", sessionID, jobID, want, last)
	return viewdoc.Document{}
}

func canonicalJobID(t *testing.T, doc viewdoc.Document) string {
	t.Helper()
	for _, block := range doc.Blocks {
		for _, pair := range block.Pairs {
			if pair.Key == "jobId" {
				return pair.Value
			}
		}
	}
	return ""
}

func canonicalJobState(doc viewdoc.Document) string {
	for _, block := range doc.Blocks {
		for _, pair := range block.Pairs {
			if pair.Key == "state" {
				return pair.Value
			}
		}
	}
	return ""
}

func assertArchitectureEvents(t *testing.T, events []Event, webSessionID, tuiSessionID, jobID string) {
	t.Helper()
	last := 0
	seenStarted, seenCanceled := false, false
	seenWebCanceledView, seenTUICanceledView := false, false
	for _, ev := range events {
		if ev.ID <= last {
			t.Fatalf("non-monotonic event id %d after %d in %#v", ev.ID, last, events)
		}
		last = ev.ID
		if ev.Job != nil && ev.Job.ID == jobID {
			switch ev.Type {
			case "job.started":
				seenStarted = true
			case "job.canceled":
				seenCanceled = true
			}
		}
		if ev.Type == "view.changed" && ev.Revision > 0 {
			if ev.SessionID == webSessionID {
				seenWebCanceledView = true
			}
			if ev.SessionID == tuiSessionID {
				seenTUICanceledView = true
			}
		}
	}
	if !seenStarted || !seenCanceled || !seenWebCanceledView || !seenTUICanceledView {
		t.Fatalf("missing architecture events for job %s: started=%v canceled=%v web_view=%v tui_view=%v events=%#v", jobID, seenStarted, seenCanceled, seenWebCanceledView, seenTUICanceledView, events)
	}
}

func getCanonicalDoc(t *testing.T, ts *httptest.Server, sessionID string) viewdoc.Document {
	t.Helper()
	resp, err := http.Get(ts.URL + "/api/v1/sessions/" + sessionID + "/view")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("view status %d", resp.StatusCode)
	}
	var doc viewdoc.Document
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		t.Fatal(err)
	}
	if doc.APIVersion != viewdoc.CanonicalAPIVersion || doc.SessionID != sessionID || doc.ViewID == "" || len(doc.Blocks) == 0 {
		t.Fatalf("invalid canonical document: %#v", doc)
	}
	ValidateViewDocument(t, doc)
	return doc
}

func docHasKind(doc viewdoc.Document, kind string) bool {
	for _, b := range doc.Blocks {
		if b.Kind == kind {
			return true
		}
	}
	return false
}

func postSessionAction(t *testing.T, ts *httptest.Server, sessionID string, payload map[string]any) {
	t.Helper()
	_ = postSessionActionDoc(t, ts, sessionID, payload)
}

func postSessionActionDoc(t *testing.T, ts *httptest.Server, sessionID string, payload map[string]any) viewdoc.Document {
	t.Helper()
	b, _ := json.Marshal(payload)
	resp, err := http.Post(ts.URL+"/api/v1/sessions/"+sessionID+"/actions", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("session action status %d body=%s", resp.StatusCode, body)
	}
	var doc viewdoc.Document
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		t.Fatal(err)
	}
	if doc.APIVersion != viewdoc.CanonicalAPIVersion || doc.SessionID != sessionID || doc.ViewID == "" || len(doc.Blocks) == 0 {
		t.Fatalf("session action did not return canonical ViewDocument: %#v", doc)
	}
	ValidateViewDocument(t, doc)
	return doc
}

func getJobs(t *testing.T, ts *httptest.Server) []Job {
	t.Helper()
	resp, err := http.Get(ts.URL + "/api/v1/jobs")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var jobs []Job
	if err := json.NewDecoder(resp.Body).Decode(&jobs); err != nil {
		t.Fatal(err)
	}
	return jobs
}

func waitJobState(t *testing.T, ts *httptest.Server, jobID, want string) []Job {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		jobs := getJobs(t, ts)
		for _, j := range jobs {
			if j.ID == jobID && j.State == want {
				return jobs
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("job %s did not reach %s: %#v", jobID, want, jobs)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func parseSSEEvents(t *testing.T, r io.Reader) []Event {
	t.Helper()
	s := bufio.NewScanner(r)
	var events []Event
	for s.Scan() {
		line := s.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		var ev Event
		if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &ev); err != nil {
			t.Fatal(err)
		}
		events = append(events, ev)
	}
	if err := s.Err(); err != nil {
		t.Fatal(err)
	}
	return events
}

func TestUnknownSessionAndSSEBoundedReset(t *testing.T) {
	ts := testServer()
	defer ts.Close()
	resp, _ := http.Get(ts.URL + "/api/v1/sessions/missing/view")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown session status %d", resp.StatusCode)
	}
	resp.Body.Close()
	srv := New(nil, nil)
	srv.firstEventID = 50
	req := httptest.NewRequest(http.MethodGet, "/api/v1/events", nil)
	req.Header.Set("Last-Event-ID", "1")
	w := httptest.NewRecorder()
	srv.sse(w, req)
	if !strings.Contains(w.Body.String(), "events.reset") {
		t.Fatalf("missing reset: %s", w.Body.String())
	}
}

func TestCanonicalSessionActionRequiresConcurrencyFields(t *testing.T) {
	ts := testServer()
	defer ts.Close()
	session := createSession(t, ts)
	cases := []struct{ name, body, field string }{
		{"missing action", `{"viewId":"ping-form","revision":1}`, "actionId"},
		{"missing view", `{"actionId":"form.update","revision":1}`, "viewId"},
		{"missing revision", `{"actionId":"form.update","viewId":"ping-form"}`, "revision"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := http.Post(ts.URL+"/api/v1/sessions/"+session.ID+"/actions", "application/json", strings.NewReader(tc.body))
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("want 400 got %d", resp.StatusCode)
			}
			var body map[string]map[string]any
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body["error"]["code"] != "missing_field" || body["error"]["field"] != tc.field {
				t.Fatalf("bad error %#v", body)
			}
		})
	}
}

func TestCanonicalSessionActionStaleConflictIncludesCurrentViewAndDoesNotMutate(t *testing.T) {
	ts := testServer()
	defer ts.Close()
	session := createSession(t, ts)
	doc := getDoc(t, ts, session.ID)
	beforeRevision := doc["revision"].(float64)
	resp, err := http.Post(ts.URL+"/api/v1/sessions/"+session.ID+"/actions", "application/json", strings.NewReader(`{"actionId":"form.update","viewId":"stale-view","revision":1,"values":{"target":"changed.example"}}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("want 409 got %d", resp.StatusCode)
	}
	var body map[string]map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["error"]["code"] != "stale_view" || body["error"]["viewId"] != doc["viewId"] || body["error"]["revision"] != beforeRevision {
		t.Fatalf("bad stale body %#v", body)
	}
	after := getDoc(t, ts, session.ID)
	if after["revision"] != beforeRevision {
		t.Fatalf("stale action mutated revision before=%v after=%v", beforeRevision, after["revision"])
	}
}

func TestOpenAPIDocumentIsExecutableForCanonicalRoutes(t *testing.T) {
	doc := readCheckedInOpenAPI(t)
	if got := doc["openapi"]; got != "3.1.0" {
		t.Fatalf("openapi version = %v, want 3.1.0", got)
	}
	paths := objectAt(t, doc, "paths")
	canonical := map[string][]string{
		"/api/v1/sessions":              {"post"},
		"/api/v1/sessions/{id}":         {"get", "delete"},
		"/api/v1/sessions/{id}/view":    {"get"},
		"/api/v1/sessions/{id}/actions": {"post"},
		"/api/v1/events":                {"get"},
	}
	for path, methods := range canonical {
		pathObj := objectValue(t, paths, path)
		for _, method := range methods {
			op := objectValue(t, pathObj, method)
			if stringValue(op, "operationId") == "" {
				t.Fatalf("%s %s missing operationId", method, path)
			}
			responses := objectValue(t, op, "responses")
			if _, ok := responses["200"]; !ok {
				t.Fatalf("%s %s missing 200 response", method, path)
			}
			if path != "/api/v1/events" {
				if _, ok := responses["404"]; !ok && path != "/api/v1/sessions" {
					t.Fatalf("%s %s missing 404 response", method, path)
				}
			}
			if _, ok := responses["500"]; !ok {
				t.Fatalf("%s %s missing 500 response", method, path)
			}
		}
	}
	components := objectAt(t, doc, "components")
	schemas := objectValue(t, components, "schemas")
	for _, name := range []string{"ErrorEnvelope", "Session", "ActionRequest", "Job", "EventPayload"} {
		if _, ok := schemas[name]; !ok {
			t.Fatalf("components.schemas missing %s", name)
		}
	}
	view := objectValue(t, objectValue(t, objectValue(t, objectValue(t, objectValue(t, paths, "/api/v1/sessions/{id}/view"), "get"), "responses"), "200"), "content")
	jsonContent := objectValue(t, view, "application/json")
	schema := objectValue(t, jsonContent, "schema")
	if ref := stringValue(schema, "$ref"); ref != "../schemas/view-document.json" {
		t.Fatalf("view response $ref = %q, want ../schemas/view-document.json", ref)
	}
	action := objectValue(t, objectValue(t, objectValue(t, objectValue(t, objectValue(t, paths, "/api/v1/sessions/{id}/actions"), "post"), "responses"), "200"), "content")
	actionJSONContent := objectValue(t, action, "application/json")
	actionSchema := objectValue(t, actionJSONContent, "schema")
	if ref := stringValue(actionSchema, "$ref"); ref != "../schemas/view-document.json" {
		t.Fatalf("session action response $ref = %q, want ../schemas/view-document.json", ref)
	}
	if _, ok := actionSchema["oneOf"]; ok {
		t.Fatalf("session action response schema must be exactly ViewDocument, got oneOf: %v", actionSchema["oneOf"])
	}
}

func TestOpenAPIViewDocumentExamplesValidateAgainstSchema(t *testing.T) {
	doc := readCheckedInOpenAPI(t)
	schema := compiledViewDocumentSchema(t)
	paths := objectAt(t, doc, "paths")
	validated := 0
	for path, pathValue := range paths {
		pathObj, ok := pathValue.(map[string]any)
		if !ok {
			t.Fatalf("path %s = %T, want object", path, pathValue)
		}
		for method, opValue := range pathObj {
			if !isOpenAPIMethod(method) {
				continue
			}
			op, ok := opValue.(map[string]any)
			if !ok {
				t.Fatalf("%s %s operation = %T, want object", method, path, opValue)
			}
			if requestBody, ok := op["requestBody"].(map[string]any); ok {
				if content, ok := requestBody["content"].(map[string]any); ok {
					validated += validateOpenAPIViewDocumentContentExamples(t, schema, content, fmt.Sprintf("%s %s request body", strings.ToUpper(method), path))
				}
			}
			responses, ok := op["responses"].(map[string]any)
			if !ok {
				continue
			}
			for status, responseValue := range responses {
				response, ok := responseValue.(map[string]any)
				if !ok {
					continue
				}
				content, ok := response["content"].(map[string]any)
				if !ok {
					continue
				}
				validated += validateOpenAPIViewDocumentContentExamples(t, schema, content, fmt.Sprintf("%s %s response %s", strings.ToUpper(method), path, status))
			}
		}
	}
	if validated == 0 {
		t.Fatal("validated 0 OpenAPI ViewDocument examples")
	}
}

func TestOpenAPIEndpointServesCheckedInDocumentSemantically(t *testing.T) {
	ts := testServer()
	defer ts.Close()
	resp, err := http.Get(ts.URL + "/api/v1/openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var served map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&served); err != nil {
		t.Fatal(err)
	}
	checkedIn := readCheckedInOpenAPI(t)
	if !sameStringKeys(objectAt(t, served, "paths"), objectAt(t, checkedIn, "paths")) {
		t.Fatalf("served paths differ from checked-in paths\nserved=%v\nchecked=%v", keys(objectAt(t, served, "paths")), keys(objectAt(t, checkedIn, "paths")))
	}
	if !sameStringKeys(objectValue(t, objectAt(t, served, "components"), "schemas"), objectValue(t, objectAt(t, checkedIn, "components"), "schemas")) {
		t.Fatalf("served component schemas differ from checked-in component schemas")
	}
}

func readCheckedInOpenAPI(t *testing.T) map[string]any {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "api", "openapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var raw any
	if err := yaml.Unmarshal(b, &raw); err != nil {
		t.Fatal(err)
	}
	doc, ok := normalizeYAML(raw).(map[string]any)
	if !ok {
		t.Fatalf("openapi root = %T, want object", raw)
	}
	return doc
}

func objectAt(t *testing.T, doc map[string]any, key string) map[string]any {
	t.Helper()
	return objectValue(t, doc, key)
}

func objectValue(t *testing.T, doc map[string]any, key string) map[string]any {
	t.Helper()
	v, ok := doc[key]
	if !ok {
		t.Fatalf("missing object key %q in %v", key, keys(doc))
	}
	m, ok := v.(map[string]any)
	if !ok {
		t.Fatalf("%q = %T, want object", key, v)
	}
	return m
}

func stringValue(doc map[string]any, key string) string {
	v, _ := doc[key].(string)
	return v
}

func validateOpenAPIViewDocumentContentExamples(t *testing.T, schema *jsonschema.Schema, content map[string]any, location string) int {
	t.Helper()
	validated := 0
	for mediaType, mediaValue := range content {
		media, ok := mediaValue.(map[string]any)
		if !ok {
			t.Fatalf("%s content %s = %T, want object", location, mediaType, mediaValue)
		}
		schemaObj, ok := media["schema"].(map[string]any)
		if !ok || stringValue(schemaObj, "$ref") != "../schemas/view-document.json" {
			continue
		}
		examples, ok := media["examples"].(map[string]any)
		if !ok || len(examples) == 0 {
			t.Fatalf("%s %s uses ViewDocument schema but has no examples", location, mediaType)
		}
		for exampleName, exampleValue := range examples {
			exampleObj, ok := exampleValue.(map[string]any)
			if !ok {
				t.Fatalf("%s %s example %s = %T, want object", location, mediaType, exampleName, exampleValue)
			}
			value, ok := exampleObj["value"]
			if !ok {
				t.Fatalf("%s %s example %s missing value", location, mediaType, exampleName)
			}
			raw, err := json.Marshal(value)
			if err != nil {
				t.Fatalf("marshal %s %s example %s: %v", location, mediaType, exampleName, err)
			}
			instance, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
			if err != nil {
				t.Fatalf("parse %s %s example %s JSON: %v", location, mediaType, exampleName, err)
			}
			if err := schema.Validate(instance); err != nil {
				t.Errorf("%s %s example %s failed ViewDocument schema validation: %v", location, mediaType, exampleName, err)
			}
			validated++
		}
	}
	return validated
}

func isOpenAPIMethod(method string) bool {
	switch method {
	case "get", "put", "post", "delete", "options", "head", "patch", "trace":
		return true
	default:
		return false
	}
}

func sameStringKeys(a, b map[string]any) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if _, ok := b[k]; !ok {
			return false
		}
	}
	return true
}

func keys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
