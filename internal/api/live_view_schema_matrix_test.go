package api

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/flipperdevices/flipctl/internal/plugin"
	"github.com/flipperdevices/flipctl/internal/viewdoc"
)

func TestValidateViewDocumentReportsInvalidProjectorOutput(t *testing.T) {
	bad := viewdoc.Document{
		APIVersion: viewdoc.CanonicalAPIVersion,
		SessionID:  "s-test",
		ViewID:     "bad-live-output",
		Revision:   1,
		Title:      "Bad",
		Blocks:     []viewdoc.Block{{ID: "mixed", Kind: "text", Text: "hello", Rows: []viewdoc.TableRow{{Cells: map[string]string{"x": "y"}}}}},
	}
	if err := validateViewDocument(bad); err == nil {
		t.Fatalf("invalid mixed text/table projector output unexpectedly validated")
	} else if !strings.Contains(err.Error(), "schema validation failed") {
		t.Fatalf("negative validation error should identify schema validation failure, got: %v", err)
	}
}

func TestLiveProjectorViewDocumentsValidateAgainstSchemaMatrix(t *testing.T) {
	ts := testServer()
	defer ts.Close()

	session := createSession(t, ts)
	home := getCanonicalDoc(t, ts, session.ID)
	assertViewState(t, home, "home", "home-app-list", []string{"list"})

	pingForm := openAppAndValidate(t, ts, session.ID, home, "network.ping", "ping-form")
	assertViewState(t, pingForm, "ping form", "ping-form", []string{"notice", "form"})
	slowPingTS := httptest.NewServer(New(appsWithSlowPing(t), nil).Handler())
	defer slowPingTS.Close()
	pingRunningSession := createSession(t, slowPingTS)
	pingRunningHome := getCanonicalDoc(t, slowPingTS, pingRunningSession.ID)
	pingRunningForm := openAppAndValidate(t, slowPingTS, pingRunningSession.ID, pingRunningHome, "network.ping", "ping-form")
	pingRunningJob := startJobAndValidate(t, slowPingTS, pingRunningSession.ID, pingRunningForm, "network.ping", "ping-running")
	_ = pingRunningJob
	pingRunning := getCanonicalDoc(t, slowPingTS, pingRunningSession.ID)
	assertViewState(t, pingRunning, "ping running", "ping-running", []string{"key_value", "progress"})

	pingSuccessSession := createSession(t, ts)
	pingSuccessHome := getCanonicalDoc(t, ts, pingSuccessSession.ID)
	pingSuccessForm := openAppAndValidate(t, ts, pingSuccessSession.ID, pingSuccessHome, "network.ping", "ping-form")
	pingSuccessJob := startJobWithoutRunningViewAssertion(t, ts, pingSuccessSession.ID, pingSuccessForm, "network.ping")
	waitJobState(t, ts, pingSuccessJob.ID, "done")
	pingSuccess := getCanonicalDoc(t, ts, pingSuccessSession.ID)
	assertViewState(t, pingSuccess, "ping success", "ping-done", []string{"key_value", "log", "table"})

	back := pingSuccess
	postSessionAction(t, ts, pingSuccessSession.ID, map[string]any{"actionId": "navigation.back", "viewId": back.ViewID, "revision": back.Revision})
	home = getCanonicalDoc(t, ts, pingSuccessSession.ID)
	nmapForm := openAppAndValidate(t, ts, pingSuccessSession.ID, home, "network.nmap", "nmap-form")
	assertViewState(t, nmapForm, "nmap form", "nmap-form", []string{"notice", "form"})
	nmapSuccessJob := startJobWithoutRunningViewAssertion(t, ts, pingSuccessSession.ID, nmapForm, "network.nmap")
	waitJobState(t, ts, nmapSuccessJob.ID, "done")
	nmapSuccess := getCanonicalDoc(t, ts, pingSuccessSession.ID)
	assertViewState(t, nmapSuccess, "nmap success", "nmap-done", []string{"key_value", "log", "table"})

	slowNmapTS := httptest.NewServer(New(appsWithSlowNmap(t), nil).Handler())
	defer slowNmapTS.Close()
	nmapRunningSession := createSession(t, slowNmapTS)
	nmapRunningHome := getCanonicalDoc(t, slowNmapTS, nmapRunningSession.ID)
	nmapRunningForm := openAppAndValidate(t, slowNmapTS, nmapRunningSession.ID, nmapRunningHome, "network.nmap", "nmap-form")
	nmapRunningJob := startJobAndValidate(t, slowNmapTS, nmapRunningSession.ID, nmapRunningForm, "network.nmap", "nmap-running")
	_ = nmapRunningJob
	nmapRunning := getCanonicalDoc(t, slowNmapTS, nmapRunningSession.ID)
	assertViewState(t, nmapRunning, "nmap running", "nmap-running", []string{"key_value", "progress"})

	historySession := createSession(t, ts)
	historyHome := getCanonicalDoc(t, ts, historySession.ID)
	postSessionAction(t, ts, historySession.ID, map[string]any{"actionId": "job.select", "viewId": historyHome.ViewID, "revision": historyHome.Revision, "jobId": pingSuccessJob.ID})
	historyDoc := getCanonicalDoc(t, ts, historySession.ID)
	assertViewState(t, historyDoc, "job history select", "ping-done", []string{"key_value", "log", "table"})
}

func TestLiveProjectorFailureViewDocumentsValidateAgainstSchema(t *testing.T) {
	ts := httptest.NewServer(New(appsWithFailingCommands(t), nil).Handler())
	defer ts.Close()

	pingSession := createSession(t, ts)
	pingHome := getCanonicalDoc(t, ts, pingSession.ID)
	pingForm := openAppAndValidate(t, ts, pingSession.ID, pingHome, "network.ping", "ping-form")
	pingJob := startJobWithoutRunningViewAssertion(t, ts, pingSession.ID, pingForm, "network.ping")
	waitJobState(t, ts, pingJob.ID, "failed")
	pingFailure := getCanonicalDoc(t, ts, pingSession.ID)
	assertViewState(t, pingFailure, "ping failure", "ping-failed", []string{"key_value", "log", "notice"})

	nmapSession := createSession(t, ts)
	nmapHome := getCanonicalDoc(t, ts, nmapSession.ID)
	nmapForm := openAppAndValidate(t, ts, nmapSession.ID, nmapHome, "network.nmap", "nmap-form")
	nmapJob := startJobWithoutRunningViewAssertion(t, ts, nmapSession.ID, nmapForm, "network.nmap")
	waitJobState(t, ts, nmapJob.ID, "failed")
	nmapFailure := getCanonicalDoc(t, ts, nmapSession.ID)
	assertViewState(t, nmapFailure, "nmap failure", "nmap-failed", []string{"key_value", "log", "notice"})
}

func openAppAndValidate(t *testing.T, ts *httptest.Server, sessionID string, doc viewdoc.Document, appID, wantViewID string) viewdoc.Document {
	t.Helper()
	postSessionAction(t, ts, sessionID, map[string]any{"actionId": "app.open", "viewId": doc.ViewID, "revision": doc.Revision, "appId": appID})
	next := getCanonicalDoc(t, ts, sessionID)
	if next.ViewID != wantViewID {
		t.Fatalf("open %s: want viewID %s, got %s", appID, wantViewID, next.ViewID)
	}
	return next
}

func startJobAndValidate(t *testing.T, ts *httptest.Server, sessionID string, doc viewdoc.Document, appID, wantRunningViewID string) Job {
	t.Helper()
	postSessionAction(t, ts, sessionID, map[string]any{"actionId": "job.start", "viewId": doc.ViewID, "revision": doc.Revision, "appId": appID})
	jobs := getJobs(t, ts)
	if len(jobs) == 0 || jobs[0].AppID != appID {
		t.Fatalf("job.start %s did not create expected latest job: %#v", appID, jobs)
	}
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		running := getCanonicalDoc(t, ts, sessionID)
		if running.ViewID == wantRunningViewID {
			return jobs[0]
		}
		time.Sleep(10 * time.Millisecond)
	}
	running := getCanonicalDoc(t, ts, sessionID)
	t.Fatalf("job.start %s: want running viewID %s, got %s", appID, wantRunningViewID, running.ViewID)
	return Job{}
}

func startJobWithoutRunningViewAssertion(t *testing.T, ts *httptest.Server, sessionID string, doc viewdoc.Document, appID string) Job {
	t.Helper()
	postSessionAction(t, ts, sessionID, map[string]any{"actionId": "job.start", "viewId": doc.ViewID, "revision": doc.Revision, "appId": appID})
	jobs := getJobs(t, ts)
	if len(jobs) == 0 || jobs[0].AppID != appID {
		t.Fatalf("job.start %s did not create expected latest job: %#v", appID, jobs)
	}
	return jobs[0]
}

func assertViewState(t *testing.T, doc viewdoc.Document, label, wantViewID string, kinds []string) {
	t.Helper()
	ValidateViewDocument(t, doc)
	if doc.ViewID != wantViewID {
		t.Fatalf("%s: want viewID %s, got %s", label, wantViewID, doc.ViewID)
	}
	for _, kind := range kinds {
		if !docHasKind(doc, kind) {
			t.Fatalf("%s: missing %s block in %#v", label, kind, doc.Blocks)
		}
	}
}

func appsWithSlowPing(t *testing.T) []plugin.App {
	t.Helper()
	return appsWithCommandOverride(t, "network.ping", "sleep 5")
}

func appsWithSlowNmap(t *testing.T) []plugin.App {
	t.Helper()
	return appsWithCommandOverride(t, "network.nmap", "sleep 5")
}

func appsWithScriptedPingOutput(t *testing.T) []plugin.App {
	t.Helper()
	return appsWithCommandOverride(t, "network.ping", strings.Join([]string{
		`printf '%s\n' 'PING 127.0.0.1 (127.0.0.1): 56 data bytes'`,
		`printf '%s\n' '64 bytes from 127.0.0.1: icmp_seq=1 ttl=64 time=0.1 ms'`,
		`printf '%s\n' '--- 127.0.0.1 ping statistics ---'`,
		`printf '%s\n' '1 packets transmitted, 1 packets received, 0% packet loss'`,
	}, "; "))
}

func appsWithCommandOverride(t *testing.T, appID, script string) []plugin.App {
	t.Helper()
	apps := loadPluginAppsForMatrix(t)
	for i := range apps {
		if apps[i].ID == appID {
			apps[i].Command.Executable = "/bin/sh"
			apps[i].Command.Args = []string{"-c", script}
		}
	}
	return apps
}

func appsWithFailingCommands(t *testing.T) []plugin.App {
	t.Helper()
	apps := loadPluginAppsForMatrix(t)
	for i := range apps {
		apps[i].Command.Executable = "/bin/sh"
		apps[i].Command.Args = []string{"-c", "echo simulated failure; exit 7"}
	}
	return apps
}

func loadPluginAppsForMatrix(t *testing.T) []plugin.App {
	t.Helper()
	apps, err := loadTestPluginApps("plugins")
	if err != nil {
		t.Fatal(err)
	}
	return apps
}
