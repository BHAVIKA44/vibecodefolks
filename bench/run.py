#!/usr/bin/env python3
"""Reproducible local benchmarks; runs no provider requests or fixture code."""
import argparse, json, os, pathlib, platform, shutil, statistics, subprocess, tempfile, threading, time, urllib.request
ROOT = pathlib.Path(__file__).resolve().parents[1]
parser = argparse.ArgumentParser()
parser.add_argument('--iterations', type=int, default=20)
parser.add_argument('--analysis-iterations', type=int, default=3)
parser.add_argument('--output', type=pathlib.Path, default=ROOT/'bench/results.json')
args = parser.parse_args()
if args.iterations < 1 or args.analysis_iterations < 1: parser.error('iterations must be positive')
def run(*cmd, **kw): return subprocess.run(cmd, check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, **kw).stdout.strip()
def rss(pid):
    try:
        if platform.system() == 'Linux':
            for line in pathlib.Path(f'/proc/{pid}/status').read_text().splitlines():
                if line.startswith('VmRSS:'): return int(line.split()[1])
            return None
        return int(run('ps', '-o', 'rss=', '-p', str(pid)).strip())
    except (OSError, ValueError, subprocess.CalledProcessError): return None
def stats(values):
    x=sorted(values); return {'n':len(x), 'min':x[0], 'p50':statistics.median(x), 'p95':x[max(0, __import__('math').ceil(len(x)*.95)-1)], 'max':x[-1]}
def request(url, method='GET'):
    req=urllib.request.Request(url, data=b'{}' if method=='POST' else None, method=method, headers={'Origin':url.split('/api/')[0], 'Content-Type':'application/json'})
    with urllib.request.urlopen(req, timeout=120) as response: return response.read()
def git(repo,*a): run('git','-C',str(repo),*a,env=fixture_env)
fixture_env=dict(os.environ,GIT_CONFIG_GLOBAL=os.devnull,GIT_CONFIG_NOSYSTEM='1',GIT_AUTHOR_NAME='Benchmark',GIT_AUTHOR_EMAIL='bench@example.invalid',GIT_COMMITTER_NAME='Benchmark',GIT_COMMITTER_EMAIL='bench@example.invalid',GIT_AUTHOR_DATE='2026-01-01T00:00:00Z',GIT_COMMITTER_DATE='2026-01-01T00:00:00Z')
with tempfile.TemporaryDirectory(prefix='vibecodefolks-bench-') as td:
    root=pathlib.Path(td); binary=root/'vibecodefolks'; probe=root/'probe'
    buildenv=dict(os.environ,CGO_ENABLED='0')
    run('go','build','-trimpath','-ldflags=-s -w','-o',str(binary),'.',cwd=ROOT,env=buildenv)
    run('go','build','-o',str(probe),'./bench/probe',cwd=ROOT,env=buildenv)
    result={'machine':{'os':platform.platform(),'architecture':platform.machine(),'cpu':platform.processor(),'go':run('go','version'),'git':run('git','--version')},'binary_bytes':binary.stat().st_size,'startup_iterations':args.iterations,'analysis_iterations':args.analysis_iterations,'fixtures':{}}
    for name,count,changed in [('small',100,5),('medium',1000,30),('large',10000,150)]:
        repo=root/name; repo.mkdir(); git(repo,'init','-q')
        (repo/'.gitignore').write_text('ignored/\n')
        paths=[]
        for i in range(count):
            path=repo/('src/d%03d'%(i//100))/('file_%05d.%s'%(i,('go','py','js','ts')[i%4])); path.parent.mkdir(parents=True,exist_ok=True); path.write_text('// fictional fixture %d\n'%i); paths.append(path)
        git(repo,'add','.');git(repo,'commit','-qm','baseline')
        paths[0].write_text(paths[0].read_text()+'// previous commit\n')
        git(repo,'add','--',str(paths[0].relative_to(repo))); git(repo,'commit','-qm','previous-change')
        for path in paths[:changed-1]: path.write_text(path.read_text()+'// changed\n')
        git(repo,'add','--',str(paths[0].relative_to(repo)))
        (repo/'untracked.py').write_text('# untracked fictional code\n')
        (repo/'ignored').mkdir(); (repo/'ignored'/'cache.txt').write_text('ignored\n')
        starts=[]; http=[]; rss_idle=[]; rss_changes=[]; rss_evidence=[]
        serverenv=dict(os.environ,VIBECODEFOLKS_AI_ENDPOINT='http://127.0.0.1:9/v1',VIBECODEFOLKS_AI_MODEL='fixture')
        for i in range(args.iterations):
            start=time.perf_counter(); p=subprocess.Popen([str(binary),'--no-open',str(repo)],stdout=subprocess.PIPE,stderr=subprocess.DEVNULL,text=True,env=serverenv)
            try:
                p.stdout.readline(); url=p.stdout.readline().strip().removeprefix('Dashboard: ')
                request(url); starts.append((time.perf_counter()-start)*1000)
                if i < args.analysis_iterations:
                    idle=rss(p.pid); rss_idle.append(idle)
                    for endpoint, dest, method in [('/api/changes',rss_changes,'GET'),('/api/explain/prepare',rss_evidence,'POST')]:
                        samples=[]; stop=threading.Event()
                        def monitor():
                            while not stop.wait(.05):
                                value=rss(p.pid)
                                if value is not None: samples.append(value)
                        thread=threading.Thread(target=monitor); thread.start(); t=time.perf_counter()
                        try: request(url+endpoint,method)
                        finally: stop.set();thread.join()
                        elapsed=(time.perf_counter()-t)*1000
                        if endpoint=='/api/changes': http.append(elapsed)
                        if idle is not None: samples.append(idle)
                        dest.append(max(samples) if samples else None)
            finally:
                p.terminate();p.wait(timeout=10)
        probes=[json.loads(run(str(probe),str(repo))) for _ in range(args.analysis_iterations)]
        git(repo,'reset','--hard','-q','HEAD'); (repo/'untracked.py').unlink()
        clean=json.loads(run(str(probe),str(repo)))
        result['fixtures'][name]={'tracked_files':count+1,'changed_files':changed,'startup_ms':stats(starts),'http_changes_ms':stats(http),'probe':{key:stats([v[key]['ms'] for v in probes]) for key in ('discovery','analysis','evidence')},'allocation_bytes_median':{key:statistics.median([v[key]['allocated_bytes'] for v in probes]) for key in ('discovery','analysis','evidence')},'allocations_median':{key:statistics.median([v[key]['allocations'] for v in probes]) for key in ('discovery','analysis','evidence')},'server_rss_kib_idle':rss_idle,'server_rss_kib_changes_peak_sampled':rss_changes,'server_rss_kib_evidence_peak_sampled':rss_evidence,'clean_probe':clean}
        print(name, 'complete', flush=True)
args.output.parent.mkdir(parents=True,exist_ok=True);args.output.write_text(json.dumps(result,indent=2)+'\n')
lines=['# Benchmark results','',f"{result['machine']['os']} · {result['machine']['architecture']} · {result['machine']['go']} · {result['machine']['git']}",'',f"Release-like binary: {result['binary_bytes']} bytes. Startup samples: {args.iterations}; analysis samples: {args.analysis_iterations}.",'','| Fixture | Files / changes | Startup p50 / p95 ms | Discovery p50 ms | Analyze p50 ms | Evidence p50 ms | HTTP changes p50 ms |','| --- | --- | --- | --- | --- | --- | --- |']
for name,f in result['fixtures'].items():
    lines.append(f"| {name} | {f['tracked_files']} / {f['changed_files']} | {f['startup_ms']['p50']:.1f} / {f['startup_ms']['p95']:.1f} | {f['probe']['discovery']['p50']:.1f} | {f['probe']['analysis']['p50']:.1f} | {f['probe']['evidence']['p50']:.1f} | {f['http_changes_ms']['p50']:.1f} |")
args.output.with_suffix('.md').write_text('\n'.join(lines)+'\n')
print('Results:',args.output)
