// Development-only tests use Node's built-in test runner; the application runtime remains Go only.
const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const path = require('node:path');

class Element {
  constructor(tag = 'div') { this.tag = tag; this.children = []; this.attributes = {}; this.listeners = {}; this.hidden = false; this.textContent = ''; this.disabled = false; this.checked = false; }
  set innerHTML(_) { throw new Error('unsafe HTML insertion'); }
  append(...children) { this.children.push(...children); }
  replaceChildren(...children) { this.children = children; }
  setAttribute(key, value) { this.attributes[key] = value; }
  addEventListener(type, handler) { this.listeners[type] = handler; }
}
function environment(fetch) {
  const elements = new Map();
  const document = { getElementById(id) { if (!elements.has(id)) elements.set(id, new Element()); return elements.get(id); }, createElement: tag => new Element(tag), createElementNS: (_ns, tag) => new Element(tag) };
  const context = vm.createContext({document, fetch, AbortController, Date, Map, Object, Array, String});
  const load = file => vm.runInContext(fs.readFileSync(path.join(__dirname, '..', 'web', file), 'utf8'), context);
  return {elements, context, load};
}
function allText(node) { return [node.textContent, ...node.children.flatMap(allText)].join(' '); }
const jsonResponse = obj => ({ok:true, json:async()=>obj});
const changes = {repository:{name:'example',branch:'main',detached:false},scope:'Working Tree',total_files:0,lines_added:0,lines_deleted:0,warnings:[],files:[]};

test('graph renders untrusted labels and evidence as text and selects details', () => {
  const {elements,context,load} = environment(() => {});
  vm.runInContext(`const element=id=>document.getElementById(id); const text=(tag,value,className)=>{const n=document.createElement(tag); n.textContent=value; if(className)n.className=className; return n}`, context);
  load('graph.js');
  const label = '<img src=x onerror=alert(1)>'; const excerpt='<script>throw 1</script>';
  context.graph={source:'ai',notice:'inferred',nodes:[{id:'a',label,description:'desc',change:'change',impact:'impact',uncertainty:'unknown',status:'modified',references:[{path:'a.js',side:'new',start:1,end:1}]}],edges:[]};
  context.evidence={files:[{path:'a.js',excerpt}],skipped:[]};
  vm.runInContext('renderGraph(graph,evidence)',context);
  assert.match(allText(elements.get('change-graph')), /<img/);
  assert.match(allText(elements.get('graph-detail')), /<script>/);
  const node=elements.get('change-graph').children.find(e=>e.attributes['aria-label']?.includes(label));
  assert.equal(node.attributes.tabindex,'0'); assert.equal(node.attributes.role,'button');
  node.listeners.keydown({key:'Enter',preventDefault(){}});
  assert.match(allText(elements.get('graph-detail')), /unknown/);
});

test('refresh invalidates an in-flight preparation and hides old explanations', async () => {
  let resolvePrepare; const prepare = new Promise(resolve => {resolvePrepare=resolve;});
  const fetch = url => url === '/api/changes' ? Promise.resolve(jsonResponse(changes)) : prepare;
  const {elements,load} = environment(fetch); load('graph.js'); load('app.js');
  await new Promise(resolve => setImmediate(resolve));
  const pending = elements.get('explain').listeners.click();
  const refreshing = elements.get('refresh').listeners.click();
  resolvePrepare(jsonResponse({token:'old',local:true,endpoint:'http://127.0.0.1',model:'mock',evidence:{files:[]},skipped:[]}));
  await Promise.all([pending,refreshing]);
  assert.equal(elements.get('consent-panel').hidden,true);
  assert.equal(elements.get('explanation').hidden,true);
  assert.match(elements.get('explain-status').textContent,/Previous explanation cleared/);
});

test('refresh discards an in-flight explanation and rendering makes no extra AI request', async () => {
  let resolveExplanation; let aiRequests = 0;
  const explanation = new Promise(resolve => { resolveExplanation = resolve; });
  const fetch = url => {
    if (url === '/api/changes') return Promise.resolve(jsonResponse(changes));
    if (url === '/api/explain/prepare') return Promise.resolve(jsonResponse({token:'snapshot',local:true,endpoint:'http://127.0.0.1',model:'mock',evidence:{files:[]},skipped:[]}));
    if (url === '/api/explain') { aiRequests++; return explanation; }
    throw new Error(`unexpected request ${url}`);
  };
  const {elements,load} = environment(fetch); load('graph.js'); load('app.js');
  await new Promise(resolve => setImmediate(resolve));
  await elements.get('explain').listeners.click();
  elements.get('send-evidence').listeners.click();
  await elements.get('refresh').listeners.click();
  resolveExplanation(jsonResponse({summary:'stale result',graph:{nodes:[],edges:[]}}));
  await new Promise(resolve => setImmediate(resolve));
  assert.equal(aiRequests,1);
  assert.equal(elements.get('explanation').hidden,true);
  assert.match(elements.get('explain-status').textContent,/Previous explanation cleared/);
});
