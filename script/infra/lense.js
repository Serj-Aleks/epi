// script/infra/lense.js — линзы карточки: ≡ контент со словарями, ↕ вкладки, ⊛ процесс, ◈ ранг, ⬡ таксоны, ⊞ слоты
// lensContent: ≡ контент — спутник Content или md, найденное внутри концепта подсвечено; ✎ — редактор спутников
async function lensContent(box, c) {
const bar = el("cbar"), body = el("cbody");
const view = async () => { bar.innerHTML = ""; const b = document.createElement("button"); b.textContent = "✎"; b.title = "redaction"; b.onclick = edit; bar.append(b);
  const f = cardFind && cardFind.name.toLowerCase() === c.name.toLowerCase() ? cardFind : null;
  const h = await cardRender(c.name, f && f.ext || c.content, []).catch(() => null); body.innerHTML = h ? h : '<div class="cempty">—</div>'; cardLinks(body); await dictMark(body, c.name); if (f) findMark(body, f.q); };
const edit = async () => { bar.innerHTML = "";
  const s = document.createElement("select"); satTypes.forEach(t => { const o = document.createElement("option"); o.value = o.textContent = t + (c.sats.includes(t) ? "" : " ·"); o.value = t; o.selected = t === c.content; s.append(o); });
  const ok = document.createElement("button"); ok.textContent = "✓"; ok.title = "record";
  const no = document.createElement("button"); no.textContent = "✕"; no.title = "cancel"; no.onclick = view;
  bar.append(s, ok, no);
  const ta = document.createElement("textarea"); ta.className = "cedit"; ta.spellcheck = false; body.innerHTML = ""; body.append(ta);
  const load = async () => { ta.value = (await api("/api/sat?name=" + encodeURIComponent(c.name) + "&ext=" + s.value)).text || ""; };
  s.onchange = load; await load();
  ok.onclick = async () => { try { Object.assign(c, await api("/api/sat", { name: c.name, ext: s.value, text: ta.value })); } catch (e) { note.textContent = e.message; return; } cardOpen(c.name); }; };
box.append(bar, body); await view(); }
// wordsOf: слова строки в нижнем регистре
function wordsOf(s) { return s.toLowerCase().split(/[\s\-_]+/).filter(Boolean); }
// wordEq: слова равны по первым d литерам, слово короче d — точно
function wordEq(a, b, d) { return a.length < d || b.length < d ? a === b : a.slice(0, d) === b.slice(0, d); }
// dictMark: в контенте отметить термины словарей концепта (Defs) — нечётко, по первым Deep литерам каждого слова
async function dictMark(root, name) {
const d = await api("/api/dict?name=" + encodeURIComponent(name)).catch(() => null);
if (!d || !d.terms.length) return;
const terms = d.terms.map(t => ({ ...t, ws: wordsOf(t.term) })).filter(t => t.ws.length);
const walk = document.createTreeWalker(root, NodeFilter.SHOW_TEXT, { acceptNode: n => n.parentElement.closest("a, code, pre, .dterm") ? NodeFilter.FILTER_REJECT : NodeFilter.FILTER_ACCEPT });
const nodes = []; while (walk.nextNode()) nodes.push(walk.currentNode);
for (const n of nodes) {
  const parts = n.textContent.split(/([\p{L}\p{N}]+)/u);
  const out = []; let changed = false;
  for (let i = 0; i < parts.length; i++) {
    if (i % 2 === 0) { out.push(parts[i]); continue; }
    const hit = terms.find(t => t.ws.every((w, k) => { const j = i + 2 * k; return j < parts.length && wordEq(parts[j].toLowerCase(), w, d.deep) && (k === 0 || /^[\s\-_]+$/.test(parts[j - 1])); }));
    if (!hit) { out.push(parts[i]); continue; }
    const span = document.createElement("span"); span.className = "dterm"; span.textContent = parts.slice(i, i + 2 * hit.ws.length - 1).join("");
    span.onmouseenter = e => dictTip(e, hit); span.onmouseleave = () => setTimeout(() => { if (!document.querySelector(".dag-tooltip:hover")) document.querySelectorAll(".dag-tooltip").forEach(x => x.remove()); }, 400); out.push(span); i += 2 * hit.ws.length - 2; changed = true; }
  if (changed) n.replaceWith(...out); } }
// dictTip: подсказка термина — определение и словарь (переход)
function dictTip(e, t) {
document.querySelectorAll(".dag-tooltip").forEach(x => x.remove());
const tip = el("dag-tooltip"); tip.append(el("dag-tooltip-title", t.term), el("dtdef", t.def));
const a = document.createElement("a"); a.href = "#"; a.className = "dtdict"; a.textContent = "↗ " + t.dict; a.onclick = ev => { ev.preventDefault(); nxGo(t.dict); }; tip.append(a);
tip.style.left = Math.min(e.clientX + 12, innerWidth - 300) + "px"; tip.style.top = (e.clientY + 14) + "px"; tip.style.maxWidth = "280px";
tip.onmouseleave = () => tip.remove(); document.body.append(tip); }
// tabsOf: дочерние концепта (обратный Taxon) — сначала по I-sort, остальные в алфавите темы
async function tabsOf(c) {
const ch = await api("/api/children?name=" + encodeURIComponent(c.name)).catch(() => []);
const order = cardSet(c, "I-sort").split(/\s+/).filter(Boolean).map(s => s.toLowerCase());
const pos = n => { const i = order.indexOf(n.toLowerCase()); return i < 0 ? order.length : i; };
return ch.sort((a, b) => pos(a.name) - pos(b.name)); }
// tabsRender: вкладки — hrznt сверху, vrtcl сбоку; во вкладке контент дочернего и его вкладки
function tabsRender(box, items, mode) {
const wrap = el(mode === "hrznt" ? "tabh" : "tabv"), bar = el("tabbar"), body = el("tabbody");
const show = async (x, b) => { bar.querySelectorAll("button").forEach(y => y.classList.toggle("on", y === b)); body.innerHTML = "";
  const h = await cardRender(x.name, "", []).catch(() => null), cd = el("cbody"); cd.innerHTML = h || ""; cardLinks(cd); body.append(cd);
  const xc = await api("/api/card?name=" + encodeURIComponent(x.name)).catch(() => ({})), xm = cardSet(xc, "Tabs");
  if (xm) { const sub = await tabsOf(xc); if (sub.length) { const nb = el("tabnest"); body.append(nb); tabsRender(nb, sub, xm); } }
  if (!h && !xm) body.append(el("cempty", "—")); };
items.forEach((x, i) => { const b = document.createElement("button"); b.textContent = x.name; b.ondblclick = () => nxGo(x.name); b.title = "двойной щелчок — карточка"; b.onclick = () => show(x, b); bar.append(b); if (!i) show(x, b); });
wrap.append(bar, body); box.append(wrap); }
// lensTabs: ↕ вкладки (как ltabs.js ПИ)
async function lensTabs(box, c) {
const items = await tabsOf(c);
if (!items.length) { box.append(el("cempty", "нет дочерних")); return; }
tabsRender(box, items, cardSet(c, "Tabs")); }
// lensProcess: ⊛ процесс (как prces.js ПИ) — ситуации: объект, препозиция, объект; ▶ — синонимы
// последняя строка — новая ситуация со следующим номером в теме
async function lensProcess(box, c) {
const rows = await api("/api/process?name=" + encodeURIComponent(c.name)).catch(() => []);
if (!rows.length) box.append(el("cempty", "нет ситуаций"));
const t = document.createElement("table"); t.className = "ptab";
t.innerHTML = "<tr><th>№</th><th>объект</th><th>препозиция</th><th>объект</th></tr>";
const cell = x => { const td = document.createElement("td");
  if (x.exists) { const a = document.createElement("a"); a.href = "#"; a.textContent = x.name; a.onclick = e => { e.preventDefault(); nxGo(x.name); }; td.append(a); } else td.append(x.name);
  if (x.alias && x.alias.length) { const tr = document.createElement("span"); tr.className = "ptrig"; tr.textContent = "▶"; tr.onclick = e => nxTip(e, { name: x.name, neighbors: x.alias.map(n => ({ name: n })) }, x.name); td.append(" ", tr); }
  return td; };
rows.forEach(r => { const tr = document.createElement("tr"), n = document.createElement("td"); n.className = "pnum"; n.textContent = r.key; tr.append(n, cell(r.a), cell(r.p), cell(r.b)); t.append(tr); });
const add = document.createElement("tr"), ins = [0, 1, 2].map(i => { const td = document.createElement("td"), v = document.createElement("input"); v.className = "cval"; v.placeholder = ["объект", "препозиция", "объект"][i]; v.spellcheck = false; td.append(v); add.append(td); return v; });
const okd = document.createElement("td"), ok = document.createElement("button"); ok.textContent = "✓"; ok.title = "добавить ситуацию"; okd.append(ok); add.prepend(okd);
ok.onclick = async () => { const s = ins.map(v => v.value.trim()); if (s.some(x => !x)) return; try { await api("/api/situation", { name: c.name, text: s.join(" ") }); } catch (e) { note.textContent = e.message; return; } cardOpen(c.name); };
ins.forEach(v => v.onkeydown = e => { if (e.key === "Enter") ok.click(); }); t.append(add);
const w = el("pwrap"); w.append(t); box.append(w); }
// lensRank: ◈ ранг (как lrank.js ПИ) — дерево дочерних на глубину Rank; ▶ раскрыть, ⬡ таксон, ▪ концепт
async function lensRank(box, c) {
const max = cardRank(c), tree = el("rtree");
const node = (x, depth) => { const n = el("rnode"), row = el("rrow"), ar = el("rarrow", depth < max ? "▶" : " ");
  const a = document.createElement("a"); a.href = "#"; a.className = "rlink"; a.textContent = x.name; a.onclick = e => { e.preventDefault(); nxGo(x.name); };
  row.append(ar, el("tpmark", x.taxon ? "⬡" : "▪"), a); n.append(row);
  if (depth < max) { const kids = el("rkids"); kids.hidden = true; n.append(kids); ar.classList.add("on");
    ar.onclick = async () => { if (kids.hidden && !kids.dataset.loaded) { kids.dataset.loaded = "1"; const ch = await api("/api/children?name=" + encodeURIComponent(x.name)).catch(() => []);
        if (!ch.length) kids.append(el("cempty", "—")); ch.forEach(y => kids.append(node(y, depth + 1))); }
      kids.hidden = !kids.hidden; ar.textContent = kids.hidden ? "▶" : "▼"; }; }
  return n; };
const ch = await api("/api/children?name=" + encodeURIComponent(c.name)).catch(() => []);
if (!ch.length) tree.append(el("cempty", "нет дочерних")); ch.forEach(x => tree.append(node(x, 1)));
box.append(tree); }
// lensTaxon: ⬡ таксоны (как ltags.js ПИ) — Taxon концепта; таксон — концепт с Rank
// плашка: картинка, имя, ▾ дочерние, × убрать; поле — подбор, «сделать таксоном» (Rank : 0), «создать таксон»
// каждая правка — предикация Концепт.Taxon : [ … ] в протокол
async function lensTaxon(box, c) {
const all = await api("/api/taxa").catch(() => []), mine = [];
for (const l of c.lines || []) { const m = l.match(/^Taxon : \[([^\]]*)\]/); if (m) mine.push(...m[1].trim().split(/\s+/).filter(Boolean)); }
const save = ns => runPred(c.name + ".Taxon : [ " + ns.join(" ") + " ]");
const pills = el("tpills");
mine.forEach(n => { const t = all.find(x => x.name.toLowerCase() === n.toLowerCase()), p = el("tpill"), ic = el("cicon tpic");
  if (t && t.color) p.style.borderColor = t.color;
  if (t) nxIcon(ic, t.icon);
  const a = document.createElement("a"); a.href = "#"; a.textContent = t ? t.name : n; a.onclick = e => { e.preventDefault(); nxGo(t ? t.name : n); };
  const ex = el("tpx", "▾"); ex.title = "members"; const rm = el("tpx", "×"); rm.title = "remove";
  rm.onclick = () => save(mine.filter(x => x !== n));
  ex.onclick = async () => { const old = p.querySelector(".tplist"); if (old) { old.remove(); return; }
    const ch = await api("/api/children?name=" + encodeURIComponent(t ? t.name : n)).catch(() => []), ls = el("tplist");
    if (!ch.length) ls.append(el("cempty", "—"));
    ch.forEach(x => { const i = document.createElement("a"); i.href = "#"; i.append(el("tpmark", x.taxon ? "⬡" : "▪"), x.name); i.title = x.taxon ? "taxon" : "concept"; i.onclick = e => { e.preventDefault(); nxGo(x.name); }; ls.append(i); });
    p.append(ls); };
  p.append(ic, a, ex, rm); pills.append(p); });
const inpT = document.createElement("input"); inpT.className = "cval"; inpT.placeholder = "add taxon..."; inpT.spellcheck = false;
const drop = el("tdrop");
const add = n => save(mine.concat([n]));
inpT.oninput = async () => { drop.innerHTML = ""; const v = inpT.value.trim(), lv = v.toLowerCase(); if (!v) return;
  const used = new Set(mine.map(x => x.toLowerCase()));
  const ms = all.filter(t => t.name.toLowerCase().includes(lv) && !used.has(t.name.toLowerCase())).slice(0, 26);
  ms.forEach(t => { const i = el("titem", t.name); i.onmousedown = e => { e.preventDefault(); add(t.name); }; drop.append(i); });
  if (ms.some(t => t.name.toLowerCase() === lv)) return;
  const ex = await api("/api/card?name=" + encodeURIComponent(v)).catch(() => ({}));
  if (inpT.value.trim() !== v) return;
  if (ex.found) { const i = el("titem tnew", "сделать таксоном «" + ex.name + "»"); i.onmousedown = async e => { e.preventDefault(); if (await runPred(ex.name + ".Rank : 0")) add(ex.name); }; drop.append(i); }
  else if (/^\p{Lu}[^\s.:\[\]{}()"]*$/u.test(v)) { const i = el("titem tnew", "создать таксон «" + v + "»"); i.onmousedown = async e => { e.preventDefault(); if (await runPred(v + " : []") && await runPred(v + ".Rank : 0")) add(v); }; drop.append(i); } };
inpT.onblur = () => setTimeout(() => drop.innerHTML = "", 200);
const wrap = el("twrap"); wrap.append(inpT, drop);
box.append(pills, wrap); }
// lensSlots: ⊞ строки концепта — ✓ отправляет Концепт.Параметр : значение в строку ввода
// у связи правится только прямой список, обратный ведёт script; первая строка — имя (rename), последняя — новый параметр
function lensSlots(box, c) {
const send = (k, v) => { if (!k.trim() || !v.trim()) return; inp.value = c.name + "." + k.trim() + " : " + v.trim(); document.getElementById("f").requestSubmit(); };
const row = (key, val, inv) => { const r = el("crow");
  const k = document.createElement(key === null ? "input" : "span"); k.className = "ckey"; if (key === null) k.placeholder = "Param"; else k.textContent = key;
  const v = document.createElement("input"); v.className = "cval"; v.value = val; v.spellcheck = false;
  const ok = document.createElement("button"); ok.textContent = "✓"; ok.title = "record"; ok.onclick = () => send(key === null ? k.value : key, v.value);
  v.onkeydown = e => { if (e.key === "Enter") ok.click(); };
  r.append(k, v, ok); if (inv && inv !== "[]") r.append(el("cinv", "← " + inv)); box.append(r); };
{ const r = el("crow"), k = el("ckey", "имя"), v = document.createElement("input"); v.className = "cval"; v.value = c.name; v.spellcheck = false;
  const ok = document.createElement("button"); ok.textContent = "✓"; ok.title = "rename"; ok.onclick = () => { const n = v.value.trim(); if (n && n !== c.name) runPred("rename [ " + c.name + " " + n + " ]"); };
  v.onkeydown = e => { if (e.key === "Enter") ok.click(); }; r.append(k, v, ok); box.append(r); }
for (const l of c.lines || []) {
  const m = l.match(/^(\p{Lu}[^\s.]*) : (.*)$/u);
  if (!m) { box.append(el("cbodyline", l)); continue; }
  if (linkKeys.includes(m[1]) && m[2].startsWith("[")) { const j = m[2].indexOf("]"); row(m[1], m[2].slice(0, j + 1), m[2].slice(j + 1).trim()); }
  else row(m[1], m[2]); }
row(null, ""); }
