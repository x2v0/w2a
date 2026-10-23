package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// writeHTMLProject writes html-project/index.html + mtapi-debug.js.
func writeHTMLProject(html string) error {
	if err := os.MkdirAll(htmlProjectDir, 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	if err := os.WriteFile(filepath.Join(htmlProjectDir, "index.html"), []byte(html), 0644); err != nil {
		return fmt.Errorf("write index.html: %w", err)
	}
	if err := os.WriteFile(filepath.Join(htmlProjectDir, "mtapi-debug.js"), []byte(mtapiDebugJS), 0644); err != nil {
		return fmt.Errorf("write mtapi-debug.js: %w", err)
	}
	return nil
}

const chatHTML = `<!DOCTYPE html>
<html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Chat</title>
<style>
body{margin:0;font-family:system-ui,sans-serif;background:#0f0f1a;color:#e0e0e0;height:100vh;display:flex;flex-direction:column}
header{padding:12px 16px;background:#1a1a2e;border-bottom:1px solid #2a2a3e;display:flex;justify-content:space-between}
h1{margin:0;font-size:16px}button{padding:6px 12px;background:#2563eb;color:#fff;border:none;border-radius:6px;cursor:pointer}
#msgs{flex:1;overflow-y:auto;padding:16px;display:flex;flex-direction:column;gap:10px}
.m{padding:10px 14px;border-radius:12px;max-width:85%;white-space:pre-wrap;word-wrap:break-word;font-size:14px}
.m.user{align-self:flex-end;background:#2563eb;color:#fff}
.m.assistant{align-self:flex-start;background:#1e1e2e;border:1px solid #2a2a3e}
.m.error{align-self:center;background:#7f1d1d;color:#fecaca}
.inp{padding:12px;background:#1a1a2e;border-top:1px solid #2a2a3e;display:flex;gap:8px}
#in{flex:1;padding:10px;background:#0f0f1a;color:#e0e0e0;border:1px solid #4a4a5e;border-radius:8px;font-family:inherit;font-size:14px;resize:none;min-height:44px}
dialog{background:#1a1a2e;color:#e0e0e0;border:1px solid #2a2a3e;border-radius:12px;padding:20px;width:90%;max-width:400px}
dialog label{display:block;margin:10px 0 4px;font-size:12px;color:#a0a0b0}
dialog input{width:100%;padding:8px;background:#0f0f1a;color:#e0e0e0;border:1px solid #4a4a5e;border-radius:6px;font-family:inherit}
.btnrow{margin-top:20px;display:flex;gap:8px;justify-content:flex-end}
</style></head><body>
<header><h1>Chat</h1><button id="st">Settings</button></header>
<div id="msgs"></div>
<div class="inp"><textarea id="in" placeholder="Type a message... (Shift+Enter = newline)"></textarea><button id="sd">Send</button></div>
<dialog id="dlg"><form method="dialog">
<label>API endpoint (OpenAI-compatible)</label><input id="ep" placeholder="https://api.deepseek.com/v1/chat/completions">
<label>API key</label><input id="ak" type="password" placeholder="sk-...">
<label>Model</label><input id="ml" placeholder="deepseek-chat">
<div class="btnrow"><button value="cancel">Cancel</button><button id="sv" value="save">Save</button></div>
</form></dialog>
<script src="mtapi-debug.js"></script>
<script>MtapiDebug.attach({container:document.body});</script>
<script>
const S={endpoint:'',apikey:'',model:'deepseek-chat'};
const MSGS=[];
function load(){try{Object.assign(S,JSON.parse(localStorage.getItem('chat-cfg')||'{}'));MSGS.push(...JSON.parse(localStorage.getItem('chat-msgs')||'[]'));}catch(e){}}
function save(){localStorage.setItem('chat-cfg',JSON.stringify(S));localStorage.setItem('chat-msgs',JSON.stringify(MSGS));}
function render(){const b=document.getElementById('msgs');b.innerHTML='';MSGS.forEach(m=>{const d=document.createElement('div');d.className='m '+m.role;d.textContent=m.content;b.appendChild(d);});b.scrollTop=b.scrollHeight;}
async function send(){const inp=document.getElementById('in'),btn=document.getElementById('sd');const text=inp.value.trim();if(!text)return;if(!S.endpoint||!S.apikey){alert('Configure Settings');return;}
MSGS.push({role:'user',content:text});inp.value='';render();btn.disabled=true;
try{const r=await fetch(S.endpoint,{method:'POST',headers:{'Content-Type':'application/json','Authorization':'Bearer '+S.apikey},body:JSON.stringify({model:S.model,messages:MSGS.filter(m=>m.role!=='error').map(m=>({role:m.role,content:m.content}))})});
if(!r.ok)throw new Error('HTTP '+r.status+': '+(await r.text()).slice(0,200));
const j=await r.json();const rep=(j.choices&&j.choices[0]&&j.choices[0].message&&j.choices[0].message.content)||'(empty)';
MSGS.push({role:'assistant',content:rep});}catch(e){MSGS.push({role:'error',content:'Error: '+e.message});}
save();render();btn.disabled=false;}
document.getElementById('sd').onclick=send;
document.getElementById('in').addEventListener('keydown',e=>{if(e.key==='Enter'&&!e.shiftKey){e.preventDefault();send();}});
document.getElementById('st').onclick=()=>{document.getElementById('ep').value=S.endpoint;document.getElementById('ak').value=S.apikey;document.getElementById('ml').value=S.model;document.getElementById('dlg').showModal();};
document.getElementById('sv').onclick=()=>{S.endpoint=document.getElementById('ep').value.trim();S.apikey=document.getElementById('ak').value.trim();S.model=document.getElementById('ml').value.trim()||'deepseek-chat';save();};
load();render();
</script></body></html>`

const notesHTML = `<!DOCTYPE html>
<html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Notes</title>
<style>
body{margin:0;font-family:system-ui,sans-serif;background:#0f0f1a;color:#e0e0e0;height:100vh;display:flex}
#side{width:220px;background:#1a1a2e;border-right:1px solid #2a2a3e;display:flex;flex-direction:column}
#side h2{margin:0;padding:12px 16px;font-size:14px;border-bottom:1px solid #2a2a3e}
#new{margin:8px;padding:8px;background:#2563eb;color:#fff;border:none;border-radius:6px;cursor:pointer}
#list{flex:1;overflow-y:auto}
.item{padding:10px 16px;cursor:pointer;border-bottom:1px solid #2a2a3e;font-size:13px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.item:hover{background:#252538}
.item.active{background:#2563eb;color:#fff}
#main{flex:1;display:flex;flex-direction:column}
#title{padding:8px 16px;background:#1a1a2e;border:none;border-bottom:1px solid #2a2a3e;color:#e0e0e0;font-size:14px;font-family:inherit}
#ed{flex:1;padding:16px;background:#0f0f1a;color:#e0e0e0;border:none;font-family:monospace;font-size:14px;resize:none}
#preview{flex:1;padding:16px;overflow-y:auto;border-top:1px solid #2a2a3e;display:none;font-size:14px;line-height:1.6}
#del{margin:8px 16px;padding:6px 12px;background:#7f1d1d;color:#fff;border:none;border-radius:6px;cursor:pointer;font-size:12px}
</style></head><body>
<div id="side"><h2>Notes</h2><button id="new">+ New</button><div id="list"></div></div>
<div id="main"><input id="title" placeholder="Title"><textarea id="ed" placeholder="Markdown notes..."></textarea><div id="preview"></div><button id="del">🗑 Delete</button></div>
<script src="mtapi-debug.js"></script>
<script>MtapiDebug.attach({container:document.body});</script>
<script>
const NOTES=JSON.parse(localStorage.getItem('notes')||'[]');
let cur=-1;
function saveAll(){localStorage.setItem('notes',JSON.stringify(NOTES));}
function list(){const l=document.getElementById('list');l.innerHTML='';NOTES.forEach((n,i)=>{const d=document.createElement('div');d.className='item'+(i===cur?' active':'');d.textContent=n.title||'(untitled)';d.onclick=()=>open(i);l.appendChild(d);});}
function open(i){cur=i;if(i<0){document.getElementById('title').value='';document.getElementById('ed').value='';return;}const n=NOTES[i];document.getElementById('title').value=n.title||'';document.getElementById('ed').value=n.body||'';list();}
function upd(){if(cur<0)return;NOTES[cur].title=document.getElementById('title').value;NOTES[cur].body=document.getElementById('ed').value;saveAll();list();}
document.getElementById('title').oninput=upd;
document.getElementById('ed').oninput=upd;
document.getElementById('new').onclick=()=>{NOTES.unshift({title:'',body:''});saveAll();open(0);};
document.getElementById('del').onclick=()=>{if(cur<0)return;if(!confirm('Delete?'))return;NOTES.splice(cur,1);saveAll();cur=-1;list();document.getElementById('title').value='';document.getElementById('ed').value='';};
if(NOTES.length)open(0);else list();
</script></body></html>`

const dashboardHTML = `<!DOCTYPE html>
<html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Dashboard</title>
<style>
body{margin:0;font-family:system-ui,sans-serif;background:#0f0f1a;color:#e0e0e0;min-height:100vh}
header{padding:12px 16px;background:#1a1a2e;border-bottom:1px solid #2a2a3e}
h1{margin:0 0 8px;font-size:16px}
#bar{display:flex;gap:8px}
#u{flex:1;padding:8px;background:#0f0f1a;color:#e0e0e0;border:1px solid #4a4a5e;border-radius:6px;font-family:monospace;font-size:12px}
#go{padding:8px 16px;background:#2563eb;color:#fff;border:none;border-radius:6px;cursor:pointer}
#stats{padding:8px 16px;font-size:11px;color:#a0a0b0}
#content{padding:16px;overflow-x:auto}
table{border-collapse:collapse;width:100%;font-size:12px}
th,td{padding:6px 8px;border:1px solid #2a2a3e;text-align:left;max-width:200px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
th{background:#1a1a2e;font-weight:600}
tr:nth-child(even){background:#141425}
.recent{display:flex;gap:6px;flex-wrap:wrap;padding:0 16px 8px}
.recent button{padding:4px 10px;background:#252538;color:#a0a0b0;border:1px solid #3a3a4e;border-radius:6px;cursor:pointer;font-size:11px}
</style></head><body>
<header><h1>Dashboard</h1><div id="bar"><input id="u" placeholder="API URL (e.g. http://127.0.0.1:7788/battery?token=...)" value=""><button id="go">Fetch</button></div></header>
<div class="recent" id="rec"></div>
<div id="stats"></div>
<div id="content"></div>
<script src="mtapi-debug.js"></script>
<script>MtapiDebug.attach({container:document.body});</script>
<script>
const REC=JSON.parse(localStorage.getItem('dash-recent')||'[]');
function showRecent(){const r=document.getElementById('rec');r.innerHTML='';REC.slice(0,5).forEach(u=>{const b=document.createElement('button');b.textContent=u.length>40?u.slice(0,40)+'…':u;b.title=u;b.onclick=()=>{document.getElementById('u').value=u;fetch1(u);};r.appendChild(b);});}
function esc(v){if(v==null)return'';if(typeof v==='object')return JSON.stringify(v).slice(0,100);return String(v).slice(0,100);}
function renderTable(obj){if(Array.isArray(obj)){if(!obj.length)return'<p>(empty array)</p>';const cols=[...new Set(obj.flatMap(o=>o&&typeof o==='object'?Object.keys(o):[]))];let h='<table><thead><tr>';cols.forEach(c=>h+='<th>'+esc(c)+'</th>');h+='</tr></thead><tbody>';obj.forEach(r=>{h+='<tr>';cols.forEach(c=>h+='<td>'+esc(r?r[c]:'')+'</td>');h+='</tr>';});return h+'</tbody></table>';}
if(obj&&typeof obj==='object'){let h='<table><tbody>';Object.entries(obj).forEach(([k,v])=>{h+='<tr><th>'+esc(k)+'</th><td>'+esc(v)+'</td></tr>';});return h+'</tbody></table>';}
return '<pre>'+esc(obj)+'</pre>';}
async function fetch1(u){if(!u)return;const t0=performance.now();
try{const r=await fetch(u);const ms=Math.round(performance.now()-t0);const text=await r.text();
let j;try{j=JSON.parse(text);}catch(e){j=text;}
document.getElementById('stats').textContent='HTTP '+r.status+' · '+ms+'ms · '+text.length+'b';
document.getElementById('content').innerHTML=renderTable(j&&j.data!==undefined?j.data:j);
if(!REC.includes(u)){REC.unshift(u);localStorage.setItem('dash-recent',JSON.stringify(REC.slice(0,10)));showRecent();}
}catch(e){document.getElementById('stats').textContent='Error';document.getElementById('content').innerHTML='<p style="color:#fecaca">'+e.message+'</p>';}}
document.getElementById('go').onclick=()=>fetch1(document.getElementById('u').value.trim());
document.getElementById('u').addEventListener('keydown',e=>{if(e.key==='Enter')fetch1(e.target.value.trim());});
showRecent();
</script></body></html>`

func scaffoldChat(name string) error {
	return writeHTMLProject(chatHTML)
}

func scaffoldNotes(name string) error {
	return writeHTMLProject(notesHTML)
}

func scaffoldDashboard(name string) error {
	return writeHTMLProject(dashboardHTML)
}
