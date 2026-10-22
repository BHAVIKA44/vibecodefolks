const element = (id) => document.getElementById(id);
const text = (tag, value, className) => {
  const node = document.createElement(tag);
  node.textContent = value;
  if (className) node.className = className;
  return node;
};

let pending = null;
let requestController = null;
let generation = 0;
let snapshotEvidence = null;
let changedFiles = [];
let fileGroups = [];
function selectTab(name) {
  for (const tab of ['overview','code','flow','quiz']) {
    element(`panel-${tab}`).hidden = tab !== name;
    element(`tab-${tab}`).className = tab === name ? 'tab active' : 'tab';
    element(`tab-${tab}`).setAttribute('aria-selected', String(tab === name));
  }
}
for (const tab of ['overview','code','flow','quiz']) element(`tab-${tab}`).addEventListener('click', () => selectTab(tab));
element('file-search').addEventListener('input', () => {
  const query = element('file-search').value.toLowerCase();
  for (const entry of changedFiles) entry.item.hidden = !entry.path.toLowerCase().includes(query);
  for (const group of fileGroups) group.heading.hidden = !changedFiles.some(entry => entry.folder === group.folder && !entry.item.hidden);
});
function showCode(path, ref) {
  selectTab('code');
  for (const entry of changedFiles) entry.item.className = entry.path === path ? 'selected' : '';
  const panel = element('code-view'); panel.replaceChildren();
  const file = snapshotEvidence?.files.find((entry) => entry.path === path);
  panel.append(text('div', path, 'code-path'));
  if (!file) { panel.append(text('p', 'Diff evidence is not available for this file. It may be excluded by the evidence limits or privacy filters. Choose Explain changes to review available evidence.', 'muted')); return; }
  panel.append(text('p', `${file.type} · captured diff evidence`, 'code-meta'));
  const lines = document.createElement('div'); lines.className = 'diff-lines';
  let oldLine = 0, newLine = 0, active = false, highlighted = null;
  for (const line of file.excerpt.split('\n')) {
    const hunk = /^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@/.exec(line);
    if (hunk) { oldLine = Number(hunk[1]); newLine = Number(hunk[2]); active = true; lines.append(text('div', line, 'diff-hunk')); continue; }
    if (!active || !line || ![' ', '+', '-'].includes(line[0])) { if (line) lines.append(text('div', line, 'diff-hunk')); continue; }
    const kind = line[0]; const oldNumber = kind === '+' ? null : oldLine++; const newNumber = kind === '-' ? null : newLine++;
    const hit = ref && (ref.side === 'old' ? oldNumber : newNumber) >= ref.start && (ref.side === 'old' ? oldNumber : newNumber) <= ref.end && (ref.side === 'old' ? oldNumber : newNumber) !== null;
    const row = document.createElement('div'); row.className = `diff-row ${kind === '+' ? 'addition' : kind === '-' ? 'deletion' : 'context'}${hit ? ' highlighted' : ''}`;
    row.append(text('span', oldNumber ?? '', 'line-number'), text('span', newNumber ?? '', 'line-number'), text('code', line, 'line-code')); lines.append(row); if (hit && !highlighted) highlighted = row;
  }
  panel.append(lines); highlighted?.scrollIntoView?.({block:'center'});
}
function citationButton(ref) {
  const button = text('button', `${ref.path} · ${ref.side} ${ref.start}–${ref.end}`, 'citation'); button.type = 'button';
  button.addEventListener('click', () => showCode(ref.path, ref)); return button;
}
async function refresh() {
  generation += 1;
  clearQuiz();
  if (requestController) requestController.abort();
  pending = null;
  snapshotEvidence = null; changedFiles = []; fileGroups = [];
  selectTab('overview'); element('overview-empty').hidden = false; element('quiz-empty').hidden = false;
  element('consent-panel').hidden = true;
  element('explanation').hidden = true;
  element('cancel-request').hidden = true;
  element('explain').disabled = true;
  element('explain-status').textContent = 'Previous explanation cleared. Refresh captures a new Git view.';
  const button = element('refresh');
  button.disabled = true;
  element('message').textContent = 'Analyzing changes…';
  element('files').replaceChildren();
  element('scope').hidden = true;
  for (const id of ['count', 'added', 'deleted']) element(id).textContent = '—';
  let loaded = false;
  try {
    const response = await fetch('/api/changes', { cache: 'no-store' });
    const data = await response.json();
    if (!response.ok) throw new Error(data.error || 'Analysis failed.');
    element('repository').textContent = `${data.repository.name || "Repository"} · ${data.repository.detached ? 'Detached HEAD ' : 'Branch '}${data.repository.branch}`;
    element('scope').textContent = data.scope;
    element('scope').hidden = false;
    element('count').textContent = data.total_files;
    element('added').textContent = `+${data.lines_added}`;
    element('deleted').textContent = `−${data.lines_deleted}`;
    const messages = [...data.warnings];
    if (!data.total_files) messages.unshift('No changes to display.');
    element('message').textContent = messages.join(' ');
    for (const file of data.files) {
      const folder = file.path.includes('/') ? file.path.slice(0, file.path.lastIndexOf('/')) : 'root';
      if (!fileGroups.some(group => group.folder === folder)) { const heading = text('li', folder, 'folder-heading'); heading.setAttribute('aria-hidden', 'true'); element('files').append(heading); fileGroups.push({folder,heading}); }
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
      item.append(paths, meta); item.setAttribute('role', 'button'); item.setAttribute('tabindex', '0');
      item.addEventListener('click', () => showCode(file.path));
      item.addEventListener('keydown', (event) => { if (event.key === 'Enter' || event.key === ' ') { event.preventDefault(); showCode(file.path); } });
      changedFiles.push({path:file.path,folder,item}); element('files').append(item);
    }
    loaded = data.total_files > 0;
  } catch (error) {
    element('repository').textContent = 'Repository unavailable';
    element('message').textContent = error.message || 'Unable to analyze this repository.';
  } finally { button.disabled = false; element('explain').disabled = !loaded; }
}
element('refresh').addEventListener('click', refresh);
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
  snapshotEvidence = result.evidence; prepareQuiz(result); element('quiz-empty').hidden = true; element('overview-empty').hidden = true; selectTab('overview');
  element('snapshot').textContent = `${e.scope} · captured ${new Date(result.captured_at).toLocaleString()} · ${result.snapshot.slice(0, 12)}`;
  renderGraph(e.graph, result.evidence);
  container.append(text('p', e.summary, 'summary-text'));
  function group(title, findings) {
    container.append(text('h3', title));
    if (!findings.length) { container.append(text('p', 'No supported findings provided.', 'muted')); return; }
    for (const finding of findings) {
      const block = document.createElement('article');
      block.append(text('h4', finding.title), text('p', finding.explanation, 'muted'));
      for (const ref of finding.references) block.append(citationButton(ref));
      container.append(block);
    }
  }
  group('Key changes', e.changes);
  group('What this may affect', e.impacts);
  group('How it works · inferred connections', e.relationships);
  container.append(text('h3', 'Things to verify'));
  const list = document.createElement('ul');
  for (const item of [...e.uncertainties, ...e.limitations]) list.append(text('li', item));
  container.append(list);
  section.hidden = false;
}
async function sendExplanation(consent) {
  element('consent-panel').hidden = true;
  element('explain-status').textContent = 'Waiting for the AI provider…';
  element('cancel-request').hidden = false;
  const currentGeneration = generation;
  const token = pending.token;
  const controller = new AbortController();
  requestController = controller;
  try {
    const result = await post('/api/explain', {token, consent}, controller.signal);
    if (currentGeneration !== generation) return;
    showExplanation(result);
    element('explain-status').textContent = 'Explanation ready.';
  } catch (error) {
    if (currentGeneration === generation) element('explain-status').textContent = error.name === 'AbortError' ? 'Request canceled.' : error.message;
  } finally { if (currentGeneration === generation) { pending = null; requestController = null; element('cancel-request').hidden = true; element('explain').disabled = false; } }
}
element('explain').addEventListener('click', async () => {
  element('explain').disabled = true;
  clearQuiz();
  element('explanation').hidden = true;
  element('consent').checked = false;
  element('explain-status').textContent = 'Collecting bounded diff evidence…';
  const currentGeneration = generation;
  try {
    const prepared = await post('/api/explain/prepare', {});
    if (currentGeneration !== generation) return;
    pending = prepared; snapshotEvidence = prepared.evidence;
    element('consent-title').textContent = pending.local ? 'Review evidence for your local provider' : 'Review before sending';
    element('consent-label').hidden = pending.local;
    element('send-evidence').textContent = pending.local ? 'Explain locally' : 'Send and explain';
    element('privacy-info').textContent = `Provider: ${pending.endpoint} · Model: ${pending.model}. Sending will transmit the selected file paths, diff excerpts, surrounding code and line references below. ${pending.local ? 'This provider is configured on a loopback address.' : 'The provider may retain this data. Review the evidence before consenting.'}`;
    addPreview(element('preview'), pending.evidence);
    element('skipped').textContent = pending.skipped.length ? `Skipped: ${pending.skipped.map((f) => `${f.path} (${f.reason})`).join(', ')}` : 'No files were skipped by the evidence filter.';
    element('consent-panel').hidden = false;
    element('explain-status').textContent = pending.local ? 'Review the evidence, then choose Explain locally.' : 'Consent required before sending code to a remote provider.';
  } catch (error) { if (currentGeneration === generation) { element('explain-status').textContent = error.message; element('explain').disabled = false; } }
});
element('send-evidence').addEventListener('click', () => {
  if (!pending.local && !element('consent').checked) { element('explain-status').textContent = 'Please check the consent box before sending.'; return; }
  sendExplanation(!pending.local && element('consent').checked);
});
element('cancel-consent').addEventListener('click', () => { pending = null; element('consent-panel').hidden = true; element('explain-status').textContent = 'Canceled; no evidence was sent.'; element('explain').disabled = false; });
element('cancel-request').addEventListener('click', () => requestController?.abort());

element('graph-fit').addEventListener('click', () => graphZoom(0));
element('graph-minus').addEventListener('click', () => graphZoom(-0.2));
element('graph-plus').addEventListener('click', () => graphZoom(0.2));

element('quiz-start').addEventListener('click', startQuiz);
refresh();
