package api

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/flipperdevices/flipctl/internal/parsers"
	"github.com/flipperdevices/flipctl/internal/plugin"
	"github.com/flipperdevices/flipctl/internal/runner"
	"github.com/flipperdevices/flipctl/internal/viewdoc"
	"gopkg.in/yaml.v3"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Server struct {
	Apps          []plugin.App
	mu            sync.Mutex
	next          int
	jobs          map[string]Job
	cancels       map[string]context.CancelFunc
	events        []Event
	subs          map[chan Event]struct{}
	static        http.Handler
	sessionsStore map[string]*Session
	nextSession   int
	firstEventID  int
}
type Job struct {
	ID        string        `json:"id"`
	JobID     string        `json:"jobId"`
	AppID     string        `json:"appId"`
	SessionID string        `json:"sessionId,omitempty"`
	State     string        `json:"state"`
	Result    runner.Result `json:"result"`
	Created   string        `json:"created"`
	Updated   string        `json:"updated"`
}
type Session struct {
	ID            string            `json:"sessionId"`
	CurrentAppID  string            `json:"currentAppId,omitempty"`
	FormValues    map[string]string `json:"formValues,omitempty"`
	SelectedJobID string            `json:"selectedJobId,omitempty"`
	Revision      int               `json:"revision"`
}

type Event struct {
	ID        int    `json:"id"`
	Type      string `json:"type"`
	Job       *Job   `json:"job,omitempty"`
	Stream    string `json:"stream,omitempty"`
	Data      string `json:"data,omitempty"`
	Progress  any    `json:"progress,omitempty"`
	SessionID string `json:"sessionId,omitempty"`
	Revision  int    `json:"revision,omitempty"`
	Reset     bool   `json:"reset,omitempty"`
	Time      string `json:"time"`
}
type apiError struct {
	Error struct {
		Code    string `json:"code,omitempty"`
		Message string `json:"message,omitempty"`
		Field   string `json:"field,omitempty"`
	} `json:"error"`
}

func New(apps []plugin.App, static http.Handler) *Server {
	return &Server{Apps: apps, jobs: map[string]Job{}, cancels: map[string]context.CancelFunc{}, subs: map[chan Event]struct{}{}, static: static, sessionsStore: map[string]*Session{}, firstEventID: 1}
}
func (s *Server) Handler() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok\n")) })
	m.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		j(w, map[string]any{"ready": len(s.Apps) > 0, "apps": len(s.Apps)})
	})
	m.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		n := len(s.jobs)
		s.mu.Unlock()
		fmt.Fprintf(w, "flipctl_apps %d\nflipctl_jobs %d\n", len(s.Apps), n)
	})
	m.HandleFunc("/api/v1/meta", func(w http.ResponseWriter, r *http.Request) {
		j(w, map[string]any{"apiVersion": "flipctl/v1alpha1", "appId": "flipctl", "name": "FlipCTL", "time": time.Now().Format(time.RFC3339)})
	})
	m.HandleFunc("/api/v1/capabilities", func(w http.ResponseWriter, r *http.Request) {
		j(w, map[string]any{"apiVersion": "flipctl/v1alpha1", "capabilities": []string{"apps", "sessions", "view-document", "actions", "jobs", "cancel", "sse-replay"}})
	})
	m.HandleFunc("/api/v1/apps", func(w http.ResponseWriter, r *http.Request) { j(w, s.Apps) })
	m.HandleFunc("/api/v1/schema/view-document.json", func(w http.ResponseWriter, r *http.Request) { j(w, viewSchema()) })
	m.HandleFunc("/api/v1/openapi.json", func(w http.ResponseWriter, r *http.Request) { j(w, openapi()) })
	m.HandleFunc("/api/v1/schema", func(w http.ResponseWriter, r *http.Request) {
		j(w, map[string]any{"viewDocument": "v1alpha1", "nodes": []string{"form", "text", "kv", "log", "actions"}})
	})
	m.HandleFunc("/api/v1/sessions", s.sessions)
	m.HandleFunc("/api/v1/sessions/", s.sessionByID)
	m.HandleFunc("/api/v1/jobs", s.jobsList)
	m.HandleFunc("/api/v1/jobs/", s.jobByID)
	m.HandleFunc("/api/v1/events", s.sse)
	if s.static != nil {
		m.Handle("/", s.static)
	}
	return cors(m)
}
func (s *Server) sessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		w.WriteHeader(405)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if r.Method == http.MethodGet {
		arr := []Session{}
		for _, ss := range s.sessionsStore {
			arr = append(arr, *ss)
		}
		j(w, map[string]any{"apiVersion": "flipctl.session/v1alpha1", "sessions": arr})
		return
	}
	s.nextSession++
	id := fmt.Sprintf("s-%d", s.nextSession)
	ss := &Session{ID: id, CurrentAppID: "home", FormValues: map[string]string{}, Revision: 1}
	s.sessionsStore[id] = ss
	j(w, ss)
}
func (s *Server) sessionByID(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/sessions/"), "/")
	id := parts[0]
	if id == "" {
		writeErr(w, 400, "invalid_session", "missing sessionId", "")
		return
	}
	if len(parts) == 1 {
		s.mu.Lock()
		ss, ok := s.sessionsStore[id]
		if r.Method == http.MethodDelete {
			if ok {
				delete(s.sessionsStore, id)
			}
			s.mu.Unlock()
			if !ok {
				writeErr(w, 404, "session_not_found", "session not found", "sessionId")
			} else {
				j(w, map[string]any{"sessionId": id, "state": "closed"})
			}
			return
		}
		if !ok {
			s.mu.Unlock()
			writeErr(w, 404, "session_not_found", "session not found", "sessionId")
			return
		}
		cp := *ss
		s.mu.Unlock()
		j(w, cp)
		return
	}
	switch parts[1] {
	case "view":
		s.viewDoc(w, id)
	case "actions":
		s.action(w, r)
	default:
		http.NotFound(w, r)
	}
}
func (s *Server) viewDoc(w http.ResponseWriter, sessionID string) {
	s.mu.Lock()
	ss, ok := s.sessionsStore[sessionID]
	if !ok {
		s.mu.Unlock()
		writeErr(w, 404, "session_not_found", "session not found", "sessionId")
		return
	}
	doc := s.projectLocked(ss)
	s.mu.Unlock()
	j(w, doc)
}
func (s *Server) action(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(405)
		return
	}
	var req struct {
		AppID     string            `json:"appId"`
		AppIDOld  string            `json:"app_id"`
		SessionID string            `json:"sessionId"`
		ActionID  string            `json:"actionId"`
		ViewID    string            `json:"viewId"`
		Revision  *int              `json:"revision"`
		Fields    map[string]string `json:"fields"`
		Values    map[string]string `json:"values"`
		Params    map[string]string `json:"params"`
		JobID     string            `json:"jobId"`
	}
	if e := json.NewDecoder(r.Body).Decode(&req); e != nil {
		writeErr(w, 400, "bad_json", e.Error(), "")
		return
	}
	if req.AppID == "" {
		req.AppID = firstNonEmpty(req.AppIDOld, req.Params["appId"])
	}
	if req.JobID == "" {
		req.JobID = req.Params["jobId"]
	}
	if req.Fields == nil {
		req.Fields = req.Values
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/sessions/"), "/")
	canonicalSessionAction := strings.HasPrefix(r.URL.Path, "/api/v1/sessions/") && len(parts) >= 2 && parts[1] == "actions"
	var ss *Session
	if canonicalSessionAction {
		req.SessionID = parts[0]
		s.mu.Lock()
		var ok bool
		ss, ok = s.sessionsStore[req.SessionID]
		if !ok {
			s.mu.Unlock()
			writeErr(w, 404, "session_not_found", "session not found", "sessionId")
			return
		}
		curDoc := s.projectLocked(ss)
		if req.ActionID == "" {
			s.mu.Unlock()
			writeErr(w, 400, "missing_field", "actionId is required", "actionId")
			return
		}
		if req.ViewID == "" {
			s.mu.Unlock()
			writeErr(w, 400, "missing_field", "viewId is required", "viewId")
			return
		}
		if req.Revision == nil {
			s.mu.Unlock()
			writeErr(w, 400, "missing_field", "revision is required", "revision")
			return
		}
		if req.ViewID != curDoc.ViewID {
			s.mu.Unlock()
			writeErrDetails(w, 409, "stale_view", "viewId does not match current view", "viewId", map[string]any{"viewId": curDoc.ViewID, "revision": curDoc.Revision})
			return
		}
		if *req.Revision != ss.Revision {
			s.mu.Unlock()
			writeErrDetails(w, 409, "stale_view", "revision does not match current session", "revision", map[string]any{"viewId": curDoc.ViewID, "revision": curDoc.Revision})
			return
		}
		s.mu.Unlock()
	}
	if req.ActionID == "" {
		req.ActionID = "job.start"
	}
	if canonicalSessionAction {
		s.mu.Lock()
		handled := s.handleSessionActionLocked(ss, req.ActionID, req.AppID, req.Fields, req.JobID)
		s.mu.Unlock()
		if handled {
			s.viewDoc(w, req.SessionID)
			return
		}
	}
	if req.ActionID != "job.start" {
		writeErr(w, 400, "invalid_action", "unsupported action", "actionId")
		return
	}
	if req.AppID == "" && ss != nil {
		req.AppID = ss.CurrentAppID
	}
	if ss != nil && len(req.Fields) == 0 {
		req.Fields = copyMap(ss.FormValues)
	}
	a, ok := plugin.ByID(s.Apps, req.AppID)
	if !ok {
		writeErr(w, 404, "app_not_found", "app not found", "appId")
		return
	}
	if _, e := runner.Build(a, req.Fields); e != nil {
		writeErr(w, 400, "validation_failed", e.Error(), "fields")
		return
	}
	s.mu.Lock()
	s.next++
	id := "j-" + strconv.Itoa(s.next)
	ctx, cancel := context.WithCancel(context.Background())
	now := time.Now().Format(time.RFC3339)
	job := Job{ID: id, JobID: id, AppID: a.ID, SessionID: req.SessionID, State: "running", Created: now, Updated: now}
	s.jobs[id] = job
	s.cancels[id] = cancel
	if ss != nil {
		ss.SelectedJobID = id
		ss.CurrentAppID = a.ID
		ss.Revision++
		s.viewChangedLocked(ss)
	}
	s.evLocked("job.started", &job, "", "")
	s.mu.Unlock()
	go s.runJob(ctx, id, a, req.Fields)
	if canonicalSessionAction {
		s.viewDoc(w, req.SessionID)
		return
	}
	j(w, job)
}
func (s *Server) runJob(ctx context.Context, id string, a plugin.App, fields map[string]string) {
	cb := func(stream, chunk string) {
		s.mu.Lock()
		job := s.jobs[id]
		s.evLocked("job.output", &job, stream, chunk)
		for _, line := range strings.Split(chunk, "\n") {
			if strings.TrimSpace(line) == "" {
				continue
			}
			if progress := parsers.Progress(a.Command.Parser, line); progress != nil {
				s.evProgressLocked(&job, progress)
			}
		}
		s.mu.Unlock()
	}
	res, err := runner.RunWithCallback(ctx, a, fields, cb)
	res.Parsed = parsers.Parse(a.Command.Parser, res.Output)
	s.mu.Lock()
	defer s.mu.Unlock()
	job := s.jobs[id]
	job.Result = res
	job.Updated = time.Now().Format(time.RFC3339)
	delete(s.cancels, id)
	if job.State == "canceled" || ctx.Err() == context.Canceled {
		job.State = "canceled"
	} else if err != nil {
		job.State = "failed"
	} else {
		job.State = "done"
	}
	s.jobs[id] = job
	for _, ss := range s.sessionsStore {
		if ss.SelectedJobID == id {
			ss.Revision++
			s.viewChangedLocked(ss)
		}
	}
	s.evLocked("job."+job.State, &job, "", "")
}
func (s *Server) handleSessionActionLocked(ss *Session, actionID, appID string, vals map[string]string, jobID string) bool {
	switch actionID {
	case "app.open":
		if appID != "" {
			ss.CurrentAppID = appID
			ss.SelectedJobID = ""
			ss.Revision++
			s.viewChangedLocked(ss)
		}
		return true
	case "navigation.back":
		ss.SelectedJobID = ""
		ss.CurrentAppID = "home"
		ss.Revision++
		s.viewChangedLocked(ss)
		return true
	case "form.update":
		if ss.FormValues == nil {
			ss.FormValues = map[string]string{}
		}
		for k, v := range vals {
			ss.FormValues[k] = v
		}
		ss.Revision++
		s.viewChangedLocked(ss)
		return true
	case "job.select":
		ss.SelectedJobID = jobID
		ss.Revision++
		s.viewChangedLocked(ss)
		return true
	case "job.cancel":
		if jobID == "" {
			jobID = ss.SelectedJobID
		}
		if c := s.cancels[jobID]; c != nil {
			c()
		}
		ss.Revision++
		s.viewChangedLocked(ss)
		return true
	case "job.run_again":
		if jb, ok := s.jobs[ss.SelectedJobID]; ok {
			ss.CurrentAppID = jb.AppID
			ss.SelectedJobID = ""
			ss.Revision++
			s.viewChangedLocked(ss)
		}
		return true
	}
	return false
}

func (s *Server) projectLocked(ss *Session) viewdoc.Document {
	doc := viewdoc.Document{APIVersion: viewdoc.CanonicalAPIVersion, SessionID: ss.ID, Revision: ss.Revision, Title: "FlipCTL", Actions: []viewdoc.Action{{ID: "navigation.back", Label: "Back", Enabled: true}}}
	if ss.SelectedJobID != "" {
		if jb, ok := s.jobs[ss.SelectedJobID]; ok {
			return s.projectJobLocked(ss, jb)
		}
	}
	if ss.CurrentAppID == "" || ss.CurrentAppID == "home" {
		doc.ViewID = "home-app-list"
		doc.Title = "FlipCTL Apps"
		items := []viewdoc.ListItem{}
		for _, a := range s.Apps {
			items = append(items, viewdoc.ListItem{Label: a.Title, Value: a.ID, Description: a.Summary, Action: &viewdoc.Action{ID: "app.open", Label: "Open " + a.Title, Enabled: true, Params: map[string]string{"appId": a.ID}}})
		}
		doc.Blocks = []viewdoc.Block{{ID: "apps", Kind: "list", Title: "Apps", Items: items}}
		doc.Actions = nil
		return doc
	}
	a, ok := plugin.ByID(s.Apps, ss.CurrentAppID)
	if !ok {
		doc.ViewID = "app-not-found"
		doc.Blocks = []viewdoc.Block{{ID: "missing", Kind: "notice", Severity: "error", Text: "app not found"}}
		return doc
	}
	doc.ViewID = strings.TrimPrefix(a.ID, "network.") + "-form"
	doc.Title = a.Title
	fields := []viewdoc.FormField{}
	for _, f := range a.Fields {
		fields = append(fields, viewdoc.FormField{ID: f.ID, Label: f.Label, Type: f.Type, Default: firstNonEmpty(ss.FormValues[f.ID], f.Default), Required: f.Required, Validation: f.Pattern})
	}
	doc.Blocks = []viewdoc.Block{{ID: "summary", Kind: "notice", Severity: "info", Text: a.Summary}, {ID: "form", Kind: "form", Title: a.Title + " options", Fields: fields}}
	doc.Actions = []viewdoc.Action{{ID: "form.update", Label: "Update form", Enabled: true}, {ID: "job.start", Label: "Start job", Enabled: true, Role: "primary", Params: map[string]string{"appId": a.ID}}, {ID: "navigation.back", Label: "Back", Enabled: true}}
	return doc
}

func (s *Server) projectJobLocked(ss *Session, jb Job) viewdoc.Document {
	doc := viewdoc.Document{APIVersion: viewdoc.CanonicalAPIVersion, SessionID: ss.ID, Revision: ss.Revision, ViewID: strings.TrimPrefix(jb.AppID, "network.") + "-" + jb.State, Title: "Job " + jb.ID, Actions: []viewdoc.Action{{ID: "navigation.back", Label: "Back", Enabled: true}, {ID: "job.run_again", Label: "Run again", Enabled: true, Params: map[string]string{"jobId": jb.ID, "appId": jb.AppID}}}}
	doc.Blocks = []viewdoc.Block{{ID: "meta", Kind: "key_value", Title: "Job", Pairs: []viewdoc.KeyValuePair{{Key: "jobId", Value: jb.ID}, {Key: "appId", Value: jb.AppID}, {Key: "state", Value: jb.State}}}}
	if jb.State == "running" {
		doc.Blocks = append(doc.Blocks, viewdoc.Block{ID: "progress", Kind: "progress", Progress: &viewdoc.Progress{Label: "Running", Text: "Job is running"}})
		doc.Actions = append(doc.Actions, viewdoc.Action{ID: "job.cancel", Label: "Cancel", Enabled: true, Params: map[string]string{"jobId": jb.ID}})
	}
	lines := []viewdoc.LogLine{}
	if jb.Result.Output != "" {
		for i, l := range strings.Split(strings.TrimSpace(jb.Result.Output), "\n") {
			if l != "" {
				lines = append(lines, viewdoc.LogLine{ID: fmt.Sprintf("l-%d", i+1), Source: "stdout", Text: l})
			}
		}
	}
	if len(lines) == 0 && jb.State != "running" {
		lines = append(lines, viewdoc.LogLine{ID: "l-1", Source: "system", Text: "No output captured."})
	}
	if len(lines) > 0 {
		doc.Blocks = append(doc.Blocks, viewdoc.Block{ID: "log", Kind: "log", Title: "Output", Lines: lines})
	}
	if jb.State == "done" {
		doc.Blocks = append(doc.Blocks, viewdoc.Block{ID: "result", Kind: "table", Title: "Result", Columns: []viewdoc.TableColumn{{ID: "field", Label: "Field"}, {ID: "value", Label: "Value"}}, Rows: []viewdoc.TableRow{{Cells: map[string]string{"field": "error", "value": jb.Result.Error}}}})
	}
	if jb.State == "failed" || jb.State == "canceled" {
		doc.Blocks = append(doc.Blocks, viewdoc.Block{ID: "notice", Kind: "notice", Severity: "warning", Text: "Job " + jb.State})
	}
	return doc
}

func (s *Server) jobsList(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	arr := []Job{}
	for _, x := range s.jobs {
		arr = append(arr, x)
	}
	sortJobsNewestFirst(arr)
	j(w, arr)
}

func sortJobsNewestFirst(jobs []Job) {
	sort.SliceStable(jobs, func(i, j int) bool {
		ni, iok := numericJobID(jobs[i].ID)
		nj, jok := numericJobID(jobs[j].ID)
		if iok && jok && ni != nj {
			return ni > nj
		}
		if jobs[i].Created != jobs[j].Created {
			return jobs[i].Created > jobs[j].Created
		}
		return jobs[i].ID > jobs[j].ID
	})
}

func numericJobID(id string) (int, bool) {
	if !strings.HasPrefix(id, "j-") {
		return 0, false
	}
	n, err := strconv.Atoi(strings.TrimPrefix(id, "j-"))
	return n, err == nil
}
func (s *Server) jobByID(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/jobs/"), "/")
	id := parts[0]
	if len(parts) == 1 && r.Method == http.MethodGet {
		s.mu.Lock()
		job, ok := s.jobs[id]
		s.mu.Unlock()
		if !ok {
			writeErr(w, 404, "job_not_found", "job not found", "jobId")
			return
		}
		j(w, job)
		return
	}
	http.NotFound(w, r)
}
func (s *Server) sse(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store, no-transform")
	w.Header().Set("X-Accel-Buffering", "no")
	fl, _ := w.(http.Flusher)
	from, _ := strconv.Atoi(r.URL.Query().Get("after"))
	if from == 0 {
		from, _ = strconv.Atoi(r.URL.Query().Get("from"))
	}
	if h := r.Header.Get("Last-Event-ID"); from == 0 && h != "" {
		from, _ = strconv.Atoi(h)
	}
	ch := make(chan Event, 16)
	s.mu.Lock()
	if from > 0 && from < s.firstEventID {
		reset := Event{ID: s.firstEventID - 1, Type: "events.reset", Reset: true, Time: time.Now().Format(time.RFC3339)}
		s.mu.Unlock()
		b, _ := json.Marshal(reset)
		fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", reset.ID, reset.Type, b)
		return
	}
	evs := append([]Event(nil), s.events...)
	s.subs[ch] = struct{}{}
	s.mu.Unlock()
	defer func() { s.mu.Lock(); delete(s.subs, ch); s.mu.Unlock() }()
	if fl != nil {
		fmt.Fprint(w, ": ok\n\n")
		fl.Flush()
	}
	send := func(e Event) {
		b, _ := json.Marshal(e)
		fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", e.ID, e.Type, b)
		if fl != nil {
			fl.Flush()
		}
	}
	for _, e := range evs {
		if e.ID > from {
			send(e)
		}
	}
	if r.URL.Query().Get("replayOnly") == "1" {
		return
	}
	for {
		select {
		case e := <-ch:
			send(e)
		case <-r.Context().Done():
			return
		}
	}
}
func (s *Server) evLocked(t string, jb *Job, stream, data string) {
	s.appendEventLocked(Event{Type: t, Job: copyJob(jb), Stream: stream, Data: data, Time: time.Now().Format(time.RFC3339)})
}
func (s *Server) viewChangedLocked(ss *Session) {
	s.appendEventLocked(Event{Type: "view.changed", SessionID: ss.ID, Revision: ss.Revision, Time: time.Now().Format(time.RFC3339)})
}
func (s *Server) evProgressLocked(jb *Job, progress any) {
	s.appendEventLocked(Event{Type: "job.progress", Job: copyJob(jb), Progress: progress, Time: time.Now().Format(time.RFC3339)})
}
func (s *Server) appendEventLocked(e Event) {
	e.ID = s.firstEventID + len(s.events)
	s.events = append(s.events, e)
	const maxEvents = 100
	if len(s.events) > maxEvents {
		trim := len(s.events) - maxEvents
		s.events = append([]Event(nil), s.events[trim:]...)
		s.firstEventID += trim
	}
	for ch := range s.subs {
		select {
		case ch <- e:
		default:
		}
	}
	slog.Info("event", "type", e.Type, "id", e.ID)
}
func copyJob(jb *Job) *Job {
	cp := *jb
	return &cp
}
func copyMap(in map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range in {
		out[k] = v
	}
	return out
}
func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
func firstAppID(apps []plugin.App) string {
	if len(apps) > 0 {
		return apps[0].ID
	}
	return ""
}
func writeErr(w http.ResponseWriter, status int, code, msg, field string) {
	writeErrDetails(w, status, code, msg, field, nil)
}
func writeErrDetails(w http.ResponseWriter, status int, code, msg, field string, details map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	err := map[string]any{"code": code, "message": msg, "field": field}
	for k, v := range details {
		err[k] = v
	}
	json.NewEncoder(w).Encode(map[string]any{"error": err})
}
func j(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
func cors(n http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "content-type,last-event-id")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,DELETE,OPTIONS")
		if r.Method == "OPTIONS" {
			return
		}
		n.ServeHTTP(w, r)
	})
}
func viewSchema() any {
	return map[string]any{"$schema": "https://json-schema.org/draft/2020-12/schema", "title": "FlipCTL ViewDocument", "type": "object", "required": []string{"apiVersion", "sessionId", "viewId", "revision", "title", "blocks"}, "properties": map[string]any{"apiVersion": map[string]any{"const": viewdoc.CanonicalAPIVersion}, "sessionId": map[string]string{"type": "string"}, "viewId": map[string]string{"type": "string"}, "revision": map[string]string{"type": "integer"}, "title": map[string]string{"type": "string"}, "blocks": map[string]any{"type": "array", "items": map[string]any{"type": "object", "properties": map[string]any{"kind": map[string]any{"enum": []string{"text", "notice", "list", "form", "key_value", "table", "log", "progress"}}}}}, "actions": map[string]string{"type": "array"}}}
}
func openapi() any {
	doc, err := loadOpenAPI()
	if err != nil {
		return map[string]any{"openapi": "3.1.0", "info": map[string]string{"title": "FlipCTL canonical HTTP API", "version": "v1"}, "x-load-error": err.Error()}
	}
	return doc
}

func loadOpenAPI() (map[string]any, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return nil, fmt.Errorf("locate server source")
	}
	path := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "api", "openapi.yaml"))
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw any
	if err := yaml.Unmarshal(b, &raw); err != nil {
		return nil, err
	}
	doc, ok := normalizeYAML(raw).(map[string]any)
	if !ok {
		return nil, fmt.Errorf("openapi root is %T, want object", raw)
	}
	return doc, nil
}

func normalizeYAML(v any) any {
	switch x := v.(type) {
	case map[string]any:
		m := map[string]any{}
		for k, v := range x {
			m[k] = normalizeYAML(v)
		}
		return m
	case map[any]any:
		m := map[string]any{}
		for k, v := range x {
			m[fmt.Sprint(k)] = normalizeYAML(v)
		}
		return m
	case []any:
		for i, v := range x {
			x[i] = normalizeYAML(v)
		}
		return x
	default:
		return v
	}
}
