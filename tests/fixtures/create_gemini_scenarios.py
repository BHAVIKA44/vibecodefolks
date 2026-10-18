#!/usr/bin/env python3
"""Create fictional Git repositories for manual provider evaluation; no sample code is executed."""
import os
from pathlib import Path
import subprocess
import sys

SCENARIOS = {
    "authentication": {
        "base": {
            "frontend/login.js": "export function login(username, password) {\n  return fetch('/api/login', { method: 'POST' });\n}\n",
            "backend/auth.py": "def login(request):\n    return {'ok': False, 'error': 'not implemented'}, 501\n",
        },
        "changed": {
            "frontend/login.js": "export async function login(username, password) {\n  const response = await fetch('/api/login', {\n    method: 'POST',\n    headers: { 'Content-Type': 'application/json' },\n    body: JSON.stringify({ username, password })\n  });\n  return response.json();\n}\n",
            "backend/auth.py": "def login(request):\n    data = request.get_json()\n    if not data.get('username') or not data.get('password'):\n        return {'ok': False, 'error': 'credentials required'}, 400\n    # Fictional demo: this is not production authentication.\n    return {'ok': True, 'username': data['username']}, 200\n",
        },
    },
    "business-logic": {
        "base": {"discount.go": "package pricing\n\nfunc Discount(total int) int {\n\treturn total / 10\n}\n"},
        "changed": {"discount.go": "package pricing\n\nimport \"errors\"\n\n// Discount returns ten percent off orders of at least 100 cents.\nfunc Discount(total int) (int, error) {\n\tif total < 0 {\n\t\treturn 0, errors.New(\"total cannot be negative\")\n\t}\n\tif total < 100 {\n\t\treturn 0, nil\n\t}\n\treturn total / 10, nil\n}\n"},
    },
}


def write_files(root, files):
    for name, content in files.items():
        path = root / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content, encoding="utf-8", newline="\n")


def run_git(root, *args):
    env = dict(os.environ, GIT_CONFIG_NOSYSTEM="1", GIT_CONFIG_GLOBAL=os.devnull,
               GIT_AUTHOR_NAME="Fixture", GIT_AUTHOR_EMAIL="fixture@example.invalid",
               GIT_COMMITTER_NAME="Fixture", GIT_COMMITTER_EMAIL="fixture@example.invalid",
               GIT_AUTHOR_DATE="2026-01-01T00:00:00Z", GIT_COMMITTER_DATE="2026-01-01T00:00:00Z")
    subprocess.run(["git", *args], cwd=root, env=env, check=True, stdout=subprocess.DEVNULL)


def main():
    if len(sys.argv) != 2:
        raise SystemExit("Usage: python3 tests/fixtures/create_gemini_scenarios.py <new-directory>")
    root = Path(sys.argv[1]).resolve()
    root.mkdir(parents=True, exist_ok=False)
    for name, scenario in SCENARIOS.items():
        repo = root / name
        repo.mkdir()
        run_git(repo, "init", "-q")
        write_files(repo, scenario["base"])
        run_git(repo, "add", ".")
        run_git(repo, "commit", "-q", "-m", "Initial fictional code")
        write_files(repo, scenario["changed"])
        print(f"Created {repo}")


if __name__ == "__main__":
    main()
