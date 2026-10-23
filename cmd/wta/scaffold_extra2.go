package main

const rssHTML = `<!DOCTYPE html>
<html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>RSS</title>
<style>
body{margin:0;font-family:system-ui,sans-serif;background:#0f0f1a;color:#e0e0e0;min-height:100vh}
header{padding:12px 16px;background:#1a1a2e;border-bottom:1px solid #2a2a3e;position:sticky;top:0;z-index:10}
h1{margin:0 0 8px;font-size:16px}
#bar{display:flex;gap:8px}
#u{flex:1;padding:8px;background:#0f0f1a;color:#e0e0e0;border:1px solid #4a4a5e;border-radius:6px;font-family:inherit;font-size:12px}
#go{padding:8px 16px;background:#2563eb;color:#fff;border:none;border-radius:6px;cursor:pointer}
#tabs{padding:8px 16px;display:flex;gap:6px;overflow-x:auto}
#tabs button{padding:4px 10px;background:#252538;color:#a0a0b0;border:1px solid #3a3a4e;border-radius:6px;cursor:pointer;font-size:11px;white-space:nowrap}
#tabs button.active{background:#2563eb;color:#fff}
#list{padding:8px 16px}
.item{padding:12px;border-bottom:1px solid #2a2a3e;cursor:pointer}
.item:hover{background:#1a1a2e}
.title{font-weight:600;font-size:14px;margin-bottom:4px}
.meta{font-size:11px;color:#a0a0b0}
.desc{font-size:12px;color:#c0c0d0;margin-top:6px;line-height:1.5}
.modal{position:fixed;top:0;left:0;right:0;bottom:0;background:rgba(0,0,0,0.9);padding:20px;overflow-y:auto;display:none;z-index:100}
.modal.show{display:block}
.modal h2{margin:0 0 8px;font-size:16px}
.modal .close{float:right;background:#7f1d1d;color:#fff;border:none;padding:4px 10px;border-radius:6px;cursor:pointer}
.modal a{color:#60a5fa;word-break:break-all}
.modal .content{margin-top:16px;font-size:14px;line-height:1.6;white-space:pre-wrap}
</style></head><body>
<header><h1>RSS Reader</h1><div id="bar"><input id="u" placeholder="RSS/Atom URL (e.g. https://hnrss.org/frontpage)"><button id="go">Load</button></div></header>
<div id="tabs"></div>
<div id="list"></div>
<div class="modal" id="m"><button class="close" onclick="document.getElementById('m').classList.remove('show')">✕</button><h2 id="mt"></h2><a id="ml" target="_blank"></a><div class="content" id="mc"></div></div>
<script src="mtapi-debug.js"></script>
<script>MtapiDebug.attach({container:document.body});</script>
<script>
const FEEDS=JSON.parse(localStorage.getItem('rss-feeds')||'[]');
let cur=-1;
function saveAll(){localStorage.setItem('rss-feeds',JSON.stringify(FEEDS));}
function tabs(){const t=document.getElementById('tabs');t.innerHTML='';FEEDS.forEach((f,i)=>{const b=document.createElement('button');b.textContent=f.name;b.className=i===cur?'active':'';b.onclick=()=>{cur=i;load();};t.appendChild(b);});}
async function load(){if(cur<0)return;const f=FEEDS[cur];const list=document.getElementById('list');list.innerHTML='<p>Loading...</p>';
try{const r=await fetch('https://api.allorigins.win/raw?url='+encodeURIComponent(f.url));const text=await r.text();
const doc=new DOMParser().parseFromString(text,'text/xml');
const items=[...doc.querySelectorAll('item,entry')].slice(0,20);
list.innerHTML='';
items.forEach(it=>{const title=it.querySelector('title')?.textContent||'(no title)';
const linkEl=it.querySelector('link');const link=(linkEl&&linkEl.textContent)||(linkEl&&linkEl.getAttribute('href'))||'';
const desc=(it.querySelector('description,summary,content')?.textContent)||'';
const pub=(it.querySelector('pubDate,published,updated')?.textContent)||'';
const d=document.createElement('div');d.className='item';
d.innerHTML='<div class="title">'+esc(title)+'</div><div class="meta">'+esc(pub.slice(0,25))+'</div><div class="desc">'+esc(desc.replace(/<[^>]+>/g,'').slice(0,200))+'</div>';
d.onclick=()=>{document.getElementById('mt').textContent=title;const ml=document.getElementById('ml');ml.href=link;ml.textContent=link;document.getElementById('mc').textContent=desc.replace(/<[^>]+>/g,'');document.getElementById('m').classList.add('show');};
list.appendChild(d);});
}catch(e){list.innerHTML='<p style="color:#fecaca">Error: '+e.message+'</p>';}}
function esc(s){return String(s||'').replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'})[c]);}
document.getElementById('go').onclick=()=>{const u=document.getElementById('u').value.trim();if(!u)return;
const name=u.replace(/^https?:\/\//,'').split('/')[0];
FEEDS.push({name,url:u});saveAll();cur=FEEDS.length-1;tabs();load();document.getElementById('u').value='';};
tabs();if(FEEDS.length)load();
</script></body></html>`

const readerHTML = `<!DOCTYPE html>
<html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Reader</title>
<style>
body{margin:0;font-family:Georgia,serif;background:#f5f5f0;color:#222;min-height:100vh}
#hdr{padding:10px 16px;background:#2a2a3e;color:#e0e0e0;position:sticky;top:0;z-index:10;display:flex;gap:8px;font-family:system-ui,sans-serif;font-size:12px}
#hdr button,#hdr label{padding:6px 12px;background:#2563eb;color:#fff;border:none;border-radius:6px;cursor:pointer}
#hdr input[type=file]{display:none}
#hdr #st{flex:1;text-align:right;opacity:0.7;font-size:11px;align-self:center}
#content{padding:20px;max-width:800px;margin:0 auto;line-height:1.7;font-size:16px}
#content img{max-width:100%;height:auto}
#content pre{background:#e8e8e0;padding:10px;overflow-x:auto;font-size:13px}
#content code{background:#e8e8e0;padding:2px 4px;border-radius:3px;font-family:monospace;font-size:14px}
#content h1,#content h2,#content h3{font-family:system-ui,sans-serif;margin-top:1.5em}
#content blockquote{border-left:4px solid #999;margin:1em 0;padding-left:1em;color:#555}
</style></head><body>
<div id="hdr">
<label for="f">📂 Open</label><input type="file" id="f" accept=".md,.markdown,.txt,.html,.htm,text/*,application/pdf">
<button onclick="document.getElementById('content').style.fontSize=(parseInt(getComputedStyle(document.getElementById('content')).fontSize)+2)+'px'">A+</button>
<button onclick="document.getElementById('content').style.fontSize=(parseInt(getComputedStyle(document.getElementById('content')).fontSize)-2)+'px'">A-</button>
<span id="st">no file</span>
</div>
<div id="content"><p style="text-align:center;color:#888">Open a file: .md .markdown .txt .html .pdf</p></div>
<script src="mtapi-debug.js"></script>
<script>MtapiDebug.attach({container:document.body});</script>
<script>
function esc(s){return String(s||'').replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'})[c]);}
function md2html(md){let h=esc(md);
h=h.replace(/^### (.*)$/gm,'<h3>$1</h3>').replace(/^## (.*)$/gm,'<h2>$1</h2>').replace(/^# (.*)$/gm,'<h1>$1</h1>');
h=h.replace(/^> (.*)$/gm,'<blockquote>$1</blockquote>');
h=h.replace(/\x60\x60\x60([\s\S]*?)\x60\x60\x60/g,'<pre>$1</pre>');
h=h.replace(/\x60([^\x60\n]+)\x60/g,'<code>$1</code>');
h=h.replace(/\*\*([^*\n]+)\*\*/g,'<b>$1</b>').replace(/\*([^*\n]+)\*/g,'<i>$1</i>');
h=h.replace(/\[([^\]]+)\]\(([^)]+)\)/g,'<a href="$2" target="_blank">$1</a>');
h=h.replace(/^- (.*)$/gm,'<li>$1</li>');h=h.replace(/(<li>[\s\S]*?<\/li>)/g,'<ul>$1</ul>');
h=h.replace(/\n\n/g,'</p><p>');
return '<p>'+h+'</p>';}
document.getElementById('f').addEventListener('change',async e=>{
const file=e.target.files[0];if(!file)return;
document.getElementById('st').textContent=file.name+' ('+(file.size/1024).toFixed(1)+'KB)';
const name=file.name.toLowerCase();const c=document.getElementById('content');
if(name.endsWith('.pdf')){c.innerHTML='<p>PDF: <a href="'+URL.createObjectURL(file)+'" target="_blank">открыть в браузере</a> (native PDF viewer).</p>';return;}
const text=await file.text();
if(name.endsWith('.md')||name.endsWith('.markdown')){c.innerHTML=md2html(text);}
else if(name.endsWith('.html')||name.endsWith('.htm')){c.innerHTML=text;}
else{c.innerHTML='<pre style="white-space:pre-wrap;font-family:inherit">'+esc(text)+'</pre>';}
});
</script></body></html>`

func scaffoldRss(name string) error      { return writeHTMLProject(rssHTML) }
func scaffoldReader(name string) error   { return writeHTMLProject(readerHTML) }
