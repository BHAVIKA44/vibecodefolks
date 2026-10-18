#!/usr/bin/env python3
"""Run a native release binary against a disposable repository without opening a browser."""
import json, os, pathlib, subprocess, sys, tempfile, urllib.request
binary = str(pathlib.Path(sys.argv[1]).resolve())
version = sys.argv[2]
assert subprocess.check_output([binary, '--version'], text=True).strip() == 'vibecodefolks ' + version
assert 'Usage:' in subprocess.run([binary, '--help'], capture_output=True, text=True).stderr
with tempfile.TemporaryDirectory() as tmp:
    env = dict(os.environ, GIT_CONFIG_NOSYSTEM='1', GIT_CONFIG_GLOBAL=os.devnull, GIT_AUTHOR_NAME='Fixture', GIT_AUTHOR_EMAIL='fixture@example.invalid', GIT_COMMITTER_NAME='Fixture', GIT_COMMITTER_EMAIL='fixture@example.invalid')
    subprocess.run(['git', 'init', '-q', tmp], check=True, env=env)
    pathlib.Path(tmp, 'example.go').write_text('package example\n')
    subprocess.run(['git', '-C', tmp, 'add', '.'], check=True, env=env)
    subprocess.run(['git', '-C', tmp, 'commit', '-qm', 'fixture'], check=True, env=env)
    pathlib.Path(tmp, 'example.go').write_text('package example\nvar Value = 1\n')
    p = subprocess.Popen([binary, '--no-open', tmp], stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, env=env)
    try:
        p.stdout.readline()
        url = p.stdout.readline().strip().removeprefix('Dashboard: ')
        with urllib.request.urlopen(url, timeout=20) as r:
            assert r.status == 200 and b'VibeCodeFolks' in r.read()
        with urllib.request.urlopen(url + '/api/changes', timeout=30) as r:
            assert json.load(r)['total_files'] == 1
        print('Native smoke test passed:', version)
    finally:
        p.terminate(); p.wait(timeout=10)
