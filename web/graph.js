// The graph is entirely server supplied. This renderer never creates relationships.
const svgNS = 'http://www.w3.org/2000/svg';
function svgElement(tag, attrs = {}) {
  const node = document.createElementNS(svgNS, tag);
  for (const [key, value] of Object.entries(attrs)) node.setAttribute(key, String(value));
  return node;
}
function svgText(parent, label, x, y, className) {
  const node = svgElement('text', {x, y, class: className});
  node.textContent = label;
  parent.append(node);
  return node;
}
function wrapLabel(label, width = 23, max = 2) {
  const chars = Array.from(label);
  const rows = [];
  for (let i = 0; i < chars.length && rows.length < max; i += width) rows.push(chars.slice(i, i + width).join(''));
  if (chars.length > width * max) rows[max - 1] = rows[max - 1].slice(0, -1) + '…';
  return rows;
}
function graphReferences(panel, refs, evidence) {
  for (const ref of refs) {
    const detail = document.createElement('details');
    detail.append(text('summary', `${ref.path} · ${ref.side} ${ref.start}–${ref.end}`));
    const file = evidence.files.find((entry) => entry.path === ref.path);
    if (file) detail.append(text('pre', file.excerpt, 'excerpt'));
    panel.append(detail);
  }
}
function graphDetail(item, kind, evidence, graph) {
  const panel = element('graph-detail');
  panel.replaceChildren();
  if (kind === 'node') {
    panel.append(text('p', graph.source === 'ai' ? 'AI INTERPRETATION' : 'GIT EVIDENCE', 'detail-label'), text('h3', item.label), text('span', item.status, `node-badge ${item.status}`));
    for (const [title, value] of [['What it does', item.description], ['What changed', item.change], ['Why it matters', item.impact], ['Not verified', item.uncertainty]]) {
      panel.append(text('h4', title), text('p', value, 'muted'));
    }
  } else {
    panel.append(text('p', 'AI-INFERRED CONNECTION', 'detail-label'), text('h3', item.label), text('p', item.description, 'muted'), text('p', 'This connection is an inference. Runtime behavior has not been verified.', 'graph-warning'));
  }
  panel.append(text('h4', 'Supporting evidence'));
  graphReferences(panel, item.references, evidence);
}
function renderGraph(graph, evidence) {
  const svg = element('change-graph');
  const panel = element('graph-detail');
  svg.replaceChildren(); panel.replaceChildren();
  element('graph-notice').textContent = graph.notice + (evidence.skipped.length ? ` ${evidence.skipped.length} file(s) were excluded from this evidence snapshot.` : '');
  if (!graph.nodes.length) { panel.append(text('p', 'No eligible file evidence is available.', 'muted')); return; }
  const columns = Math.min(3, graph.nodes.length), width = columns * 252 + 22;
  const height = Math.ceil(graph.nodes.length / columns) * 152 + 30;
  svg.setAttribute('viewBox', `0 0 ${width} ${height}`);
  svg.setAttribute('width', width); svg.setAttribute('height', height);
  const defs = svgElement('defs'); const marker = svgElement('marker', {id:'arrowhead', viewBox:'0 0 10 10', refX:9, refY:5, markerWidth:7, markerHeight:7, orient:'auto-start-reverse'});
  marker.append(svgElement('path', {d:'M 0 0 L 10 5 L 0 10 z', fill:'#b5a4dc'})); defs.append(marker); svg.append(defs);
  const positions = new Map();
  graph.nodes.forEach((node, i) => positions.set(node.id, {x: 20 + (i % columns) * 252, y: 20 + Math.floor(i / columns) * 152}));
  const bind = (group, action) => {
    group.setAttribute('role','button'); group.setAttribute('tabindex','0');
    group.addEventListener('click', action);
    group.addEventListener('keydown', (event) => { if (event.key === 'Enter' || event.key === ' ') { event.preventDefault(); action(); } });
  };
  for (const edge of graph.edges) {
    const from = positions.get(edge.from), to = positions.get(edge.to);
    if (!from || !to) continue;
    let x1, y1, x2, y2;
    if (from.y === to.y) { const right = to.x > from.x; x1 = from.x + (right ? 210 : 0); y1 = from.y + 45; x2 = to.x + (right ? 0 : 210); y2 = to.y + 45; }
    else { const down = to.y > from.y; x1=from.x+105; y1=from.y+(down ? 90 : 0); x2=to.x+105; y2=to.y+(down ? 0 : 90); }
    const group = svgElement('g', {class:'graph-edge', 'aria-label':`Inferred connection: ${edge.label}`});
    const line = svgElement('line', {x1,y1,x2,y2, 'marker-end':'url(#arrowhead)'});
    group.append(line);
    svgText(group, wrapLabel(edge.label, 16, 1)[0], (x1+x2)/2, (y1+y2)/2-8, 'svg-edge-label').setAttribute('text-anchor','middle');
    bind(group, () => graphDetail(edge, 'edge', evidence, graph)); svg.append(group);
  }
  for (const node of graph.nodes) {
    const {x,y} = positions.get(node.id);
    const group = svgElement('g', {class:`graph-node ${node.status}`, 'aria-label':`${node.label}, ${node.status}. Select for details.`});
    group.append(svgElement('rect',{x,y,width:210,height:90,rx:12}));
    svgText(group, node.status.toUpperCase(), x+14, y+23, 'svg-status');
    wrapLabel(node.label).forEach((line, i) => svgText(group,line,x+14,y+46+i*18,'svg-label'));
    bind(group, () => graphDetail(node,'node',evidence,graph)); svg.append(group);
  }
  graphDetail(graph.nodes[0], 'node', evidence, graph);
}
