// script/infra/cards.js — карточка концепта: одна панель рядом с лентой
// в контенте [[Имя]] — ссылка, {{Имя}} — трансклюзия
// pPush: вид панели в историю ← (просмотр, без протокола)
function pPush(s) { const t = pstack[pstack.length - 1]; if (!t || t.k !== s.k || t.a !== s.a) { pstack.push(s); if (pstack.length > 50) pstack.shift(); } }
// pBack: прежний вид панели — карточка, карточка темы или список qwest
function pBack() { if (pstack.length < 2) return; pstack.pop(); const s = pstack[pstack.length - 1]; s.k === "c" ? cardShow(s.a, true) : s.k === "t" ? themeShow(true) : qwestShow(s.a, true, s.all); }
// pBackBtn: кнопка ←
function pBackBtn() { const b = document.createElement("button"); b.textContent = "←"; b.title = "back"; b.disabled = pstack.length < 2; b.onclick = pBack; return b; }
// cardShow: открыть карточку концепта
function cardShow(name, back) { if (!back) pPush({ k: "c", a: name }); cardName = name; qwestKey = null; clearTimeout(cardTimer); cardTimer = setTimeout(() => cardOpen(cardName), 0); }
// cardHide: закрыть панель
function cardHide() { cardName = null; qwestKey = null; pstack = []; card.hidden = true; card.innerHTML = ""; cardCss(""); favicon(""); }
// favicon: картинка вкладки браузера — концепта открытой карточки
function favicon(url) { let l = document.querySelector("link[rel=icon]"); if (!l) { l = document.createElement("link"); l.rel = "icon"; document.head.append(l); } l.href = url || "data:,"; }
// cardCss: стиль концепта (спутник css) — body, html, :root относятся к карточке, остальное внутри неё
function cardCss(css) {
let st = document.getElementById("ccss");
if (!st) { st = document.createElement("style"); st.id = "ccss"; document.head.append(st); }
st.textContent = (css || "").replace(/\/\*[\s\S]*?\*\//g, "").split("}").map(r => { const i = r.indexOf("{"); if (i < 0) return r;
  const sel = r.slice(0, i).trim(); if (!sel || sel.startsWith("@")) return r;
  return sel.split(",").map(s => s.trim().replace(/^(html|body|:root)\b/, "#card")).map(s => s.startsWith("#card") ? s : "#card " + s).join(", ") + " " + r.slice(i); }).join("}"); }
// esc: текст для html
function esc(s) { return String(s || "").replace(/&/g, "&amp;").replace(/"/g, "&quot;").replace(/</g, "&lt;"); }
// mdHtml: md в html — заголовки, списки, таблицы, код, ссылки; своя картинка — из img концепта
function mdHtml(md, name) {
const code = [];
md = md.replace(/```([\s\S]+?)```/g, (_, c) => { code.push("<pre><code>" + esc(c.replace(/^\n/, "").replace(/\n$/, "")) + "</code></pre>"); return `\x00C${code.length - 1}\x00`; });
const inl = s => s.replace(/\[\[([^\]]+)\]\]/g, (_, n) => `<a href="#" class="cref" data-n="${esc(n.trim())}">${esc(n.trim())}</a>`)
  .replace(/!\[([^\]]*)\]\(([^)]+)\)/g, (_, alt, src) => `<img src="${/^(https?:\/)?\//.test(src) ? src : "/api/img?name=" + encodeURIComponent(name) + "&f=" + encodeURIComponent(src)}" alt="${esc(alt)}" style="max-width:100%">`)
  .replace(/\[([^\]]+)\]\((https?:\/\/[^)]+)\)/g, '<a href="$2" target="_blank" rel="noopener">$1</a>')
  .replace(/\*\*(.+?)\*\*/g, "<strong>$1</strong>").replace(/\*(.+?)\*/g, "<em>$1</em>").replace(/`(.+?)`/g, "<code>$1</code>");
let h = "", list = "", tab = false, head = false;
const endList = () => { if (list) { h += `</${list}>`; list = ""; } }, endTab = () => { if (tab) { h += "</table>"; tab = false; } };
for (const line of md.split("\n")) {
  if (/^\|.+\|$/.test(line)) { endList();
    if (!tab) { h += "<table>"; tab = true; head = true; }
    if (/^\|[-| :]+\|$/.test(line)) continue;
    const cells = line.split("|").slice(1, -1).map(c => inl(c.trim()));
    h += "<tr>" + cells.map(c => head ? `<th>${c}</th>` : `<td>${c}</td>`).join("") + "</tr>"; head = false; continue; }
  endTab();
  let m;
  if ((m = line.match(/^(#{1,3}) (.+)$/))) { endList(); h += `<h${m[1].length}>${inl(m[2])}</h${m[1].length}>`; }
  else if ((m = line.match(/^[-*] (.+)$/)) || (m = line.match(/^\d+\. (.+)$/))) { const k = /^\d/.test(line) ? "ol" : "ul"; if (list !== k) { endList(); h += `<${k}>`; list = k; } h += "<li>" + inl(m[1]) + "</li>"; }
  else if (/^---+$/.test(line.trim())) { endList(); h += "<hr>"; }
  else if (line.trim() === "") endList();
  else { endList(); h += "<p>" + inl(line) + "</p>"; } }
endList(); endTab();
return h.replace(/\x00C(\d+)\x00/g, (_, i) => code[i]); }
// cardRender: контент концепта с трансклюзиями, без повторов по кругу
async function cardRender(name, ext, seen) {
const c = await api("/api/card?name=" + encodeURIComponent(name));
if (!c.found) return null;
ext = ext || c.content;
const raw = (await api("/api/sat?name=" + encodeURIComponent(name) + "&ext=" + ext)).text || "";
if (!raw.trim()) return "";
const inc = [], vis = new Set(seen).add(c.name.toLowerCase());
const marked = raw.split(/(```[\s\S]*?```)/).map((p, i) => i % 2 ? p : p.replace(/\{\{([^{}]+)\}\}/g, (_, n) => { inc.push(n.trim()); return `\x00I${inc.length - 1}\x00`; })).join("");
const html = ext === "md" ? mdHtml(marked, c.name) : ext === "html" || ext === "svg" ? marked : "<pre>" + esc(marked) + "</pre>";
const parts = await Promise.all(inc.map(n => vis.has(n.toLowerCase()) ? null : cardRender(n, "", vis).catch(() => null)));
return html.replace(/\x00I(\d+)\x00/g, (_, i) => parts[i] == null ? esc("{{" + inc[i] + "}}") : `<div class="cinc" title="${esc(inc[i])}">${parts[i]}</div>`); }
// cardLinks: ссылки [[Имя]] ведут к концепту
function cardLinks(box) { box.querySelectorAll("a.cref").forEach(a => a.onclick = e => { e.preventDefault(); nxGo(a.dataset.n); }); }
// cardSet: значение параметра концепта без скобок
function cardSet(c, k) { for (const l of c.lines || []) { if (l.startsWith(k + " : ")) return l.slice(k.length + 3).replace(/^\[\s*|\s*\]$/g, "").trim(); } return ""; }
// cardRank: Rank концепта, без него 0
function cardRank(c) { for (const l of c.lines || []) { const m = l.match(/^Rank : (\d+)/); if (m) return +m[1]; } return 0; }
// cardOpen: собрать карточку отдельно и поставить целиком (правки подряд не дублируют линзы); имя сверяется без учёта регистра (NewSeans — карточка Newseans)
// та же карточка после правки — на месте, новая — с начала; цвет концепта — в шапке, картинка — во вкладке
// кнопки линз ◈, ↕, ⊛ — только при Rank > 0, Tabs, Process
async function cardOpen(name) {
const gen = ++cardGen, root = document.createElement("div");
let c;
try { c = await api("/api/card?name=" + encodeURIComponent(name)); } catch (e) { note.textContent = "ошибка: " + e.message; return; }
if (!c.found || !cardName || name.toLowerCase() !== cardName.toLowerCase()) return;
cardName = c.name;
const keep = card.dataset.name === c.name ? card.scrollTop : 0; card.dataset.name = c.name;
const head = el("chead"), ic = el("cicon"); nxIcon(ic, c.icon);
head.append(pBackBtn(), ic, el("cname", c.name));
if (c.color) { head.style.borderBottom = "3px solid " + c.color; head.querySelector(".cname").style.color = c.color; }
favicon(c.icon);
const tools = el("ctools");
[["graph", "✦"], ["taxon", "⬡"], ["content", "≡"], ["process", "⊛"], ["rank", "◈"], ["tabs", "↕"], ["slots", "⊞"]].filter(([k]) => (k !== "rank" || cardRank(c) > 0) && (k !== "tabs" || cardSet(c, "Tabs")) && (k !== "process" || cardSet(c, "Process"))).forEach(([k, g]) => { const b = document.createElement("button"); b.textContent = g; b.title = k; b.className = lens[k] ? "on" : ""; b.onclick = () => { lens[k] = !lens[k]; cardName = c.name; cardOpen(c.name); }; tools.append(b); });
const cl = document.createElement("button"); cl.textContent = "⧉"; cl.title = "clone"; cl.onclick = () => { cloneSrc = c.name; askName("clone"); }; tools.prepend(cl);
const x = document.createElement("button"); x.textContent = "✕"; x.title = "close"; x.onclick = cardHide; tools.append(x);
const fi = document.createElement("input"); fi.className = "cval cfind"; fi.placeholder = "find"; fi.spellcheck = false; fi.title = "поиск внутри концепта и его синонимов (md, html, json, xml, txt)";
const mine = cardFind && cardFind.name.toLowerCase() === c.name.toLowerCase(); if (mine) fi.value = cardFind.q;
const fbox = el("clens flist"); fbox.hidden = true;
fi.onkeydown = e => { if (e.key === "Enter") findRun(fbox, c, fi.value.trim()); if (e.key === "Escape") { fi.value = ""; cardFind = null; cardOpen(c.name); } };
head.append(fi, tools); root.append(head, fbox);
if (mine && cardFind.q) findRun(fbox, c, cardFind.q);
const css = c.sats.includes("css") ? (await api("/api/sat?name=" + encodeURIComponent(c.name) + "&ext=css").catch(() => ({}))).text : "";
if (lens.graph) { const d = await api("/api/nexus?name=" + encodeURIComponent(name)).catch(() => null);
  if (d && d.found) { const sides = ["nw", "ne", "wt", "et", "sw", "se", "nr", "st"];
    const hide = new Set((d.graph || []).map(s => s.toLowerCase()));
    const sd = Object.assign({}, d); sides.forEach(s => { if (hide.has(s)) sd[s] = []; });
    const box = el("clens"); nxSvg(box, sd); box.append(Object.assign(document.createElement("hr"), { className: "dag-divider" })); nxList(box, d); root.append(box); } }
if (lens.taxon) { const box = el("clens tlens"); root.append(box); await lensTaxon(box, c); }
if (lens.content) { const box = el("clens"); root.append(box); await lensContent(box, c); }
if (lens.process && cardSet(c, "Process")) { const box = el("clens"); root.append(box); await lensProcess(box, c); }
if (lens.rank && cardRank(c) > 0) { const box = el("clens"); root.append(box); await lensRank(box, c); }
if (lens.tabs && cardSet(c, "Tabs")) { const box = el("clens"); root.append(box); await lensTabs(box, c); }
if (lens.slots) { const box = el("clens"); lensSlots(box, c); root.append(box); }
if (gen !== cardGen) return;
cardCss(css); card.hidden = false; card.replaceChildren(...root.childNodes); card.scrollTop = keep; }
