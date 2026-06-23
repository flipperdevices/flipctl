#!/usr/bin/env python3
import json, os, queue, shutil, subprocess, sys, threading, time, urllib.request
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
PKG = ROOT / 'docs/plans/2026-06-22-flipctl-finalization'
ART = PKG / 'artifacts' / 'real-tools-smoke'
LOGS = PKG / 'verification' / 'logs'
BASE = os.environ.get('REAL_TOOLS_BASE_URL', 'http://localhost:18081')
SERVICE = 'flipctld-real-tools'
PROJECT = os.environ.get('COMPOSE_PROJECT_NAME', 'flipctl-real-tools-smoke')
ENV = {**os.environ, 'COMPOSE_PROJECT_NAME': PROJECT}

ART.mkdir(parents=True, exist_ok=True); LOGS.mkdir(parents=True, exist_ok=True)
summary = []

def run(cmd, *, check=True, capture=True, **kw):
    print('+', ' '.join(cmd), flush=True)
    p = subprocess.run(cmd, cwd=ROOT, env=ENV, text=True, capture_output=capture, **kw)
    if capture:
        out = (p.stdout or '') + (p.stderr or '')
        print(out[-2000:], end='' if out.endswith('\n') or not out else '\n')
    if check and p.returncode != 0:
        raise SystemExit(f"command failed ({p.returncode}): {' '.join(cmd)}")
    return p

def http_json(path, data=None, timeout=10):
    body = None; headers = {}
    if data is not None:
        body = json.dumps(data).encode(); headers['Content-Type'] = 'application/json'
    req = urllib.request.Request(BASE + path, data=body, headers=headers)
    with urllib.request.urlopen(req, timeout=timeout) as r:
        return json.loads(r.read().decode())

def wait_ready():
    for _ in range(60):
        try:
            with urllib.request.urlopen(BASE + '/readyz', timeout=2) as r:
                if r.status == 200: return
        except Exception:
            time.sleep(1)
    raise SystemExit('real-tools daemon did not become ready on ' + BASE)

class SSE:
    def __init__(self):
        self.events=[]; self.q=queue.Queue(); self.stop=False
        self.thread=threading.Thread(target=self.run, daemon=True)
    def start(self): self.thread.start(); time.sleep(.5)
    def run(self):
        try:
            with urllib.request.urlopen(BASE + '/api/v1/events', timeout=120) as r:
                event=None; data=[]; eid=None
                for raw in r:
                    if self.stop: break
                    line=raw.decode(errors='replace').rstrip('\n')
                    if line.startswith('id:'): eid=int(line[3:].strip())
                    elif line.startswith('event:'): event=line[6:].strip()
                    elif line.startswith('data:'): data.append(line[5:].strip())
                    elif line == '' and event:
                        payload=json.loads('\n'.join(data)) if data else {}
                        rec={'id':eid,'event':event,'data':payload}
                        self.events.append(rec); self.q.put(rec)
                        event=None; data=[]; eid=None
        except Exception as e:
            if not self.stop: self.q.put({'error':str(e)})
    def wait_job(self, jobid, terminal_timeout=60):
        deadline=time.time()+terminal_timeout
        while time.time()<deadline:
            try: rec=self.q.get(timeout=.5)
            except queue.Empty: continue
            if 'error' in rec: raise RuntimeError(rec['error'])
            job=(rec.get('data') or {}).get('job') or {}
            if job.get('id')==jobid and rec.get('event') in ('job.done','job.failed','job.canceled'):
                return
        raise SystemExit('timed out waiting for terminal event for '+jobid)

def assert_monotonic(events):
    ids=[e['id'] for e in events if isinstance(e.get('id'), int)]
    if ids != sorted(ids) or len(ids) != len(set(ids)):
        raise SystemExit('SSE event IDs were not strictly monotonic/unique: '+repr(ids))

def job_events(events, jobid):
    return [e for e in events if ((e.get('data') or {}).get('job') or {}).get('id') == jobid]

def require_job_evidence(events, jobid, app, require_progress):
    evs=job_events(events, jobid); assert_monotonic(evs)
    names=[e['event'] for e in evs]
    if 'job.output' not in names: raise SystemExit(f'{app} missing job.output events: {names}')
    term_index=next((i for i,n in enumerate(names) if n in ('job.done','job.failed','job.canceled')), None)
    if term_index is None or 'job.output' not in names[:term_index]: raise SystemExit(f'{app} missing pre-terminal output: {names}')
    if require_progress and 'job.progress' not in names[:term_index]: raise SystemExit(f'{app} missing pre-terminal job.progress: {names}')
    return evs

def session_action(session_id, doc, action_id, **extra):
    payload={'actionId': action_id, 'viewId': doc['viewId'], 'revision': doc['revision']}
    payload.update(extra)
    return http_json(f'/api/v1/sessions/{session_id}/actions', payload)

def open_app(session_id, app_id):
    home=http_json(f'/api/v1/sessions/{session_id}/view')
    for block in home.get('blocks', []):
        for item in block.get('items', []):
            action=item.get('action') or {}
            params=action.get('params') or {}
            if action.get('id') == 'app.open' and params.get('appId') == app_id:
                session_action(session_id, home, action['id'], params=params)
                return http_json(f'/api/v1/sessions/{session_id}/view')
    raise SystemExit('canonical home list did not expose app.open for '+app_id)

def run_app(session_id, app_id, values):
    doc=open_app(session_id, app_id)
    doc=session_action(session_id, doc, 'form.update', values=values)
    job=session_action(session_id, doc, 'job.start')
    if not job.get('id'):
        raise SystemExit('canonical job.start response did not include job id')
    return job

def validate_and_render(session_id, app):
    doc=http_json(f'/api/v1/sessions/{session_id}/view')
    path=ART / f'{app}-canonical-session-view.json'
    path.write_text(json.dumps(doc, indent=2))
    run(['go','run','./scripts/validate-viewdocument','schemas/view-document.json',str(path)])
    out=ART / f'{app}-terminal-render.txt'
    p=run(['go','run','./cmd/flipctl-tui','--render-viewdoc',str(path),'--render-width','100','--render-height','40','--render-color','plain'], capture=True)
    out.write_text(p.stdout)
    if doc.get('viewId') not in p.stdout and doc.get('title') not in p.stdout:
        raise SystemExit(f'terminal renderer output for {app} does not match canonical session view')
    return path, out

try:
    host = shutil.which('nmap')
    (LOGS/'host-nmap.txt').write_text('host nmap: '+(host or 'absent; not required')+'\n')
    run(['docker','compose','--profile','real-tools','config'], stdout=open(LOGS/'compose-config.txt','w'), stderr=subprocess.STDOUT, capture=False)
    run(['docker','compose','--profile','real-tools','up','--build','-d',SERVICE])
    wait_ready()
    uid=run(['docker','compose','--profile','real-tools','exec','-T',SERVICE,'id','-u']).stdout.strip()
    if uid in ('0',''): raise SystemExit('daemon/container command user is root or unknown: '+uid)
    run(['docker','compose','--profile','real-tools','ps'])
    run(['docker','compose','--profile','real-tools','exec','-T',SERVICE,'ping','-n','-c1','-W1','ya.ru'])
    run(['docker','compose','--profile','real-tools','exec','-T',SERVICE,'nmap','--version'])
    s=SSE(); s.start()
    ping_session=http_json('/api/v1/sessions', {})['sessionId']
    ping=run_app(ping_session, 'network.ping', {'target':'ya.ru','count':'3'})
    s.wait_job(ping['id'], 30)
    nmap_session=http_json('/api/v1/sessions', {})['sessionId']
    nmap=run_app(nmap_session, 'network.nmap', {'target':'ya.ru'})
    s.wait_job(nmap['id'], 60); s.stop=True
    ping_job=http_json('/api/v1/jobs/'+ping['id']); nmap_job=http_json('/api/v1/jobs/'+nmap['id'])
    if ping_job.get('state')!='done' or nmap_job.get('state')!='done': raise SystemExit('real jobs did not finish done')
    require_job_evidence(s.events,ping['id'],'ping',True)
    nmap_evs=require_job_evidence(s.events,nmap['id'],'nmap',False)
    if not any(e['event']=='job.progress' for e in nmap_evs):
        summary.append('nmap produced no structured pre-terminal progress in fast internal scan; pre-terminal job.output was observed')
    validate_and_render(ping_session,'ping'); validate_and_render(nmap_session,'nmap')
    (ART/'real-tools-events.json').write_text(json.dumps(s.events, indent=2))
    (ART/'real-tools-jobs.json').write_text(json.dumps({'ping':ping_job,'nmap':nmap_job}, indent=2))
    (LOGS/'demo-real-tools-smoke-summary.txt').write_text('\n'.join(summary+[f'base={BASE}',f'ping={ping["id"]}',f'nmap={nmap["id"]}',f'pingSession={ping_session}',f'nmapSession={nmap_session}'])+'\n')
finally:
    run(['docker','compose','--profile','real-tools','down','--remove-orphans'], check=False)
