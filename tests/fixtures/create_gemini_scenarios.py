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
    "mixed-checkout": {
        "base": {
            "README.md": "# Sample shop\nA fictional store for testing.\n",
            "api/checkout.py": "def checkout(data):\n    return {'total': sum(data['items'])}, 200\n",
            "api/old_cart.py": "def legacy_checkout(data):\n    return {'total': 0}\n",
            "web/cart.js": "export function checkout(items) {\n  return fetch('/api/checkout', { method: 'POST', body: JSON.stringify({ items }) });\n}\n",
        },
        "changed": {
            "README.md": "# Sample shop\nA fictional store for testing.\n\nSupport hours: weekdays.\n",
            "api/checkout.py": "def checkout(data):\n    items = data.get('items', [])\n    if not items:\n        return {'error': 'cart is empty'}, 400\n    total = sum(items)\n    if data.get('coupon') == 'SAVE10':\n        total = total * 90 // 100\n    return {'total': total}, 200\n",
            "web/cart.js": "export async function checkout(items, coupon) {\n  const response = await fetch('/api/checkout', {\n    method: 'POST', headers: { 'Content-Type': 'application/json' },\n    body: JSON.stringify({ items, coupon })\n  });\n  return response.json();\n}\n",
        },
        "removed": ["api/old_cart.py"],
    },
}


SCENARIOS.update({'react-search': {'base': {'Search.jsx': 'export function Results({ items }) {\n'
                                         '  return items.map(item => <li key={item.id}>{item.name}</li>);\n'
                                         '}\n'},
                  'changed': {'Search.jsx': 'export function Results({ items, query }) {\n'
                                            '  const term = query.trim().toLowerCase();\n'
                                            '  const shown = items.filter(item => '
                                            'item.name.toLowerCase().includes(term));\n'
                                            '  return shown.map(item => <li '
                                            'key={item.id}>{item.name}</li>);\n'
                                            '}\n'}},
 'cursor-pagination': {'base': {'api.py': 'def page(rows):\n    return rows[:20]\n'},
                       'changed': {'api.py': 'def page(rows, after, limit=20):\n'
                                             "    matches = [row for row in rows if row['id'] > after]\n"
                                             '    batch = matches[:limit]\n'
                                             "    cursor = batch[-1]['id'] if len(matches) > limit else "
                                             'None\n'
                                             '    return batch, cursor\n'}},
 'zero-stock': {'base': {'stock.js': 'export function stock(inventory, sku) {\n'
                                     '  return inventory[sku] || 5;\n'
                                     '}\n'},
                'changed': {'stock.js': 'export function stock(inventory, sku) {\n'
                                        '  return inventory[sku] ?? 5;\n'
                                        '}\n'}},
 'tenant-cache': {'base': {'profiles.py': 'def profile(db, tenant, user):\n'
                                          '    return db.load_profile(tenant, user)\n'},
                  'changed': {'profiles.py': 'cache = {}\n'
                                             '\n'
                                             'def profile(db, tenant, user):\n'
                                             '    if user not in cache:\n'
                                             '        cache[user] = db.load_profile(tenant, user)\n'
                                             '    return cache[user]\n'}},
 'webhook-retry': {'base': {'delivery.py': 'def dispatch(queue, send):\n'
                                           '    for message in queue:\n'
                                           '        send(message)\n'},
                   'changed': {'delivery.py': 'def dispatch(queue, send):\n'
                                              '    failed = []\n'
                                              '    for message in queue:\n'
                                              '        try:\n'
                                              '            send(message)\n'
                                              '        except TimeoutError:\n'
                                              '            failed.append(message)\n'
                                              '            break\n'
                                              '    return failed\n'}},
 'render-lookups': {'base': {'orders.js': 'export function labels(orders, productsById) {\n'
                                          '  return orders.map(order => '
                                          'productsById[order.productId].name);\n'
                                          '}\n'},
                    'changed': {'orders.js': 'export function labels(orders, products) {\n'
                                             '  return orders.map(order => products.find(p => p.id === '
                                             "order.productId)?.name ?? 'Unknown');\n"
                                             '}\n'}},
 'shared-normalization': {'base': {'importer.py': 'def import_email(email):\n'
                                                  '    return email.strip().lower()\n',
                                   'lookup.py': 'def lookup_email(email):\n    return email.lower()\n'},
                          'changed': {'importer.py': 'from normalize import canonical_email\n'
                                                     '\n'
                                                     'def import_email(email):\n'
                                                     '    return canonical_email(email)\n',
                                      'lookup.py': 'from normalize import canonical_email\n'
                                                   '\n'
                                                   'def lookup_email(email):\n'
                                                   '    return canonical_email(email)\n',
                                      'normalize.py': 'def canonical_email(email):\n'
                                                      '    return email.strip().lower()\n'}}})

SCENARIOS.update({
    "react-state": {
        "base": {"Cart.tsx": "export function add(setItems, item, items) {\n  setItems([...items, item]);\n}\n"},
        "changed": {"Cart.tsx": "export function add(setItems, item) {\n  setItems(previous => [...previous, item]);\n}\n"},
    },
    "java-validation": {
        "base": {"Order.java": "class Order {\n  int charge(int price, int quantity) { return price * quantity; }\n}\n"},
        "changed": {"Order.java": "class Order {\n  int charge(int price, int quantity) {\n    if (quantity <= 0) throw new IllegalArgumentException();\n    return Math.multiplyExact(price, quantity);\n  }\n}\n"},
    },
    "sql-inventory": {
        "base": {"reserve.sql": "UPDATE inventory SET stock = stock - 1 WHERE sku = :sku;\n"},
        "changed": {"reserve.sql": "UPDATE inventory SET stock = stock - 1 WHERE sku = :sku AND stock > 0 RETURNING stock;\n"},
    },
    "async-gather": {
        "base": {"load.py": "async def load(ids, fetch):\n    return [await fetch(i) for i in ids]\n"},
        "changed": {"load.py": "import asyncio\n\nasync def load(ids, fetch):\n    return await asyncio.gather(*(fetch(i) for i in ids), return_exceptions=True)\n"},
    },
    "pure-refactor": {
        "base": {"price.py": "def price(amount):\n    if amount < 0:\n        return 0\n    return amount * 2\n"},
        "changed": {"price.py": "def price(amount):\n    return max(0, amount) * 2\n"},
    },
})

def write_files(root, files):
    for name, content in files.items():
        path = root / name
        path.parent.mkdir(parents=True, exist_ok=True)
        with path.open("w", encoding="utf-8", newline="\n") as handle:
            handle.write(content)


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
        for name in scenario.get("removed", []):
            (repo / name).unlink()
        print(f"Created {repo}")


if __name__ == "__main__":
    main()
