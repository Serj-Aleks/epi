// script/infra/panel.js — лента: состояние сервера, предикации и ответы, строка ввода, кнопки шапки
// el: div с классом и текстом
function el(cls, text) { const d = document.createElement("div"); d.className = cls; if (text !== undefined) d.textContent = text; return d; }
// scroll: лента к последней записи
function scroll() { log.scrollTop = log.scrollHeight; }
// addPred: предикация в ленту (replay — лента заново, прокрутка одна в конце)
function addPred(text, replay) { const t = el("turn"); t.append(el("pred", text)); log.append(t); if (!replay) scroll(); return t; }
// addAnswer: ответ в ленту; после nexus, clone, rename, union, qwest, qwizt и правки — панель (replay — лента заново, без панели)
function addAnswer(t, turn, replay) {
const a = el("ans" + (turn.error ? " err" : ""));
a.append(el("v", turn.answer));
const f = el("f", (turn.frames || []).join(" ")); f.title = "interpres"; a.append(f);
t.append(a); if (!replay) scroll();
if (replay) return;
if (!turn.error && /^nexus \S+$/.test(turn.pred || "")) cardShow(turn.pred.slice(6));
else if (!turn.error && /^(clone|rename|union)\s/.test(turn.pred || "")) cardShow(turn.answer);
else if (!turn.error && /^qwest\s/.test(turn.pred || "")) qwestShow(turn.pred.slice(6).trim().replace(/^"(.*)"$/, "$1"));
else if (!turn.error && /^qwizt\s/.test(turn.pred || "")) qwestShow(turn.pred.slice(6).trim().replace(/^"(.*)"$/, "$1"), false, true);
else if (!turn.error && cardName) cardShow(cardName); }
// api: запрос к серверу, с body — POST
async function api(path, body) {
const r = await fetch(path, body === undefined ? {} : { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });
if (!r.ok) throw new Error(await r.text());
return r.json(); }
// show: состояние сервера — темы, сеансы, лента текущего сеанса; другая тема или сеанс закрывает панель, без панели — карточка последнего nexus
function show(st) {
if (!last || last.theme !== st.theme || last.current !== st.current) cardHide();
last = st; mode = null; inp.placeholder = "";
note.textContent = "";
tsel.innerHTML = "";
if (!st.theme) { const o = document.createElement("option"); o.textContent = "—"; o.value = ""; tsel.append(o); }
for (const t of st.themes || []) { const o = document.createElement("option"); o.value = o.textContent = t; o.selected = t === st.theme; tsel.append(o); }
document.getElementById("tren").disabled = !st.theme;
{ const ti = document.getElementById("ticon"); ti.innerHTML = ""; ti.disabled = !st.theme; if (st.theme) nxIcon(ti, "/api/themeicon?name=" + encodeURIComponent(st.theme)); }
document.getElementById("cnew").disabled = !st.theme || !st.current || st.closed;
document.getElementById("new").disabled = !st.theme;
if (!st.theme) note.textContent = "создайте тему ＋";
sel.innerHTML = "";
if (!st.current) { const o = document.createElement("option"); o.textContent = "—"; o.value = ""; sel.append(o); }
for (const s of st.seanses || []) { const o = document.createElement("option"); o.value = s.name; o.textContent = s.name.replace(/^seans-/, "").replace(/\.uic$/, "") + (s.closed ? " ■" : ""); o.selected = s.name === st.current; sel.append(o); }
writable = !!st.current && !st.closed;
document.getElementById("close").disabled = !writable;
log.innerHTML = ""; hist = [];
for (const turn of st.turns || []) { const t = addPred(turn.pred, true); addAnswer(t, turn, true); hist.push(turn.pred); }
if (cardBox.hidden) { const n = (st.turns || []).filter(t => !t.error && /^nexus \S+$/.test(t.pred)).pop(); if (n) cardShow(n.pred.slice(6)); }
scroll(); hpos = hist.length;
inp.disabled = !writable;
if (st.pending && st.pending.question) { inp.disabled = true; respond(addPred(st.pending.pred), { question: st.pending.question, options: st.pending.options }); }
else if (writable) inp.focus(); }
// act: действие на сервере и новое состояние
async function act(path, body) { try { show(await api(path, body)); } catch (e) { note.textContent = e.message; } }
// respond: ответ сервера — ответы на предикации строки, вопрос trans с выбором словаря или вопрос ✓ ✕
// при выборе темы или atrbt имя концепта можно поправить (форма слова — синоним)
function respond(t, rep) {
if (rep.turns) {
  rep.turns.forEach((turn, i) => { const tt = i === 0 ? t : addPred(turn.pred); if (i === 0) tt.querySelector(".pred").textContent = turn.pred; addAnswer(tt, turn); });
  if (!rep.turns.length) t.remove();
  inp.disabled = !writable; inp.focus(); return; }
if (rep.options) {
  const q = el("q", rep.question); t.append(q); scroll();
  const word = (rep.question.match(/«([^»]+)»/) || [])[1] || "";
  const choose = async (c, label) => {
    q.querySelectorAll("button").forEach(b => b.remove()); q.append(el("f", label));
    mode = null; inp.placeholder = ""; inp.value = ""; inp.disabled = true;
    respond(t, await api("/api/answer", { choice: c })); };
  rep.options.forEach(o => {
    const b = document.createElement("button"); b.textContent = o; q.append(b);
    b.onclick = () => {
      if (o !== last.theme && o !== "atrbt") { choose(o, o); return; }
      const pre = o === "atrbt" ? "atrbt:" : "thema:";
      mode = "concept"; inp.disabled = false; inp.value = word.charAt(0).toUpperCase() + word.slice(1);
      inp.placeholder = (o === "atrbt" ? "имя атрибута" : "имя концепта темы " + o) + " (Enter — принять, Esc — пропустить)"; inp.focus();
      conceptDone = (name) => choose(name ? pre + name : "✕", name ? o + ": " + name : "✕"); }; });
  return; }
const q = el("q", rep.question);
const yes = document.createElement("button"); yes.textContent = "✓"; yes.title = "yes (y)";
const no = document.createElement("button"); no.textContent = "✕"; no.title = "no (n)";
q.append(yes, no); t.append(q); scroll(); yes.focus();
const answer = async (v) => { document.removeEventListener("keydown", key); yes.remove(); no.remove(); q.append(el("f", v ? "✓" : "✕")); respond(t, await api("/api/answer", { answer: v })); };
const key = (e) => { if (e.key === "y") answer(true); if (e.key === "n") answer(false); };
yes.onclick = () => answer(true); no.onclick = () => answer(false);
document.addEventListener("keydown", key); }
// runPred: предикация в протокол, как ввод в строке; true — выполнена, без сеанса — подсказка
async function runPred(text) {
if (!writable) { note.textContent = "сеанс закрыт — откройте сеанс ＋"; inp.value = text; return false; }
hist.push(text); hpos = hist.length; inp.disabled = true;
const t = addPred(text);
try { respond(t, await api("/api/eval", { text })); return true; }
catch (err) { addAnswer(t, { answer: String(err.message), error: true }); inp.disabled = !writable; inp.focus(); return false; } }
// askName: строка ввода ждёт имя — темы, клона или нового концепта
function askName(m) {
mode = m; inp.disabled = false;
inp.value = m === "rename" ? last.theme : "";
inp.placeholder = m === "rename" ? "новое имя темы (Esc — отмена)" : m === "clone" ? "имя клона " + cloneSrc + " (Esc — отмена)" : m === "cnew" ? "имя нового концепта (Esc — отмена)" : "имя новой темы (Esc — отмена)";
inp.focus(); }
// строка ввода, Enter: ответ trans, clone [ Исходный Новый ], новый концепт (Имя : [], затем nexus), имя темы или предикация
document.getElementById("f").onsubmit = async (e) => {
e.preventDefault();
if (mode === "concept") { conceptDone(inp.value.trim()); return; }
if (mode === "clone") {
  const name = inp.value.trim(); if (!name) return;
  if (!/^\p{Lu}[^\s.:\[\]{}()"]*$/u.test(name)) { note.textContent = "имя концепта — с прописной буквы, без пробелов"; return; }
  mode = null; inp.placeholder = ""; inp.value = ""; note.textContent = "";
  runPred("clone [ " + cloneSrc + " " + name + " ]"); return; }
if (mode === "cnew") {
  const name = inp.value.trim(); if (!name) return;
  if (!/^\p{Lu}[^\s.:\[\]{}()"]*$/u.test(name)) { note.textContent = "имя концепта — с прописной буквы, без пробелов"; return; }
  mode = null; inp.placeholder = ""; inp.value = ""; note.textContent = "";
  const c = await api("/api/card?name=" + encodeURIComponent(name)).catch(() => ({}));
  if (!c.found && !await runPred(name + " : []")) return;
  nxGo(name); return; }
if (mode) {
  const name = inp.value.trim(); if (!name) return;
  const m = mode; inp.value = "";
  if (m === "new") act("/api/thema/new", { name });
  else act("/api/thema/rename", { name: last.theme, to: name });
  return; }
const text = inp.value.trim(); if (!text) return;
inp.value = ""; runPred(text); };
// строка ввода, клавиши: Esc — отмена ввода имени, ↑ ↓ — история
inp.addEventListener("keydown", (e) => {
if (mode === "concept") { if (e.key === "Escape") conceptDone(""); return; }
if (mode) { if (e.key === "Escape") { mode = null; inp.value = ""; inp.placeholder = ""; note.textContent = ""; inp.disabled = !writable; } return; }
if (e.key === "ArrowUp" && hpos > 0) { inp.value = hist[--hpos]; e.preventDefault(); }
if (e.key === "ArrowDown") { hpos = Math.min(hist.length, hpos + 1); inp.value = hist[hpos] || ""; e.preventDefault(); } });
// кнопки шапки и выбор темы, сеанса
document.getElementById("tnew").onclick = () => askName("new");
document.getElementById("tren").onclick = () => askName("rename");
document.getElementById("ticon").onclick = () => themeShow();
document.getElementById("cnew").onclick = () => askName("cnew");
tsel.onchange = () => { if (tsel.value) act("/api/thema/open", { name: tsel.value }); };
document.getElementById("new").onclick = () => act("/api/new", {});
document.getElementById("close").onclick = () => act("/api/close", {});
sel.onchange = () => { if (sel.value) act("/api/open", { name: sel.value }); };
// ошибка кода панели — в шапку, чтобы её было видно без консоли
window.addEventListener("error", e => { note.textContent = "ошибка: " + e.message; });
window.addEventListener("unhandledrejection", e => { note.textContent = "ошибка: " + (e.reason && e.reason.message || e.reason); });
// старт: фон дня, состояние сервера
document.body.classList.add(days[new Date().getDay()]);
api("/api/state").then(show);
