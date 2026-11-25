package main

// webIndexHTML is the single-page GUI served at "/". Plain HTML + vanilla JS
// (no build step, no external assets) so the binary stays self-contained.
//
// NOTE: this is a Go raw string (backtick-delimited), so the JS below must not
// use backtick template literals — string concatenation only.
const webIndexHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>PhantomDeck</title>
<style>
  :root{
    --bg:#0d1117; --panel:#161b22; --panel2:#1c232c; --border:#2b333d;
    --fg:#c9d1d9; --dim:#8b949e; --accent:#bd93f9; --accent2:#7c3aed;
    --ok:#3fb950; --warn:#d29922; --red:#f85149; --cyan:#39c5cf;
    --mono:ui-monospace,SFMono-Regular,Menlo,Consolas,monospace;
  }
  *{box-sizing:border-box}
  body{margin:0;background:var(--bg);color:var(--fg);font:14px/1.5 system-ui,Segoe UI,Roboto,sans-serif}
  header{display:flex;align-items:center;gap:16px;padding:10px 18px;border-bottom:1px solid var(--border);background:var(--panel);flex-wrap:wrap}
  header h1{font-size:16px;margin:0;letter-spacing:.5px}
  header h1 span{color:var(--accent)}
  .nav{display:flex;gap:4px}
  .nav button{padding:6px 14px;border:1px solid transparent;border-radius:6px;background:transparent;color:var(--dim);cursor:pointer;font-size:13px}
  .nav button.active{color:#fff;background:var(--accent2);border-color:var(--accent)}
  .proj{margin-left:auto;display:flex;align-items:center;gap:6px}
  .proj select,.proj input{padding:6px 8px;border:1px solid var(--border);border-radius:6px;background:var(--bg);color:var(--fg);font-size:13px}
  .btn{padding:7px 14px;border:1px solid var(--accent);border-radius:6px;background:var(--accent2);color:#fff;cursor:pointer;font-size:13px}
  .btn.ghost{background:transparent;color:var(--accent);border-color:var(--border)}
  .btn.sm{padding:3px 9px;font-size:12px}
  .btn:disabled{opacity:.5;cursor:not-allowed}
  .view{display:none}
  .view.active{display:block}
  main{padding:18px 22px}
  .muted{color:var(--dim)}
  .card{background:var(--panel);border:1px solid var(--border);border-radius:10px;padding:16px;margin-bottom:16px}
  .card h3{margin:0 0 12px;font-size:14px}
  .kv{display:grid;grid-template-columns:140px 1fr;gap:6px 14px;font-size:13px}
  .kv .k{color:var(--dim)}
  .mono{font-family:var(--mono)}
  .pill{display:inline-block;padding:1px 8px;border-radius:999px;font-size:11px;border:1px solid var(--border)}
  .pill.vpn{background:rgba(63,185,80,.15);border-color:var(--ok);color:var(--ok)}
  .dot{width:9px;height:9px;border-radius:50%;background:var(--dim);display:inline-block}
  .dot.on{background:var(--ok);box-shadow:0 0 6px var(--ok)}
  .row{display:flex;gap:8px;align-items:center;flex-wrap:wrap}
  pre{font-family:var(--mono);font-size:12px;white-space:pre-wrap;word-break:break-all}

  /* Build view */
  .buildwrap{display:grid;grid-template-columns:240px 1fr;gap:0;height:calc(100vh - 110px)}
  .sidebar{border-right:1px solid var(--border);background:var(--panel);overflow:auto;padding:10px}
  .cats{display:flex;flex-wrap:wrap;gap:6px;margin-bottom:10px}
  .cat{padding:4px 9px;border:1px solid var(--border);border-radius:6px;cursor:pointer;font-size:12px;color:var(--dim);background:var(--panel2)}
  .cat.active{color:#fff;border-color:var(--accent);background:var(--accent2)}
  .search{width:100%;padding:7px 9px;border:1px solid var(--border);border-radius:6px;background:var(--bg);color:var(--fg);margin-bottom:8px;font-size:13px}
  .tool{padding:7px 9px;border-radius:6px;cursor:pointer;border:1px solid transparent}
  .tool:hover{background:var(--panel2)}
  .tool.active{background:var(--panel2);border-color:var(--accent)}
  .tool .tn{font-weight:600}
  .tool .td{color:var(--dim);font-size:12px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
  .detail{overflow:auto;padding:0 22px}
  .flag{display:grid;grid-template-columns:200px 1fr;gap:12px;align-items:center;padding:8px 0;border-bottom:1px solid var(--border)}
  .flag .lbl{font-family:var(--mono);font-size:13px}
  .flag .lbl .req{color:var(--warn)}
  .flag .desc{color:var(--dim);font-size:12px;margin-top:2px}
  .flag input[type=text],.flag select{width:100%;padding:6px 8px;border:1px solid var(--border);border-radius:6px;background:var(--bg);color:var(--fg);font-family:var(--mono);font-size:13px}
  .flag input[type=checkbox]{width:18px;height:18px;accent-color:var(--accent)}
  .cmdbar{position:sticky;bottom:0;margin-top:18px;background:var(--panel);border:1px solid var(--border);border-radius:8px;padding:12px}
  .cmdbar pre{margin:0 0 10px;color:#e6edf3}
  .missing{color:var(--warn);font-size:12px;margin-top:6px}

  /* VPN */
  .vpn pre{max-height:180px;overflow:auto;background:var(--bg);border:1px solid var(--border);border-radius:6px;padding:8px;margin:8px 0 0}

  /* Map */
  .mapcols{display:grid;grid-template-columns:1fr 1fr;gap:16px}
  .node{border:1px solid var(--border);border-radius:8px;padding:10px;margin-bottom:10px;background:var(--panel)}
  .node.rec{border-left:3px solid var(--cyan)}
  .node.exploit{border-left:3px solid var(--red)}
  .node.run{border-left:3px solid var(--ok)}
  .node .nh{display:flex;justify-content:space-between;gap:8px;font-size:12px;color:var(--dim);margin-bottom:6px}
  .node pre{margin:0;background:var(--bg);border:1px solid var(--border);border-radius:6px;padding:7px}
  .node .na{margin-top:7px;display:flex;gap:6px}
  .hostgrp{margin-bottom:14px}
  .hostgrp h4{margin:0 0 6px;font-size:13px;color:var(--warn)}
  textarea{width:100%;min-height:120px;padding:8px;border:1px solid var(--border);border-radius:6px;background:var(--bg);color:var(--fg);font-family:var(--mono);font-size:12px}

  .toast{position:fixed;bottom:18px;right:18px;background:var(--accent2);color:#fff;padding:9px 14px;border-radius:6px;opacity:0;transition:opacity .2s;z-index:50}
  .toast.show{opacity:1}
</style>
</head>
<body>
<header>
  <h1>Phantom<span>Deck</span></h1>
  <nav class="nav" id="nav">
    <button data-view="home" class="active">Home</button>
    <button data-view="build">Build</button>
    <button data-view="map">Map</button>
  </nav>
  <div class="proj">
    <span class="muted">Theme</span>
    <select id="themeSel" title="Omarchy theme"></select>
    <span class="muted">Project</span>
    <select id="projSel"></select>
    <button class="btn ghost sm" id="projNew">+ New</button>
  </div>
</header>

<main>
  <!-- HOME -->
  <section class="view active" id="view-home">
    <div class="card">
      <h3>Who am I</h3>
      <div class="kv" id="whoami"><span class="muted">loading…</span></div>
    </div>
    <div class="card vpn">
      <h3><span class="dot" id="vpnDot"></span> VPN profile</h3>
      <div class="row">
        <input type="file" id="vpnFile" accept=".ovpn,.conf">
        <button class="btn ghost" id="vpnUpload">Upload</button>
        <button class="btn" id="vpnConnect" disabled>Connect</button>
        <button class="btn ghost" id="vpnDisconnect" disabled>Disconnect</button>
        <span class="muted" id="vpnState">no profile</span>
      </div>
      <pre id="vpnLog" style="display:none"></pre>
      <p class="muted" style="font-size:12px;margin:8px 0 0">Connecting runs openvpn — needs root (launch with sudo, or passwordless sudo for openvpn).</p>
    </div>
  </section>

  <!-- BUILD -->
  <section class="view" id="view-build">
    <div class="buildwrap">
      <aside class="sidebar">
        <div class="cats" id="cats"></div>
        <input class="search" id="search" placeholder="filter tools…">
        <div id="tools"></div>
      </aside>
      <div class="detail" id="detail">
        <p class="muted">Pick a tool from the left to configure its flags.</p>
      </div>
    </div>
  </section>

  <!-- MAP -->
  <section class="view" id="view-map">
    <div class="card">
      <h3>Import scan → attack vectors</h3>
      <p class="muted" style="font-size:12px;margin-top:0">Paste raw nmap <span class="mono">-oN</span> output (or NmapResponder JSON). Recommended commands land on the map.</p>
      <textarea id="nmapInput" placeholder="Nmap scan report for ...&#10;22/tcp open ssh OpenSSH 8.2p1 ..."></textarea>
      <div class="row" style="margin-top:8px">
        <input class="search" id="nmapTarget" style="width:200px;margin:0" placeholder="target override (optional)">
        <label class="muted" style="font-size:12px"><input type="checkbox" id="nmapExploits"> searchsploit exploits</label>
        <button class="btn" id="nmapImport">Import to map</button>
      </div>
    </div>
    <div class="row" style="margin-bottom:12px">
      <button class="btn ghost sm" id="canvasDl">Download .canvas</button>
      <button class="btn ghost sm" id="mapRefresh">Refresh</button>
      <button class="btn ghost sm" id="mapClear">Clear map</button>
      <span class="muted mono" id="canvasPath" style="font-size:11px"></span>
    </div>
    <div class="mapcols">
      <div>
        <h3 style="margin:0 0 10px">Attack vectors <span class="muted" style="font-size:12px">(recommended)</span></h3>
        <div id="recCol"><p class="muted">Nothing yet — import a scan above.</p></div>
      </div>
      <div>
        <h3 style="margin:0 0 10px">Path taken <span class="muted" style="font-size:12px">(commands you ran)</span></h3>
        <div id="runCol"><p class="muted">Copy a command in Build, or "Mark run" a vector.</p></div>
      </div>
    </div>
  </section>
</main>
<div class="toast" id="toast"></div>

<script>
var CATS=[], curCat=0, curToolIdx=-1, working=null, cmdTimer=null;

function esc(s){return (s||'').replace(/[&<>"]/g,function(c){return {'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;'}[c];});}
function toast(m){var t=document.getElementById('toast');t.textContent=m;t.classList.add('show');setTimeout(function(){t.classList.remove('show');},1500);}
function api(path,opts){return fetch(path,opts).then(function(r){if(!r.ok)return r.text().then(function(t){throw new Error(t||r.status);});return r.json();});}

// ── view switching ──
document.getElementById('nav').addEventListener('click',function(e){
  var b=e.target.closest('button'); if(!b)return;
  var v=b.dataset.view;
  document.querySelectorAll('.nav button').forEach(function(x){x.classList.toggle('active',x===b);});
  document.querySelectorAll('.view').forEach(function(x){x.classList.toggle('active',x.id==='view-'+v);});
  if(v==='home')loadWhoami();
  if(v==='map')loadMap();
});

// ── projects ──
function loadProjects(){
  return api('/api/projects').then(function(j){renderProjects(j);});
}
function renderProjects(j){
  var sel=document.getElementById('projSel'); sel.innerHTML='';
  (j.projects||[]).forEach(function(p){
    var o=document.createElement('option'); o.value=p; o.textContent=p;
    if(p===j.current)o.selected=true; sel.appendChild(o);
  });
}
document.getElementById('projSel').addEventListener('change',function(e){
  api('/api/projects/select',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({name:e.target.value})})
   .then(function(j){renderProjects(j);loadMap();loadWhoami();toast('Switched to '+j.current);});
});
document.getElementById('projNew').onclick=function(){
  var name=prompt('New project name (engagement):'); if(!name)return;
  api('/api/projects',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({name:name})})
   .then(function(j){renderProjects(j);loadMap();loadWhoami();toast('Created '+j.current);});
};

// ── home / whoami ──
function loadWhoami(){
  api('/api/whoami').then(function(j){
    var el=document.getElementById('whoami'); var h='';
    var u=j.user||{};
    h+='<span class="k">User</span><span class="mono">'+esc(u.username||'?')+(u.root==='true'?' <span class="pill" style="border-color:var(--red);color:var(--red)">root</span>':'')+'</span>';
    h+='<span class="k">Host</span><span class="mono">'+esc(j.hostname||'?')+'</span>';
    h+='<span class="k">Project</span><span class="mono">'+esc(j.project||'?')+'</span>';
    (j.interfaces||[]).forEach(function(ifc){
      h+='<span class="k">'+esc(ifc.name)+'</span><span class="mono">'+esc((ifc.addrs||[]).join(', '))+(ifc.vpn?' <span class="pill vpn">VPN</span>':'')+'</span>';
    });
    el.innerHTML=h;
    if(j.vpn)applyVpn(j.vpn);
  }).catch(function(){});
}

// ── VPN ──
document.getElementById('vpnUpload').onclick=function(){
  var f=document.getElementById('vpnFile').files[0];
  if(!f){toast('Choose a .ovpn file');return;}
  var fd=new FormData(); fd.append('profile',f);
  fetch('/api/vpn/upload',{method:'POST',body:fd}).then(function(r){return r.json();})
   .then(function(s){applyVpn(s);toast('Profile uploaded');}).catch(function(){toast('Upload failed');});
};
document.getElementById('vpnConnect').onclick=function(){
  fetch('/api/vpn/connect',{method:'POST'}).then(function(r){return r.ok?r.json().then(applyVpn):r.text().then(toast);});
};
document.getElementById('vpnDisconnect').onclick=function(){
  fetch('/api/vpn/disconnect',{method:'POST'}).then(function(r){if(r.ok)r.json().then(applyVpn);});
};
function pollVpn(){
  fetch('/api/vpn/status').then(function(r){return r.ok?r.json():null;}).then(function(s){if(s)applyVpn(s);}).catch(function(){})
   .finally(function(){setTimeout(pollVpn,2500);});
}
function applyVpn(s){
  var dot=document.getElementById('vpnDot'),st=document.getElementById('vpnState'),log=document.getElementById('vpnLog');
  dot.className='dot'+(s.connected?' on':'');
  document.getElementById('vpnConnect').disabled=!s.profile||s.connected;
  document.getElementById('vpnDisconnect').disabled=!s.connected;
  if(!s.profile)st.textContent='no profile';
  else if(s.connected)st.textContent=s.profile+' — connected'+(s.uptime?(' ('+s.uptime+'s)'):'');
  else st.textContent=s.profile+' — ready';
  if(s.log){log.style.display='block';var atBottom=log.scrollTop+log.clientHeight>=log.scrollHeight-20;log.textContent=s.log;if(atBottom)log.scrollTop=log.scrollHeight;}
}

// ── build ──
function loadCatalogs(){
  return api('/api/catalogs').then(function(d){
    CATS=d; renderCats();
    var wi=CATS.findIndex(function(c){return c.Key==='web';});
    selectCat(wi<0?0:wi);
  });
}
function renderCats(){
  var el=document.getElementById('cats'); el.innerHTML='';
  CATS.forEach(function(c,i){
    var d=document.createElement('div'); d.className='cat'+(i===curCat?' active':''); d.textContent=c.Label;
    d.onclick=function(){selectCat(i);}; el.appendChild(d);
  });
}
function selectCat(i){curCat=i;curToolIdx=-1;renderCats();renderTools();document.getElementById('detail').innerHTML='<p class="muted">Pick a tool from the left to configure its flags.</p>';}
function renderTools(){
  var q=document.getElementById('search').value.toLowerCase();
  var el=document.getElementById('tools'); el.innerHTML='';
  (CATS[curCat].Tools||[]).forEach(function(t,idx){
    var hay=(t.Name+' '+t.Category+' '+t.Desc).toLowerCase();
    if(q&&hay.indexOf(q)<0)return;
    var d=document.createElement('div'); d.className='tool'+(idx===curToolIdx?' active':'');
    d.innerHTML='<div class="tn"></div><div class="td"></div>';
    d.querySelector('.tn').textContent=t.Name;
    d.querySelector('.td').textContent=(t.Category?'['+t.Category+'] ':'')+t.Desc;
    d.onclick=function(){selectTool(idx);}; el.appendChild(d);
  });
}
document.getElementById('search').addEventListener('input',renderTools);
function selectTool(idx){
  curToolIdx=idx; renderTools();
  working=JSON.parse(JSON.stringify(CATS[curCat].Tools[idx]));
  renderDetail(); refreshCommand();
}
function control(f,i){
  if(f.Kind===0)return '<input type="checkbox" data-fi="'+i+'"'+(f.Enabled?' checked':'')+'>';
  if(f.Kind===3){
    var o=(f.Choices||[]).map(function(c){return '<option'+(c===f.Value?' selected':'')+'>'+esc(c)+'</option>';}).join('');
    return '<select data-fi="'+i+'"><option value=""></option>'+o+'</select>';
  }
  return '<input type="text" data-fi="'+i+'" value="'+esc(f.Value||'')+'" placeholder="'+esc(f.Placeholder||'')+'">';
}
function renderDetail(){
  var t=working,el=document.getElementById('detail');
  var h='<h2 style="margin:0 0 2px">'+esc(t.Name)+'</h2><div class="muted mono" style="margin-bottom:12px">'+esc(t.Binary)+'</div>';
  (t.Flags||[]).forEach(function(f,i){
    var req=f.Required?' <span class="req">*</span>':'';
    h+='<div class="flag"><div><div class="lbl">'+esc(f.Flag||f.Name)+(f.Flag?'  <span class="muted">'+esc(f.Name)+'</span>':'')+req+'</div>'+
       (f.Desc?'<div class="desc">'+esc(f.Desc)+'</div>':'')+'</div><div>'+control(f,i)+'</div></div>';
  });
  el.innerHTML=h;
  el.querySelectorAll('[data-fi]').forEach(function(inp){
    var i=+inp.dataset.fi, ev=inp.type==='checkbox'?'change':'input';
    inp.addEventListener(ev,function(){
      var f=working.Flags[i];
      if(inp.type==='checkbox')f.Enabled=inp.checked; else f.Value=inp.value;
      refreshCommand();
    });
  });
  var bar=document.createElement('div'); bar.className='cmdbar';
  bar.innerHTML='<pre id="cmd"></pre><div class="row"><button class="btn" id="copy">Copy &amp; log to map</button><button class="btn ghost" id="copyonly">Copy only</button><span class="missing" id="missing"></span></div>';
  el.appendChild(bar);
  document.getElementById('copy').onclick=function(){doCopy(true);};
  document.getElementById('copyonly').onclick=function(){doCopy(false);};
}
function doCopy(log){
  var txt=document.getElementById('cmd').textContent;
  if(!txt)return;
  copyText(txt,log?'Copied + added to map':'Copied');
  if(log){
    api('/api/history',{method:'POST',headers:{'Content-Type':'application/json'},
      body:JSON.stringify({command:txt,tool:working.Name,catalog:CATS[curCat].Label})}).catch(function(){});
  }
}
function refreshCommand(){
  clearTimeout(cmdTimer);
  cmdTimer=setTimeout(function(){
    fetch('/api/command',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(working)})
     .then(function(r){return r.json();}).then(function(j){
      var c=document.getElementById('cmd'); if(c)c.textContent=j.command;
      var m=document.getElementById('missing');
      if(m)m.textContent=(j.missing&&j.missing.length)?('missing required: '+j.missing.join(', ')):'';
    });
  },120);
}

// ── map ──
function loadMap(){
  api('/api/history').then(function(j){renderMap(j.entries||[]);document.getElementById('canvasPath').textContent=j.canvas||'';});
}
function renderMap(entries){
  var rec=document.getElementById('recCol'), run=document.getElementById('runCol');
  var recs=entries.filter(function(e){return e.kind==='recommended';});
  var runs=entries.filter(function(e){return e.kind!=='recommended';});
  // recommended grouped by host
  if(!recs.length){rec.innerHTML='<p class="muted">Nothing yet — import a scan above.</p>';}
  else{
    var groups={}, order=[];
    recs.forEach(function(e){var k=e.host||'recommendations';if(!groups[k]){groups[k]=[];order.push(k);}groups[k].push(e);});
    var h='';
    order.forEach(function(k){
      h+='<div class="hostgrp"><h4>'+esc(k)+'</h4>';
      groups[k].forEach(function(e){h+=nodeHTML(e,true);});
      h+='</div>';
    });
    rec.innerHTML=h;
    rec.querySelectorAll('[data-run]').forEach(function(b){b.onclick=function(){markRun(b.dataset.run);};});
    rec.querySelectorAll('[data-copy]').forEach(function(b){b.onclick=function(){copyText(b.dataset.copy);};});
  }
  // run path
  if(!runs.length){run.innerHTML='<p class="muted">Copy a command in Build, or "Mark run" a vector.</p>';}
  else{
    run.innerHTML=runs.map(function(e,i){return (i>0?'<div class="muted" style="text-align:center">↓</div>':'')+nodeHTML(e,false);}).join('');
    run.querySelectorAll('[data-copy]').forEach(function(b){b.onclick=function(){copyText(b.dataset.copy);};});
  }
}
function nodeHTML(e,isRec){
  var cls=isRec?(e.service==='exploit'?'node exploit':'node rec'):'node run';
  var head=isRec?(esc((e.service||'')+' '+((e.host&&e.port)?(e.host+':'+e.port):(e.host||'')))):esc((e.tool||'command')+(e.catalog?' · '+e.catalog:''));
  var h='<div class="'+cls+'"><div class="nh"><span>'+head+'</span><span>'+esc(e.time||'')+'</span></div>';
  h+='<pre>'+esc(e.command)+'</pre>';
  if(e.note)h+='<div class="muted" style="font-size:11px;margin-top:5px">'+esc(e.note)+'</div>';
  h+='<div class="na"><button class="btn ghost sm" data-copy="'+esc(e.command)+'">Copy</button>';
  if(isRec)h+='<button class="btn sm" data-run="'+esc(e.command)+'">Mark run</button>';
  h+='</div></div>';
  return h;
}
function copyText(t,okMsg){
  okMsg=okMsg||'Copied';
  if(navigator.clipboard&&window.isSecureContext){
    navigator.clipboard.writeText(t).then(function(){toast(okMsg);}).catch(function(){legacyCopy(t,okMsg);});
    return;
  }
  legacyCopy(t,okMsg);
}
function legacyCopy(t,okMsg){
  var ta=document.createElement('textarea');
  ta.value=t; ta.setAttribute('readonly','');
  ta.style.position='fixed'; ta.style.top='-1000px'; ta.style.opacity='0';
  document.body.appendChild(ta); ta.focus(); ta.select();
  var ok=false; try{ok=document.execCommand('copy');}catch(e){}
  document.body.removeChild(ta);
  toast(ok?okMsg:'Copy failed');
}
function markRun(cmd){
  api('/api/history',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({command:cmd,tool:'from vector'})})
   .then(function(j){renderMap(j.entries||[]);toast('Added to path');});
}
document.getElementById('mapRefresh').onclick=loadMap;
document.getElementById('canvasDl').onclick=function(){window.location='/api/canvas';};
document.getElementById('mapClear').onclick=function(){
  if(!confirm('Clear all commands on this project map?'))return;
  api('/api/history/clear',{method:'POST'}).then(function(j){renderMap(j.entries||[]);toast('Cleared');});
};
document.getElementById('nmapImport').onclick=function(){
  var txt=document.getElementById('nmapInput').value.trim();
  if(!txt){toast('Paste nmap output first');return;}
  var qs=[]; if(document.getElementById('nmapExploits').checked)qs.push('exploits=1');
  var tgt=document.getElementById('nmapTarget').value.trim(); if(tgt)qs.push('target='+encodeURIComponent(tgt));
  var url='/api/import/nmap'+(qs.length?('?'+qs.join('&')):'');
  var ct=txt[0]==='{'?'application/json':'text/plain';
  toast('Importing…');
  api(url,{method:'POST',headers:{'Content-Type':ct},body:txt})
   .then(function(j){renderMap(j.entries||[]);document.getElementById('canvasPath').textContent=j.canvas||'';toast('Imported '+j.imported+' vectors');})
   .catch(function(e){toast('Import failed: '+e.message);});
};

// ── themes (Omarchy) ──
function hex2rgb(h){h=(h||'').replace('#','');if(h.length===3)h=h[0]+h[0]+h[1]+h[1]+h[2]+h[2];var n=parseInt(h||'0',16);return [(n>>16)&255,(n>>8)&255,n&255];}
function rgb2hex(r){return '#'+r.map(function(v){return ('0'+Math.max(0,Math.min(255,Math.round(v))).toString(16)).slice(-2);}).join('');}
function mix(a,b,t){var x=hex2rgb(a),y=hex2rgb(b);return rgb2hex([x[0]+(y[0]-x[0])*t,x[1]+(y[1]-x[1])*t,x[2]+(y[2]-x[2])*t]);}
function applyTheme(p){
  if(!p||!p.bg||!p.fg)return;
  var s=document.documentElement.style, set=function(k,v){if(v)s.setProperty(k,v);};
  var bg=p.bg,fg=p.fg,acc=p.accent||p.magenta||p.blue||fg;
  set('--bg',bg); set('--fg',fg);
  set('--panel',mix(bg,fg,0.05)); set('--panel2',mix(bg,fg,0.10));
  set('--border',mix(bg,fg,0.18)); set('--dim',mix(bg,fg,0.55));
  set('--accent',acc); set('--accent2',mix(acc,bg,0.35));
  set('--ok',p.green||acc); set('--warn',p.yellow||p.green||acc);
  set('--red',p.red||acc); set('--cyan',p.cyan||p.blue||acc);
}
function loadThemes(){
  return api('/api/themes').then(function(j){
    var sel=document.getElementById('themeSel'); sel.innerHTML='';
    (j.themes||[]).forEach(function(t){
      var o=document.createElement('option'); o.value=t; o.textContent=t; sel.appendChild(o);
    });
    var pick=localStorage.getItem('pd_theme')||j.current||(j.themes&&j.themes[0]);
    if(pick){sel.value=pick; setTheme(pick);}
  }).catch(function(){});
}
function setTheme(name){
  api('/api/theme?name='+encodeURIComponent(name)).then(applyTheme).catch(function(){});
}
document.getElementById('themeSel').addEventListener('change',function(e){
  localStorage.setItem('pd_theme',e.target.value); setTheme(e.target.value); toast('Theme: '+e.target.value);
});

// ── boot ──
loadThemes();
loadProjects().then(loadWhoami);
loadCatalogs();
pollVpn();
</script>
</body>
</html>`
