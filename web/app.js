const element = (id) => document.getElementById(id);
const text = (tag, value, className) => {
  const node = document.createElement(tag);
  node.textContent = value;
  if (className) node.className = className;
  return node;
};

async function refresh() {
  const button = element('refresh');
  button.disabled = true;
  element('message').textContent = 'Analyzing changes…';
  element('files').replaceChildren();
  element('scope').hidden = true;
  for (const id of ['count', 'added', 'deleted']) element(id).textContent = '—';
  try {
    const response = await fetch('/api/changes', { cache: 'no-store' });
    const data = await response.json();
    if (!response.ok) throw new Error(data.error || 'Analysis failed.');
    element('repository').textContent = `${data.repository.name} · ${data.repository.detached ? 'Detached HEAD ' : 'Branch '}${data.repository.branch}`;
    element('scope').textContent = data.scope;
    element('scope').hidden = false;
    element('count').textContent = data.total_files;
    element('added').textContent = `+${data.lines_added}`;
    element('deleted').textContent = `−${data.lines_deleted}`;
    const messages = [...data.warnings];
    if (!data.total_files) messages.unshift('No changes to display.');
    element('message').textContent = messages.join(' ');
    for (const file of data.files) {
      const item = document.createElement('li');
      const paths = document.createElement('div');
      paths.append(text('span', file.path, 'path'));
      if (file.previous_path) paths.append(text('span', ` from ${file.previous_path}`, 'previous'));
      const meta = document.createElement('div');
      meta.className = 'file-meta';
      meta.append(text('span', file.type, 'status'));
      const details = file.binary ? 'Binary' : file.lines_added === null || file.lines_deleted === null ? 'Stats unavailable' : `+${file.lines_added} / −${file.lines_deleted}`;
      meta.append(text('span', details, 'file-stats'));
      if (file.details) meta.append(text('span', file.details, 'file-stats'));
      item.append(paths, meta);
      element('files').append(item);
    }
  } catch (error) {
    element('repository').textContent = 'Repository unavailable';
    element('message').textContent = error.message || 'Unable to analyze this repository.';
  } finally { button.disabled = false; }
}
element('refresh').addEventListener('click', refresh);
refresh();

let pending = null;
let requestController = null;
async function post(url, data, signal) {
  const response = await fetch(url, {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify(data), signal});
  const result = await response.json();
  if (!response.ok) throw new Error(result.error || 'Request failed.');
  return result;
}
function addPreview(container, evidence) {
  container.replaceChildren();
  for (const file of evidence.files) {
    const detail = document.createElement('details');
    detail.append(text('summary', `${file.path} (${file.type})`));
    detail.append(text('pre', file.excerpt, 'excerpt'));
    container.append(detail);
  }
}
function showExplanation(result) {
  const section = element('explanation');
  const container = element('explanation-content');
  container.replaceChildren();
  const e = result.explanation;
  container.append(text('h3', 'Summary'), text('p', e.summary, 'muted'));
  function group(title, findings) {
    container.append(text('h3', title));
    if (!findings.length) { container.append(text('p', 'No supported findings provided.', 'muted')); return; }
    for (const finding of findings) {
      const block = document.createElement('article');
      block.append(text('h4', finding.title), text('p', finding.explanation, 'muted'));
      for (const ref of finding.references) {
        const detail = document.createElement('details');
        detail.className = 'reference';
        detail.append(text('summary', `${ref.path} · ${ref.side} lines ${ref.start}–${ref.end}`));
        const file = result.evidence.files.find((f) => f.path === ref.path);
        if (file) detail.append(text('pre', file.excerpt, 'excerpt'));
        block.append(detail);
      }
      container.append(block);
    }
  }
  group('What AI changed · observed in code', e.changes);
  group('Why it matters · likely impact', e.impacts);
  group('How the changed pieces connect', e.relationships);
  container.append(text('h3', 'What we couldn’t verify'));
  const list = document.createElement('ul');
  for (const item of [...e.uncertainties, ...e.limitations]) list.append(text('li', item));
  container.append(list);
  section.hidden = false;
}
async function sendExplanation(consent) {
  element('consent-panel').hidden = true;
  element('explain-status').textContent = 'Waiting for the AI provider…';
  element('cancel-request').hidden = false;
  requestController = new AbortController();
  try {
    const result = await post('/api/explain', {token: pending.token, consent}, requestController.signal);
    showExplanation(result);
    element('explain-status').textContent = 'Explanation ready.';
  } catch (error) {
    element('explain-status').textContent = error.name === 'AbortError' ? 'Request canceled.' : error.message;
  } finally { pending = null; requestController = null; element('cancel-request').hidden = true; element('explain').disabled = false; }
}
element('explain').addEventListener('click', async () => {
  element('explain').disabled = true;
  element('explanation').hidden = true;
  element('consent').checked = false;
  element('explain-status').textContent = 'Collecting bounded diff evidence…';
  try {
    pending = await post('/api/explain/prepare', {});
    element('consent-title').textContent = pending.local ? 'Review evidence for your local provider' : 'Review before sending';
    element('consent-label').hidden = pending.local;
    element('send-evidence').textContent = pending.local ? 'Explain locally' : 'Send and explain';
    element('privacy-info').textContent = `Provider: ${pending.endpoint} · Model: ${pending.model}. Sending will transmit the selected file paths, diff excerpts, surrounding code and line references below. ${pending.local ? 'This provider is configured on a loopback address.' : 'The provider may retain this data. Review the evidence before consenting.'}`;
    addPreview(element('preview'), pending.evidence);
    element('skipped').textContent = pending.skipped.length ? `Skipped: ${pending.skipped.map((f) => `${f.path} (${f.reason})`).join(', ')}` : 'No files were skipped by the evidence filter.';
    element('consent-panel').hidden = false;
    element('explain-status').textContent = pending.local ? 'Review the evidence, then choose Explain locally.' : 'Consent required before sending code to a remote provider.';
  } catch (error) { element('explain-status').textContent = error.message; element('explain').disabled = false; }
});
element('send-evidence').addEventListener('click', () => {
  if (!pending.local && !element('consent').checked) { element('explain-status').textContent = 'Please check the consent box before sending.'; return; }
  sendExplanation(!pending.local && element('consent').checked);
});
element('cancel-consent').addEventListener('click', () => { pending = null; element('consent-panel').hidden = true; element('explain-status').textContent = 'Canceled; no evidence was sent.'; element('explain').disabled = false; });
element('cancel-request').addEventListener('click', () => requestController?.abort());
