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
  const {elements,load} = environment(fetch); load('graph.js'); load('quiz.js'); load('app.js');
  await new Promise(resolve => setImmediate(resolve));
  const pending = elements.get('explain').listeners.click();
  const refreshing = elements.get('refresh').listeners.click();
  resolvePrepare(jsonResponse({token:'old',local:true,endpoint:'http://127.0.0.1',model:'mock',evidence:{files:[]},skipped:[]}));
  await Promise.all([pending,refreshing]);
  assert.equal(elements.get('consent-panel').hidden,true);
  assert.equal(elements.get('explanation').hidden,true);
  assert.match(elements.get('explain-status').textContent,/Previous explanation cleared|Select a file/);
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
  const {elements,load} = environment(fetch); load('graph.js'); load('quiz.js'); load('app.js');
  await new Promise(resolve => setImmediate(resolve));
  await elements.get('explain').listeners.click();
  elements.get('send-evidence').listeners.click();
  await elements.get('refresh').listeners.click();
  resolveExplanation(jsonResponse({summary:'stale result',graph:{nodes:[],edges:[]}}));
  await new Promise(resolve => setImmediate(resolve));
  assert.equal(aiRequests,1);
  assert.equal(elements.get('explanation').hidden,true);
  assert.match(elements.get('explain-status').textContent,/Previous explanation cleared|Select a file/);
});

const sampleQuiz = {questions:[{id:'one',text:'What changed <script>?',options:['First','Second','Third','Fourth'],correct_index:1,explanation:'The cited line shows the second option.',option_explanations:['The first option misses the changed branch.','The second option follows the branch.','The third option skips the check.','The fourth option adds a nonexistent step.'],category:'change',references:[{path:'file.js',side:'new',start:1,end:1}]}]};
test('quiz scores answers, explains mistakes, retries and makes no AI requests', async () => {
  let aiCalls = 0, freshCalls = 0;
  const fetch = url => { if(url === '/api/harnesses') return Promise.resolve(jsonResponse([])); if(url === '/api/quiz/fresh') {freshCalls++; return Promise.resolve(jsonResponse({fresh:true}));} if(url === '/api/changes') return Promise.resolve(jsonResponse(changes)); aiCalls++; throw Error('unexpected AI request'); };
  const {elements,context,load} = environment(fetch); load('graph.js'); load('quiz.js'); load('app.js');
  await new Promise(resolve => setImmediate(resolve));
  context.result={snapshot:'a'.repeat(64),explanation:{quiz:sampleQuiz},evidence:{files:[{path:'file.js',excerpt:'<img onerror=alert(1)>'}]}};
  vm.runInContext('prepareQuiz(result)',context);
  await elements.get('quiz-start').listeners.click();
  const panel=elements.get('quiz-content');
  assert.match(allText(panel),/<script>/);
  assert.match(allText(panel),/<img/);
  assert.doesNotMatch(allText(panel),/Correct answer:/);
  let options=panel.children.find(e=>e.className==='quiz-options');
  options.children[0].listeners.click();
  assert.ok(options.children.every(e=>e.disabled));
  assert.match(allText(panel),/Correct answer: B\. Second/);
  assert.match(allText(panel),/cited line/);
  assert.match(allText(panel),/Your choice: The first option misses the changed branch/);
  assert.match(allText(panel),/<img/);
  let feedback=panel.children.find(e=>e.className==='quiz-feedback');
  feedback.children.find(e=>e.textContent==='See score').listeners.click();
  assert.match(allText(panel),/0 \/ 1 correct/);
  assert.match(allText(panel),/Revisit.*change/);
  assert.match(allText(panel),/Your answer: First/);
  await panel.children.find(e=>e.textContent==='Retry same questions').listeners.click();
  options=panel.children.find(e=>e.className==='quiz-options'); options.children[1].listeners.click();
  feedback=panel.children.find(e=>e.className==='quiz-feedback'); feedback.children.find(e=>e.textContent==='See score').listeners.click();
  assert.match(allText(panel),/1 \/ 1 correct/);
  assert.equal(freshCalls,2); assert.equal(aiCalls,0);
});

test('refresh invalidates pending quiz snapshot check and stale results', async () => {
  let resolveFresh; const promise=new Promise(resolve=>{resolveFresh=resolve;});
  const fetch=url=>url==='/api/quiz/fresh'?promise:Promise.resolve(jsonResponse(changes));
  const {elements,context,load}=environment(fetch);load('graph.js');load('quiz.js');load('app.js');
  await new Promise(resolve=>setImmediate(resolve));
  context.result={snapshot:'a'.repeat(64),explanation:{quiz:sampleQuiz},evidence:{files:[]}};
  vm.runInContext('prepareQuiz(result)',context);
  const pending=elements.get('quiz-start').listeners.click();
  await elements.get('refresh').listeners.click();
  resolveFresh(jsonResponse({fresh:true})); await pending;
  assert.equal(elements.get('quiz-section').hidden,true);
  assert.equal(elements.get('quiz-start').disabled,false);
  assert.equal(elements.get('quiz-content').children.length,0);
});

test('changed snapshot blocks quiz', async () => {
  const fetch=url=>Promise.resolve(jsonResponse(url==='/api/quiz/fresh'?{fresh:false}:changes));
  const {elements,context,load}=environment(fetch);load('graph.js');load('quiz.js');load('app.js');
  await new Promise(resolve=>setImmediate(resolve));
  context.result={snapshot:'a'.repeat(64),explanation:{quiz:sampleQuiz},evidence:{files:[]}};
  vm.runInContext('prepareQuiz(result)',context); await elements.get('quiz-start').listeners.click();
  assert.match(elements.get('quiz-status').textContent,/evidence has changed/);
  assert.equal(elements.get('quiz-content').children.length,0);
});

test('explain is disabled until the refreshed Git view finishes', async () => {
  let resolveChanges;
  const changesRequest = new Promise(resolve => {resolveChanges=resolve;});
  const {elements,load}=environment(url=>url==='/api/changes'?changesRequest:Promise.reject(Error('unexpected request')));
  load('graph.js');load('quiz.js');load('app.js');
  assert.equal(elements.get('explain').disabled,true);
  resolveChanges(jsonResponse({...changes, total_files: 1}));
  await new Promise(resolve=>setImmediate(resolve));
  assert.equal(elements.get('explain').disabled,false);
});

test('unavailable repository keeps explanation disabled until refresh succeeds', async () => {
  let available = false;
  const fetch = async () => available ? jsonResponse({...changes, total_files: 1}) : {ok:false, json:async()=>({error:'Repository unavailable'})};
  const {elements,load}=environment(fetch); load('graph.js'); load('quiz.js'); load('app.js');
  await new Promise(resolve=>setImmediate(resolve));
  assert.equal(elements.get('explain').disabled,true);
  assert.match(elements.get('message').textContent,/Repository unavailable/);
  available = true;
  await elements.get('refresh').listeners.click();
  assert.equal(elements.get('explain').disabled,false);
});

test('a clean repository has no changes to explain', async () => {
  const fetch = async () => jsonResponse({...changes, total_files: 0, files: []});
  const {elements,load}=environment(fetch); load('graph.js'); load('quiz.js'); load('app.js');
  await new Promise(resolve=>setImmediate(resolve));
  assert.equal(elements.get('explain').disabled,true);
  assert.match(elements.get('message').textContent,/No changes to display/);
});

test('citations open the referenced diff and highlight the exact added line', async () => {
  const {elements,context,load}=environment(() => Promise.resolve(jsonResponse(changes)));
  load('graph.js'); load('quiz.js'); load('app.js');
  await new Promise(resolve=>setImmediate(resolve));
  context.fixtureEvidence={files:[{path:'src/App.js',type:'modified',excerpt:'@@ -1,2 +1,3 @@\n function App() {\n+  return fetch("/api/items");\n }\n'}]};
  vm.runInContext("snapshotEvidence=fixtureEvidence; showCode('src/App.js',{path:'src/App.js',side:'new',start:2,end:2})",context);
  assert.equal(elements.get('panel-code').hidden,false);
  const diff=elements.get('code-view').children.find(e=>e.className==='diff-lines');
  const highlighted=diff.children.find(e=>e.className.includes('highlighted'));
  assert.match(allText(highlighted), /fetch\("\/api\/items"\)/);
  assert.equal(diff.children.filter(e=>e.className.includes('highlighted')).length,1);
});

test('unavailable graph is explicit and does not render generic file cards', () => {
  const {elements,context,load}=environment(()=>{});
  vm.runInContext(`const element=id=>document.getElementById(id); const text=(tag,value,className)=>{const n=document.createElement(tag); n.textContent=value; if(className)n.className=className; return n}`,context);
  load('graph.js');
  context.graph={source:'unavailable',diagnostic:'invalid',notice:'No verified change flow available.',nodes:[],edges:[]};
  context.evidence={files:[],skipped:[]};
  vm.runInContext('renderGraph(graph,evidence)',context);
  assert.match(elements.get('graph-notice').textContent,/No verified change flow/);
  assert.match(allText(elements.get('graph-detail')),/No verified change flow/);
  assert.equal(elements.get('change-graph').children.length,0);
});

test('a new explanation attempt clears a stale change flow before contacting the provider', async () => {
  const fetch = url => url === '/api/changes' ? Promise.resolve(jsonResponse({...changes, total_files: 1})) : Promise.resolve({ok:false, json:async()=>({error:'Provider unavailable'})});
  const {elements,load}=environment(fetch); load('graph.js'); load('quiz.js'); load('app.js');
  await new Promise(resolve=>setImmediate(resolve));
  elements.get('change-graph').append(new Element('g'));
  elements.get('graph-notice').textContent = 'Old graph';
  await elements.get('explain').listeners.click();
  assert.equal(elements.get('change-graph').children.length,0);
  assert.match(elements.get('graph-notice').textContent,/Preparing a new explanation/);
});

test('opens a changed file before configuring or invoking an AI provider', async () => {
  const path = 'server.py';
  const calls = [];
  const fetch = url => {
    calls.push(url);
    if (url === '/api/changes') return Promise.resolve(jsonResponse({...changes,total_files:1,files:[{path,type:'modified'}]}));
    if (url === '/api/evidence') return Promise.resolve(jsonResponse({files:[{path,type:'modified',excerpt:'@@ -1 +1 @@\n-return 0\n+return 1'}],skipped:[]}));
    throw Error('unexpected provider call');
  };
  const {elements,load} = environment(fetch);
  load('graph.js'); load('quiz.js'); load('app.js');
  await new Promise(resolve => setImmediate(resolve));
  elements.get('files').children[1].listeners.click();
  await new Promise(resolve => setImmediate(resolve));
  assert.match(allText(elements.get('code-view')), /\+return 1/);
  assert.deepEqual(calls, ['/api/harnesses','/api/changes','/api/evidence']);
});

test('quiz unavailability explains omission or invalid citations without exposing model content', () => {
  const {elements,context,load} = environment(() => Promise.resolve(jsonResponse(changes)));
  load('graph.js'); load('quiz.js'); load('app.js');
  for (const [reason, expected] of [
    ['model_null', /provider returned no quiz/i],
    ['empty', /no questions/i],
    ['invalid_structure', /invalid format/i],
    ['invalid_reference', /cited code outside/i],
  ]) {
    context.result = {explanation:{quiz:null,quiz_diagnostic:reason}};
    vm.runInContext('prepareQuiz(result)',context);
    assert.match(elements.get('quiz-status').textContent, expected);
    assert.equal(elements.get('quiz-start').hidden, true);
  }
});

test('remote quiz generation requires fresh consent and retry uses the existing questions', async () => {
  const calls = [];
  const fetch = async (url, options) => {
    calls.push({url, body: options?.body && JSON.parse(options.body)});
    if (url === '/api/harnesses') return jsonResponse([]);
    if (url === '/api/changes') return jsonResponse(changes);
    if (url === '/api/quiz/fresh') return jsonResponse({fresh:true});
    if (url === '/api/quiz/generate') return jsonResponse({snapshot:'a'.repeat(64),quiz:sampleQuiz});
    throw Error(`unexpected ${url}`);
  };
  const {elements,context,load} = environment(fetch);
  load('graph.js'); load('quiz.js'); load('app.js');
  await new Promise(resolve => setImmediate(resolve));
  context.result={snapshot:'a'.repeat(64),quiz_session:'session',quiz_local:false,quiz_provider:'https://example.com',quiz_model:'model',explanation:{},evidence:{files:[]}};
  vm.runInContext('prepareQuiz(result)',context);
  elements.get('quiz-generate').listeners.click();
  assert.equal(elements.get('quiz-consent-panel').hidden,false);
  assert.match(elements.get('quiz-privacy-info').textContent,/another request.*incur costs/);
  await elements.get('quiz-confirm').listeners.click();
  assert.equal(calls.filter(call=>call.url==='/api/quiz/generate').length,0);
  elements.get('quiz-consent').checked=true;
  await elements.get('quiz-confirm').listeners.click();
  assert.deepEqual(calls.find(call=>call.url==='/api/quiz/generate').body,{session:'session',snapshot:'a'.repeat(64),consent:true});
  const panel=elements.get('quiz-content');
  panel.children.find(e=>e.className==='quiz-options').children[0].listeners.click();
  panel.children.find(e=>e.className==='quiz-feedback').children.find(e=>e.textContent==='See score').listeners.click();
  await panel.children.find(e=>e.textContent==='Retry same questions').listeners.click();
  assert.equal(calls.filter(call=>call.url==='/api/quiz/generate').length,1);
});

test('refresh discards pending generated quiz and aborts its request', async () => {
  let resolveQuiz, signal;
  const deferred = new Promise(resolve => {resolveQuiz=resolve;});
  const fetch = (url, options) => {
    if (url === '/api/quiz/generate') { signal=options.signal; return deferred; }
    if (url === '/api/harnesses') return Promise.resolve(jsonResponse([]));
    return Promise.resolve(jsonResponse(changes));
  };
  const {elements,context,load}=environment(fetch);
  load('graph.js'); load('quiz.js'); load('app.js');
  await new Promise(resolve=>setImmediate(resolve));
  context.result={snapshot:'a'.repeat(64),quiz_session:'session',quiz_local:true,explanation:{},evidence:{files:[]}};
  vm.runInContext('prepareQuiz(result)',context);
  elements.get('quiz-generate').listeners.click();
  await elements.get('refresh').listeners.click();
  assert.equal(signal.aborted,true);
  resolveQuiz(jsonResponse({snapshot:'a'.repeat(64),quiz:sampleQuiz}));
  await new Promise(resolve=>setImmediate(resolve));
  assert.equal(elements.get('quiz-section').hidden,true);
  assert.equal(elements.get('quiz-content').children.length,0);
});

test('cancel during post-generation freshness check does not display the quiz', async () => {
  let resolveFresh, freshSignal;
  const pendingFresh = new Promise(resolve => { resolveFresh = resolve; });
  const fetch = (url, options) => {
    if (url === '/api/harnesses') return Promise.resolve(jsonResponse([]));
    if (url === '/api/changes') return Promise.resolve(jsonResponse(changes));
    if (url === '/api/quiz/generate') return Promise.resolve(jsonResponse({snapshot:'a'.repeat(64),quiz:sampleQuiz}));
    if (url === '/api/quiz/fresh') { freshSignal = options.signal; return pendingFresh; }
    throw Error(`unexpected ${url}`);
  };
  const {elements,context,load} = environment(fetch);
  load('graph.js'); load('quiz.js'); load('app.js');
  await new Promise(resolve => setImmediate(resolve));
  context.result = {snapshot:'a'.repeat(64),quiz_session:'session',quiz_local:true,explanation:{},evidence:{files:[]}};
  vm.runInContext('prepareQuiz(result)', context);
  const pending = vm.runInContext('generateQuiz(false)', context);
  await new Promise(resolve => setImmediate(resolve));
  elements.get('quiz-cancel-request').listeners.click();
  assert.equal(freshSignal.aborted, true);
  resolveFresh(jsonResponse({fresh:true}));
  await pending;
  assert.equal(elements.get('quiz-content').children.length, 0);
  assert.match(elements.get('quiz-status').textContent, /canceled/i);
  assert.equal(elements.get('quiz-generate').hidden, false);
});

test('send-evidence button when pending is null does not throw', async () => {
  const fetch = () => Promise.resolve(jsonResponse({...changes, total_files: 1}));
  const {elements,load}=environment(fetch); load('graph.js'); load('quiz.js'); load('app.js');
  await new Promise(resolve=>setImmediate(resolve));
  assert.doesNotThrow(() => {
    elements.get('send-evidence').listeners.click();
  });
  assert.match(elements.get('explain-status').textContent, /No prepared evidence/);
});

test('import context lines are rendered as numbered diff rows and highlighted when cited', async () => {
  const {elements,context,load}=environment(() => Promise.resolve(jsonResponse(changes)));
  load('graph.js'); load('quiz.js'); load('app.js');
  await new Promise(resolve=>setImmediate(resolve));
  const excerpt = '@@ -10,2 +10,2 @@\n-oldCall();\n+newCall();\n\n[Additional import context from the current file]\n1: import { useEffect } from "react";\n';
  context.fixtureEvidence={files:[{path:'src/hook.ts',type:'modified',excerpt}]};
  vm.runInContext("snapshotEvidence=fixtureEvidence; showCode('src/hook.ts',{path:'src/hook.ts',side:'new',start:1,end:1})",context);
  const diff=elements.get('code-view').children.find(e=>e.className==='diff-lines');
  const highlighted=diff.children.filter(e=>e.className.includes('highlighted'));
  assert.equal(highlighted.length,1);
  assert.match(allText(highlighted[0]), /import \{ useEffect \}/);
});

test('provider failure during sendExplanation preserves pending token and consent panel for immediate retry', async () => {
  let explainCalls = 0;
  const fetch = url => {
    if (url === '/api/harnesses') return Promise.resolve(jsonResponse([]));
    if (url === '/api/changes') return Promise.resolve(jsonResponse({...changes, total_files: 1}));
    if (url === '/api/explain/prepare') return Promise.resolve(jsonResponse({token:'keep-me',local:true,endpoint:'Codex local harness',model:'',evidence:{files:[]},skipped:[]}));
    if (url === '/api/explain') {
      explainCalls++;
      if (explainCalls === 1) return Promise.resolve({ok:false, json:async()=>({error:'Codex reported a usage or rate limit.'})});
      return Promise.resolve(jsonResponse({snapshot:'a'.repeat(64),captured_at:new Date().toISOString(),explanation:{summary:'Done',scope:'Working Tree',changes:[],impacts:[],relationships:[],uncertainties:[],limitations:[]},evidence:{files:[],skipped:[]}}));
    }
    throw new Error(`unexpected ${url}`);
  };
  const {elements,load}=environment(fetch); load('graph.js'); load('quiz.js'); load('app.js');
  await new Promise(resolve=>setImmediate(resolve));
  await elements.get('explain').listeners.click();
  await elements.get('send-evidence').listeners.click();
  await new Promise(resolve=>setImmediate(resolve));
  assert.equal(elements.get('consent-panel').hidden, false);
  assert.match(elements.get('explain-status').textContent, /usage or rate limit/);
  await elements.get('send-evidence').listeners.click();
  await new Promise(resolve=>setImmediate(resolve));
  assert.equal(explainCalls, 2);
  assert.equal(elements.get('explanation').hidden, false);
});

test('clicking Code changes tab selects first file and supports prev/next navigation', async () => {
  const multiChanges = {...changes, total_files: 2, files: [{path:'a.go',type:'modified',lines_added:1,lines_deleted:0},{path:'b.go',type:'modified',lines_added:2,lines_deleted:1}]};
  const fetch = url => {
    if (url === '/api/harnesses') return Promise.resolve(jsonResponse([]));
    if (url === '/api/changes') return Promise.resolve(jsonResponse(multiChanges));
    if (url === '/api/evidence') return Promise.resolve(jsonResponse({files:[{path:'a.go',type:'modified',excerpt:'@@ -1 +1 @@\n-a\n+a2'},{path:'b.go',type:'modified',excerpt:'@@ -1 +1 @@\n-b\n+b2'}],skipped:[]}));
    throw new Error(`unexpected ${url}`);
  };
  const {elements,load}=environment(fetch); load('graph.js'); load('quiz.js'); load('app.js');
  await new Promise(resolve=>setImmediate(resolve));
  elements.get('tab-code').listeners.click();
  await new Promise(resolve=>setImmediate(resolve));
  assert.match(allText(elements.get('code-view')), /a\.go/);
  assert.match(allText(elements.get('code-view')), /1 \/ 2/);
  const header = elements.get('code-view').children.find(e=>e.className==='code-path');
  const nav = header.children.find(e=>e.className==='code-nav');
  const nextBtn = nav.children.find(e=>e.textContent.includes('Next'));
  nextBtn.listeners.click();
  await new Promise(resolve=>setImmediate(resolve));
  assert.match(allText(elements.get('code-view')), /b\.go/);
  assert.match(allText(elements.get('code-view')), /2 \/ 2/);
});

test('retry button click event does not pass DOM Event as fetch AbortSignal', async () => {
  const signals = [];
  const fetch = (url, options) => {
    if (url === '/api/harnesses') return Promise.resolve(jsonResponse([]));
    if (url === '/api/changes') return Promise.resolve(jsonResponse(changes));
    if (url === '/api/quiz/fresh') {
      signals.push(options?.signal);
      if (options?.signal !== undefined && typeof options.signal.aborted !== 'boolean') {
        return Promise.reject(new TypeError('Failed to convert value to AbortSignal'));
      }
      return Promise.resolve(jsonResponse({fresh:true}));
    }
    throw new Error(`unexpected ${url}`);
  };
  const {elements,context,load}=environment(fetch); load('graph.js'); load('quiz.js'); load('app.js');
  await new Promise(resolve=>setImmediate(resolve));
  context.result={snapshot:'a'.repeat(64),explanation:{quiz:sampleQuiz},evidence:{files:[]}};
  vm.runInContext('prepareQuiz(result)',context);
  await elements.get('quiz-start').listeners.click({type:'click'});
  const panel=elements.get('quiz-content');
  panel.children.find(e=>e.className==='quiz-options').children[1].listeners.click({type:'click'});
  panel.children.find(e=>e.className==='quiz-feedback').children.find(e=>e.textContent==='See score').listeners.click({type:'click'});
  await panel.children.find(e=>e.textContent==='Retry same questions').listeners.click({type:'click'});
  assert.deepEqual(signals, [undefined, undefined]);
  assert.ok(panel.children.some(e=>e.className==='quiz-options'));
});

test('lazy model discovery populates available models and preserves provider session state across switches', async () => {
  const savedPayloads = [];
  const fetch = (url, options) => {
    if (url === '/api/harnesses') return Promise.resolve(jsonResponse([]));
    if (url === '/api/changes') return Promise.resolve(jsonResponse(changes));
    if (url === '/api/provider/models') {
      return Promise.resolve(jsonResponse({
        provider: 'groq',
        verified: true,
        key_saved: true,
        models: ['openai/gpt-oss-120b', 'qwen/qwen3.8-27b', 'openai/gpt-oss-20b'],
      }));
    }
    if (url === '/api/provider') {
      const parsed = JSON.parse(options.body);
      savedPayloads.push(parsed);
      return Promise.resolve(jsonResponse({
        configured: true,
        verified: parsed.endpoint.includes('groq'),
        key_saved: true,
      }));
    }
    throw new Error(`unexpected ${url}`);
  };
  const {elements,load}=environment(fetch); load('graph.js'); load('quiz.js'); load('app.js');
  await new Promise(resolve=>setImmediate(resolve));
  elements.get('provider-choice').value = 'groq';
  elements.get('provider-choice').listeners.change();
  elements.get('provider-key').value = 'test-session-key';
  await elements.get('provider-discover').listeners.click();
  assert.equal(elements.get('provider-model-select-label').hidden, false);
  assert.equal(elements.get('provider-model-select').children.length, 4);
  elements.get('provider-model-select').value = 'qwen/qwen3.8-27b';
  elements.get('provider-model-select').listeners.change();
  assert.equal(elements.get('provider-model').value, 'qwen/qwen3.8-27b');
  await elements.get('provider-connect').listeners.click();
  assert.match(elements.get('provider-badge').textContent, /Groq · qwen\/qwen3\.8-27b \(verified\)/);
  assert.equal(elements.get('provider-key').placeholder, 'Saved for session (leave blank to keep)');

  // Switch to OpenAI (unverified) and back to Groq
  elements.get('provider-choice').value = 'openai';
  elements.get('provider-choice').listeners.change();
  assert.equal(elements.get('provider-model-select-label').hidden, true);
  elements.get('provider-choice').value = 'groq';
  elements.get('provider-choice').listeners.change();
  assert.equal(elements.get('provider-model').value, 'qwen/qwen3.8-27b');
  assert.equal(elements.get('provider-model-select-label').hidden, false);
});

test('coding agent selection supports Codex, Claude Code, and Gemini CLI without API keys', async () => {
  const calls = [];
  const fetch = async (url, options) => {
    calls.push({url, body: options?.body && JSON.parse(options.body)});
    if (url === '/api/harnesses') {
      return jsonResponse([
        {name:'codex',display:'Codex CLI',installed:true,supported:true,version:'0.166.0',auth:'signed_in'},
        {name:'claude',display:'Claude Code',installed:false,supported:true},
        {name:'gemini',display:'Gemini CLI',installed:false,supported:true},
      ]);
    }
    if (url === '/api/changes') return jsonResponse(changes);
    if (url === '/api/provider/models') {
      const parsed = JSON.parse(options.body);
      const display = parsed.harness === 'codex' ? 'Codex CLI' : (parsed.harness === 'claude' ? 'Claude Code' : 'Gemini CLI');
      return jsonResponse({
        provider: parsed.harness,
        verified: parsed.harness === 'codex',
        models: [],
        message: `${display} found on PATH.`,
      });
    }
    if (url === '/api/provider') {
      const parsed = JSON.parse(options.body);
      return jsonResponse({
        configured: true,
        verified: parsed.harness === 'codex',
        key_saved: false,
        provider: parsed.harness,
        model: parsed.model,
      });
    }
    throw new Error(`unexpected ${url}`);
  };

  const {elements,load}=environment(fetch); load('graph.js'); load('quiz.js'); load('app.js');
  await new Promise(resolve=>setImmediate(resolve));

  // Check initial harness status text
  assert.match(elements.get('harness-status').textContent, /Codex CLI: installed, signed in, available for setup/);
  assert.match(elements.get('harness-status').textContent, /Claude Code: not installed, available for setup/);

  // Codex CLI is default
  assert.equal(elements.get('provider-key-label').hidden, true);
  assert.match(elements.get('provider-model').placeholder, /Use Codex CLI default/);
  assert.equal(elements.get('provider-discover').textContent, 'Check CLI');

  await elements.get('provider-discover').listeners.click();
  assert.match(elements.get('provider-status').textContent, /Codex CLI found on PATH/);

  await elements.get('provider-connect').listeners.click();
  assert.match(elements.get('provider-badge').textContent, /Codex CLI · default \(verified\)/);

  // Switch to Claude Code
  elements.get('provider-choice').value = 'claude';
  elements.get('provider-choice').listeners.change();
  assert.equal(elements.get('provider-key-label').hidden, true);
  assert.match(elements.get('provider-model').placeholder, /Use Claude Code default/);

  // Switch to Gemini CLI
  elements.get('provider-choice').value = 'gemini-cli';
  elements.get('provider-choice').listeners.change();
  assert.equal(elements.get('provider-key-label').hidden, true);
  assert.match(elements.get('provider-model').placeholder, /Use Gemini CLI default/);
});
