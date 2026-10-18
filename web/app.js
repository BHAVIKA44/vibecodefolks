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
