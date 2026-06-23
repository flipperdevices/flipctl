#!/usr/bin/env python3
import argparse, json, os, platform, subprocess, sys, time, urllib.request
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
PKG = ROOT / 'docs/plans/2026-06-22-flipctl-finalization'
ART = PKG / 'artifacts' / 'docker-userspace-smoke'
LOGS = PKG / 'verification' / 'logs'
ART.mkdir(parents=True, exist_ok=True); LOGS.mkdir(parents=True, exist_ok=True)

def run(cmd, *, env=None, check=True, capture=True, log=None):
    print('+', ' '.join(cmd), flush=True)
    p = subprocess.run(cmd, cwd=ROOT, env=env or os.environ, text=True, capture_output=capture)
    out = (p.stdout or '') + (p.stderr or '')
    if log:
        Path(log).write_text(out)
    if capture and out:
        print(out[-2000:], end='' if out.endswith('\n') else '\n')
    if check and p.returncode != 0:
        raise SystemExit(f"command failed ({p.returncode}): {' '.join(cmd)}")
    return p

def http_json(base, path, data=None, timeout=10):
    body = None; headers = {}
    if data is not None:
        body = json.dumps(data).encode(); headers['Content-Type'] = 'application/json'
    req = urllib.request.Request(base + path, data=body, headers=headers)
    with urllib.request.urlopen(req, timeout=timeout) as r:
        return json.loads(r.read().decode())

def http_ok(base, path, timeout=10):
    with urllib.request.urlopen(base + path, timeout=timeout) as r:
        return r.status, r.headers.get('content-type','')

def wait_ready(base):
    last = None
    for _ in range(90):
        try:
            with urllib.request.urlopen(base + '/readyz', timeout=2) as r:
                if r.status == 200: return
        except Exception as e:
            last = e; time.sleep(1)
    raise SystemExit(f'daemon did not become ready on {base}: {last}')

def session_action(base, session_id, doc, action_id, **extra):
    payload={'actionId': action_id, 'viewId': doc['viewId'], 'revision': doc['revision']}
    payload.update(extra)
    return http_json(base, f'/api/v1/sessions/{session_id}/actions', payload)

def view_action_param(doc, action_id, param_name):
    for action in doc.get('actions', []):
        if action.get('id') == action_id:
            params = action.get('params') or {}
            value = params.get(param_name)
            if value:
                return value
    return ''

def view_key_value(doc, key):
    for block in doc.get('blocks', []):
        for pair in block.get('pairs', []):
            if pair.get('key') == key and pair.get('value'):
                return pair['value']
    return ''

def started_job_id(doc):
    return (
        view_action_param(doc, 'job.cancel', 'jobId')
        or view_action_param(doc, 'job.run_again', 'jobId')
        or view_key_value(doc, 'jobId')
    )

def open_ping(base, session_id):
    home=http_json(base, f'/api/v1/sessions/{session_id}/view')
    for block in home.get('blocks', []):
        for item in block.get('items', []):
            action=item.get('action') or {}; params=action.get('params') or {}
            if action.get('id') == 'app.open' and params.get('appId') == 'network.ping':
                session_action(base, session_id, home, action['id'], params=params)
                return http_json(base, f'/api/v1/sessions/{session_id}/view')
    raise SystemExit('canonical home list did not expose app.open for network.ping')

def run_ping_flow(base, name, platform_name):
    ready = http_json(base, '/api/v1/meta')
    apps = http_json(base, '/api/v1/apps')
    index_status, index_type = http_ok(base, '/')
    sess = http_json(base, '/api/v1/sessions', {})['sessionId']
    initial = http_json(base, f'/api/v1/sessions/{sess}/view')
    ping_doc = open_ping(base, sess)
    updated = session_action(base, sess, ping_doc, 'form.update', values={'target':'127.0.0.1','count':'2'})
    running_doc = session_action(base, sess, updated, 'job.start')
    jid = started_job_id(running_doc)
    if not jid: raise SystemExit('job.start ViewDocument did not expose job id via semantic action params or key_value block')
    final_job = None
    for _ in range(60):
        final_job = http_json(base, '/api/v1/jobs/'+jid)
        if final_job.get('state') in ('done','failed','canceled'): break
        time.sleep(1)
    if final_job.get('state') != 'done':
        raise SystemExit(f'fake ping job did not finish done: {final_job}')
    final = http_json(base, f'/api/v1/sessions/{sess}/view')
    for label, doc in [('initial', initial), ('final', final)]:
        p = ART / f'{name}-{label}-canonical-session-view.json'
        p.write_text(json.dumps(doc, indent=2))
        run(['go','run','./scripts/validate-viewdocument','schemas/view-document.json',str(p)], log=LOGS/f'{name}-{label}-validate.log')
    summary = {
        'name': name, 'platform': platform_name, 'hostMachine': platform.machine(),
        'pythonPlatform': platform.platform(), 'baseURL': base, 'sessionId': sess,
        'jobId': jid, 'jobState': final_job.get('state'), 'webIndexStatus': index_status,
        'webIndexContentType': index_type, 'appsCount': len(apps) if isinstance(apps, list) else len(apps.get('apps', [])),
        'daemonMeta': ready,
    }
    (ART / f'{name}-summary.json').write_text(json.dumps(summary, indent=2))
    (LOGS / f'{name}-summary.txt').write_text('\n'.join(f'{k}={v}' for k,v in summary.items())+'\n')

def compose_mode():
    base='http://localhost:8080'; project=os.environ.get('COMPOSE_PROJECT_NAME','flipctl-docker-fake-smoke')
    env={**os.environ,'COMPOSE_PROJECT_NAME':project}
    try:
        run(['docker','compose','config'], env=env, log=LOGS/'docker-fake-compose-config.txt')
        run(['docker','compose','up','--build','-d','flipctld'], env=env, log=LOGS/'docker-fake-up.log')
        wait_ready(base)
        uid=run(['docker','compose','exec','-T','flipctld','id','-u'], env=env).stdout.strip()
        if uid in ('0',''): raise SystemExit('fake smoke container user is root or unknown: '+uid)
        run_ping_flow(base, 'docker-fake-smoke', 'linux/amd64-or-host-docker')
    finally:
        run(['docker','compose','down','--remove-orphans'], env=env, check=False, log=LOGS/'docker-fake-down.log')

def arm64_mode():
    base='http://localhost:18082'; image=os.environ.get('ARM64_SMOKE_IMAGE','flipctl:e2-arm64-userspace')
    cidfile=LOGS/'arm64-container.cid'
    run(['docker','buildx','version'], log=LOGS/'arm64-buildx-version.txt')
    run(['docker','run','--rm','--platform','linux/arm64','debian:bookworm-slim','uname','-m'], log=LOGS/'arm64-qemu-preflight.log')
    run(['docker','buildx','build','--platform','linux/arm64','--target','runtime','--load','-t',image,'.'], log=LOGS/'arm64-build.log')
    cid=''
    try:
        p=run(['docker','run','-d','--rm','--platform','linux/arm64','-p','18082:8080',image], log=LOGS/'arm64-run.log')
        cid=p.stdout.strip(); cidfile.write_text(cid+'\n')
        wait_ready(base)
        inspect=run(['docker','inspect',cid], log=LOGS/'arm64-inspect.json').stdout
        run(['docker','exec',cid,'id','-u'], log=LOGS/'arm64-id-u.txt')
        run_ping_flow(base, 'arm64-userspace-smoke', 'linux/arm64')
        (ART/'arm64-platform.json').write_text(inspect)
    finally:
        if cid:
            run(['docker','stop',cid], check=False, log=LOGS/'arm64-stop.log')

def main():
    ap=argparse.ArgumentParser()
    ap.add_argument('mode', choices=['docker-fake','arm64'])
    args=ap.parse_args()
    if args.mode=='docker-fake': compose_mode()
    else: arm64_mode()
if __name__ == '__main__': main()
