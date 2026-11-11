// script/infra/qwest.js — поиск qwest, qwizt, карточка темы, поиск внутри концепта
// qwestShow: список найденного в панели; all — qwizt, во всех темах
function qwestShow(qs, back, all) { if (!back) pPush({ k: "q", a: qs, f: "", all }); cardName = null; qwestKey = (all ? "\u0001" : "") + qs; clearTimeout(cardTimer); cardTimer = setTimeout(() => qwestOpen(qs, all), 0); }
// mark: подсветить слова в тексте
function mark(s, ts) { let r = esc(s); ts.forEach(t => { r = r.replace(new RegExp(esc(t).replace(/[.*+?^${}()|[\]\\]/g, "\\$&"), "gi"), m => `<mark>${m}</mark>`); }); return r; }
// qwestOpen: список qwest с фильтром по имени (& — и, как в ПИ), ⬡ — таксон, ▪ — концепт; концепт другой темы переключает тему
async function qwestOpen(qs, all) {
let rs;
try { rs = await api((all ? "/api/qwizt?q=" : "/api/qwest?q=") + encodeURIComponent(qs)); } catch (e) { return; }
if ((all ? "\u0001" : "") + qs !== qwestKey) return;
rs = rs || []; card.hidden = false; card.innerHTML = ""; card.scrollTop = 0; card.dataset.name = ""; cardCss("");
const ts = qs.toLowerCase().split("&").map(t => t.trim()).filter(Boolean);
const head = el("chead"), st = pstack[pstack.length - 1] || {}; head.append(pBackBtn(), el("cname", (all ? "qwizt " : "qwest ") + (qs ? "«" + qs + "»" : "—") + " · " + rs.length));
const flt = document.createElement("input"); flt.className = "cval"; flt.placeholder = "filter"; flt.spellcheck = false;
const tools = el("ctools"), x = document.createElement("button"); x.textContent = "✕"; x.title = "close"; x.onclick = cardHide; tools.append(x);
head.append(flt, tools); card.append(head);
const list = el("clens qlist"); card.append(list);
const draw = () => { list.innerHTML = ""; const fs = flt.value.toLowerCase().split("&").map(t => t.trim()).filter(Boolean);
  const shown = rs.filter(r => fs.every(f => r.name.toLowerCase().includes(f)));
  if (!shown.length) { list.append(el("cempty", "—")); return; }
  shown.forEach(r => { const row = el("qrow"), ic = el("cicon"); nxIcon(ic, r.icon);
    const a = document.createElement("a"); a.href = "#"; a.className = "qname"; a.innerHTML = mark(r.name, ts.concat(fs)); a.onclick = async e => { e.preventDefault(); if (all && r.theme !== last.theme) await act("/api/thema/open", { name: r.theme }); nxGo(r.name); };
    const top = el("qtop"), mk = el("tpmark", r.taxon ? "⬡" : "▪"); top.append(mk, ic, a); if (r.type || all) top.append(el("qtype", (all ? r.theme + (r.type ? " · " : "") : "") + (r.type || ""))); row.append(top);
    if (r.excerpt) { const ex = el("qex"); ex.innerHTML = mark(r.excerpt, ts); row.append(ex); }
    list.append(row); }); };
flt.value = st.f || ""; flt.oninput = () => { st.f = flt.value; draw(); }; draw(); }
// themeShow: карточка темы (вместо портала ПИ) — картинка темы в шапке
function themeShow(back) { if (!back) pPush({ k: "t", a: "", f: "" }); cardName = null; qwestKey = "\u0000t"; clearTimeout(cardTimer); cardTimer = setTimeout(themeOpen, 0); }
// themeOpen: сверху все темы (щелчок — переключить и открыть её карточку), ниже концепты текущей темы с фильтром (⬡ — таксон, ▪ — концепт)
async function themeOpen() {
const ts = await api("/api/themes").catch(() => []), rs = last && last.theme ? await api("/api/qwest?q=").catch(() => []) : [];
if (qwestKey !== "\u0000t") return;
card.hidden = false; card.innerHTML = ""; card.scrollTop = 0; card.dataset.name = ""; cardCss("");
const st = pstack[pstack.length - 1] || {}, head = el("chead"), ic = el("cicon"), cur = ts.find(t => t.name === (last && last.theme));
if (cur) nxIcon(ic, cur.icon);
head.append(pBackBtn(), ic, el("cname", (last && last.theme) || "—"));
const tools = el("ctools"), x = document.createElement("button"); x.textContent = "✕"; x.title = "close"; x.onclick = cardHide; tools.append(x); head.append(tools); card.append(head);
const tl = el("clens thlist");
ts.forEach(t => { const row = el("throw" + (t.name === (last && last.theme) ? " on" : "")), ti = el("cicon"); nxIcon(ti, t.icon);
  const a = document.createElement("a"); a.href = "#"; a.className = "qname"; a.textContent = t.name; a.onclick = async e => { e.preventDefault(); if (t.name !== last.theme) { await act("/api/thema/open", { name: t.name }); themeShow(); } };
  row.append(ti, a, el("thmeta", `концептов ${t.concepts} · таксонов ${t.taxa} · спутников ${t.sats} · сеансов ${t.seans}` + (t.last ? ` · последний ${t.last}` : ""))); tl.append(row); });
card.append(tl);
const fh = el("thfilter"), flt = document.createElement("input"); flt.className = "cval"; flt.placeholder = "filter"; flt.spellcheck = false;
const qa = document.createElement("button"); qa.textContent = "∀"; qa.title = "qwizt"; qa.onclick = () => { const v = flt.value.trim(); if (v) runPred("qwizt \"" + v.replace(/"/g, "") + "\""); };
fh.append(flt, qa, el("thmeta", String(rs.length))); card.append(fh);
const list = el("clens qlist"); card.append(list);
const draw = () => { list.innerHTML = ""; const fs = flt.value.toLowerCase().split("&").map(t => t.trim()).filter(Boolean);
  const shown = rs.filter(r => fs.every(f => r.name.toLowerCase().includes(f)));
  if (!shown.length) { list.append(el("cempty", "—")); return; }
  shown.forEach(r => { const row = el("qrow"), ic2 = el("cicon"); nxIcon(ic2, r.icon);
    const a = document.createElement("a"); a.href = "#"; a.className = "qname"; a.innerHTML = mark(r.name, fs); a.onclick = e => { e.preventDefault(); nxGo(r.name); };
    const top = el("qtop"), mk = el("tpmark", r.taxon ? "⬡" : "▪"); top.append(mk, ic2, a); row.append(top); list.append(row); }); };
flt.value = st.f || ""; flt.oninput = () => { st.f = flt.value; draw(); }; draw(); }
// findRun: поиск внутри концепта (как SearchInConcept ПИ) — концепт и синонимы, тип спутника, отрывок
// щелчок — концепт, его спутник в линзе контента, найденное подсвечено
async function findRun(box, c, q) {
box.innerHTML = ""; box.hidden = !q; if (!q) return;
const rs = await api("/api/find?name=" + encodeURIComponent(c.name) + "&q=" + encodeURIComponent(q)).catch(() => []), ts = q.toLowerCase().split("&").map(t => t.trim()).filter(Boolean);
if (!rs.length) { box.append(el("cempty", "не найдено")); return; }
rs.forEach(r => { const row = el("qrow"), a = document.createElement("a"); a.href = "#"; a.className = "qname"; a.innerHTML = mark(r.name, ts);
  a.onclick = e => { e.preventDefault(); cardFind = { name: r.name, ext: r.type, q }; lens.content = true; r.name.toLowerCase() === c.name.toLowerCase() ? cardOpen(c.name) : nxGo(r.name); };
  const top = el("qtop"); top.append(a, el("qtype", r.type || "имя")); row.append(top);
  if (r.excerpt) { const ex = el("qex"); ex.innerHTML = mark(r.excerpt, ts); row.append(ex); }
  box.append(row); }); }
// findMark: подсветить слова запроса в тексте линзы и прокрутить к первому
function findMark(root, q) {
const ts = q.toLowerCase().split("&").map(t => t.trim()).filter(Boolean); if (!ts.length) return;
const walk = document.createTreeWalker(root, NodeFilter.SHOW_TEXT), nodes = []; while (walk.nextNode()) nodes.push(walk.currentNode);
const re = new RegExp("(" + ts.map(t => t.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")).join("|") + ")", "gi");
nodes.forEach(n => { if (!re.test(n.textContent)) return; re.lastIndex = 0; const s = document.createElement("span"); s.innerHTML = esc(n.textContent).replace(re, "<mark>$1</mark>"); n.replaceWith(...s.childNodes); });
const m = root.querySelector("mark"); if (m) setTimeout(() => m.scrollIntoView({ block: "center" }), 50); }
