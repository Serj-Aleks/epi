// script/infra/nexus.js — граф локального контекста nexus
// концепт в центре, слева входящие (обратные списки), справа исходящие (прямые), под графом — список связей
// nxGo: переход к концепту — nexus Имя в протокол; без открытого сеанса — только карточка
function nxGo(name) { document.querySelectorAll(".dag-tooltip").forEach(e => e.remove()); if (!writable) { cardShow(name); return; } inp.value = "nexus " + name; document.getElementById("f").requestSubmit(); }
// nodeGo: щелчок по узлу — тема открывает свою карточку, концепт другой темы переключает тему
async function nodeGo(n) { if (n.type === "theme") { if (n.name !== last.theme) await act("/api/thema/open", { name: n.name }); themeShow(); return; }
if (n.theme && n.theme !== last.theme) await act("/api/thema/open", { name: n.theme }); nxGo(n.name); }
// nxIcon: картинка концепта в div — svg вставляется, остальное через img
function nxIcon(div, url) {
if (!url) return;
fetch(url).then(r => r.ok ? (/svg/.test(r.headers.get("Content-Type") || "") ? r.text().then(t => { div.innerHTML = t; const s = div.querySelector("svg"); if (s) s.setAttribute("style", "width:100%;height:100%;"); }) : (div.innerHTML = `<img src="${url}" style="width:100%;height:100%;object-fit:contain">`)) : 0).catch(() => {}); }
// nxColH: высота столбца из n узлов
function nxColH(n) { return n > 0 ? n * NXG.nodeH + (n - 1) * NXG.vSpacing : 0; }
// nxGroup: узлы столбца с координатой y
function nxGroup(list, y0) { return list.map((n, i) => ({ n, y: y0 + i * (NXG.nodeH + NXG.vSpacing) })); }
// nxColumn: y верхней, средней и нижней групп столбца вокруг оси
function nxColumn(upper, middle, lower, yAxis) {
const act = [["upper", nxColH(upper.length)], ["middle", nxColH(middle.length)], ["lower", nxColH(lower.length)]].filter(a => a[1] > 0);
const r = { upperY: 0, middleY: 0, lowerY: 0 };
if (act.length === 1) r[act[0][0] + "Y"] = yAxis - act[0][1] / 2;
if (act.length === 2) { r[act[0][0] + "Y"] = yAxis - NXG.groupGap / 2 - act[0][1]; r[act[1][0] + "Y"] = yAxis + NXG.groupGap / 2; }
if (act.length === 3) { const hM = nxColH(middle.length), midY = yAxis - hM / 2; r.upperY = midY - NXG.groupGap - nxColH(upper.length); r.middleY = midY; r.lowerY = midY + hM + NXG.groupGap; }
return r; }
// nxNeeds: место над осью и под ней для трёх групп
function nxNeeds(u, m, l) {
const L = nxColumn(new Array(u), new Array(m), new Array(l), 0);
const top = u ? L.upperY : m ? L.middleY : l ? L.lowerY : 0;
const bot = l ? L.lowerY + nxColH(l) : m ? L.middleY + nxColH(m) : u ? L.upperY + nxColH(u) : 0;
return { above: Math.max(0, -top), below: Math.max(0, bot) }; }
// nxVisible: соседи узла для подсказки; у темы — её концепты с совпадающими именами
function nxVisible(n, center) {
const nb = n.neighbors || [];
if (n.name === center || n.type === "theme") return nb;
return nb.some(x => x.name === center) ? nb.filter(x => x.name !== center) : []; }
// nxTip: подсказка узла — имя и соседи, щелчок — переход
function nxTip(evt, n, center) {
evt.stopPropagation(); document.querySelectorAll(".dag-tooltip").forEach(e => e.remove());
const tip = el("dag-tooltip"), h = document.createElement("a"); h.href = "#"; h.textContent = n.name; h.className = "dag-tooltip-title";
h.onclick = e => { e.preventDefault(); nodeGo(n); }; tip.append(h);
const clean = nxVisible(n, center);
if (clean.length) { const ul = document.createElement("ul"); ul.className = "dag-tooltip-list";
  clean.forEach(x => { const li = document.createElement("li"), a = document.createElement("a"); a.href = "#"; a.textContent = x.name; a.onclick = e => { e.preventDefault(); nodeGo(x); }; li.append(a); ul.append(li); });
  tip.append(ul); }
tip.style.left = Math.min(evt.clientX + 12, innerWidth - 230) + "px"; tip.style.top = evt.clientY + "px";
document.body.append(tip); }
// щелчок вне подсказки убирает её
document.addEventListener("click", () => document.querySelectorAll(".dag-tooltip").forEach(e => e.remove()));
// nxSvg: граф — nw, wt, sw слева, ne, et, se справа, nr сверху, st снизу; тёмный треугольник — в подсказке есть соседи
function nxSvg(box, data) {
const g = NXG, center = data.center.name;
const [nw, sw, wt, ne, se, et, nr, st] = ["nw", "sw", "wt", "ne", "se", "et", "nr", "st"].map(s => data[s] || []);
const ln = nxNeeds(nw.length, wt.length, sw.length), rn = nxNeeds(ne.length, et.length, se.length);
const nrH = nxColH(nr.length), stH = nxColH(st.length);
const topH = nr.length ? nrH + g.vSpacing * 2 : 0, botH = st.length ? stH + g.vSpacing * 2 : 0;
const above = Math.max(ln.above, rn.above, topH + g.nodeH / 2), below = Math.max(ln.below, rn.below, botH + g.nodeH / 2);
const yAxis = above + g.padding, H = above + below + g.padding * 2;
const hasL = !!(nw.length || sw.length || wt.length), hasR = !!(ne.length || se.length || et.length), hasIn = hasL || nr.length > 0;
let x = g.padding;
const xL = hasL ? x : null; if (hasL) x += g.nodeW + g.hSpacing;
const xLC = hasL ? x : null, xM = (xLC || g.padding) + (nr.length ? g.collectorGap : 0);
x = Math.max(x, xM) + g.hSpacing;
const xC = x; x += g.nodeW + g.hSpacing;
const xBC = x; if (hasR) x += g.collectorGap;
const xRC = hasR ? x : null; if (hasR) x += g.hSpacing;
const xR = hasR ? x : null; if (hasR) x += g.nodeW;
const W = (st.length && !hasR ? xBC : x) + g.padding;
const svg = document.createElementNS(SVGNS, "svg"); svg.setAttribute("width", W); svg.setAttribute("height", H); svg.setAttribute("class", "dag-svg");
svg.innerHTML = '<defs><marker id="dag-arrow" viewBox="0 0 10 10" refX="8" refY="5" markerWidth="5" markerHeight="5" orient="auto-start-reverse"><path d="M 0 0 L 10 5 L 0 10 z" fill="#a8a49e"/></marker></defs>';
box.append(svg);
const poly = (pts, arrow, cv) => { const e = document.createElementNS(SVGNS, "polyline"); e.setAttribute("points", pts.map(p => p.x + "," + p.y).join(" "));
  e.setAttribute("stroke", cv ? `var(${cv})` : "#c9bfa8"); e.setAttribute("stroke-width", "1.5"); e.setAttribute("fill", "none"); if (arrow) e.setAttribute("marker-end", "url(#dag-arrow)"); svg.append(e); };
const node = (n, x, y, isC, cv) => {
  const grp = document.createElementNS(SVGNS, "g"); grp.setAttribute("transform", `translate(${x},${y})`); grp.setAttribute("class", isC ? "dag-node dag-node-center" : "dag-node");
  const rect = document.createElementNS(SVGNS, "rect"); rect.setAttribute("width", g.nodeW); rect.setAttribute("height", g.nodeH); rect.setAttribute("rx", 3); rect.setAttribute("class", "dag-node-rect");
  if (n.color) { rect.style.stroke = n.color; rect.style.fill = "transparent"; } else if (cv) { rect.style.stroke = `var(${cv})`; rect.style.fill = `var(${cv}-bg, transparent)`; }
  grp.append(rect);
  const fo = document.createElementNS(SVGNS, "foreignObject"), sz = g.nodeH - 6; fo.setAttribute("x", 2); fo.setAttribute("y", 2); fo.setAttribute("width", sz); fo.setAttribute("height", sz); fo.style.pointerEvents = "none";
  const div = document.createElement("div"); div.style.cssText = "width:100%;height:100%;display:flex;align-items:center;justify-content:flex-start;"; nxIcon(div, n.icon); fo.append(div); grp.append(fo);
  const tx = g.nodeW - 10, ty = g.nodeH / 2, tg = document.createElementNS(SVGNS, "g"); tg.setAttribute("class", "dag-trigger");
  const hit = document.createElementNS(SVGNS, "rect"); hit.setAttribute("x", g.nodeW - 18); hit.setAttribute("width", 18); hit.setAttribute("height", g.nodeH); hit.setAttribute("fill", "transparent");
  const arr = document.createElementNS(SVGNS, "path"); arr.setAttribute("d", `M ${tx - 4} ${ty - 4} L ${tx + 2} ${ty} L ${tx - 4} ${ty + 4} z`);
  arr.setAttribute("class", isC || nxVisible(n, center).length ? "dag-trigger-arrow active" : "dag-trigger-arrow");
  tg.append(hit, arr); tg.addEventListener("click", e => nxTip(e, n, center));
  if (!isC) grp.addEventListener("click", e => { if (!e.target.closest(".dag-trigger")) nodeGo(n); });
  grp.append(tg); svg.append(grp); };
const incoming = (list, xN, y0, xCol, mX, cv) => { if (!list.length) return;
  const it = nxGroup(list, y0), dX = xCol < xN ? xN : xN + g.nodeW;
  it.forEach(({ n, y }) => { node(n, xN, y, false, cv); poly([{ x: dX, y: y + g.nodeH / 2 }, { x: xCol, y: y + g.nodeH / 2 }], false, cv); });
  const y1 = it[0].y + g.nodeH / 2, y2 = it[it.length - 1].y + g.nodeH / 2;
  if (y2 > y1) poly([{ x: xCol, y: y1 }, { x: xCol, y: y2 }], false, cv);
  const ne = Math.abs(y1 - yAxis) < Math.abs(y2 - yAxis) ? y1 : y2;
  poly([{ x: xCol, y: ne }, { x: xCol, y: yAxis }, { x: mX, y: yAxis }], false, cv); };
const outgoing = (list, xN, y0, xCol, cX, cv) => { if (!list.length) return;
  const it = nxGroup(list, y0), y1 = it[0].y + g.nodeH / 2, y2 = it[it.length - 1].y + g.nodeH / 2;
  const ne = Math.abs(y1 - yAxis) < Math.abs(y2 - yAxis) ? y1 : y2;
  poly([{ x: cX, y: yAxis }, { x: xCol, y: yAxis }, { x: xCol, y: ne }], false, cv);
  if (y2 > y1) poly([{ x: xCol, y: y1 }, { x: xCol, y: y2 }], false, cv);
  const aX = xCol < xN ? xN : xN + g.nodeW;
  it.forEach(({ n, y }) => { poly([{ x: xCol, y: y + g.nodeH / 2 }, { x: aX, y: y + g.nodeH / 2 }], true, cv); node(n, xN, y, false, cv); }); };
const yCT = yAxis - g.nodeH / 2;
if (hasL) { const L = nxColumn(nw, wt, sw, yAxis);
  incoming(nw, xL, L.upperY, xLC, xM, "--graph-nw"); incoming(wt, xL, L.middleY, xLC, xM, "--graph-wt"); incoming(sw, xL, L.lowerY, xLC, xM, "--graph-sw"); }
if (nr.length) incoming(nr, xC, yCT - (nrH + g.vSpacing * 2), xM, xM, "--graph-nr");
if (hasIn) poly([{ x: xM, y: yAxis }, { x: xC, y: yAxis }], true);
node(data.center, xC, yCT, true);
if (hasR) { const R = nxColumn(ne, et, se, yAxis);
  outgoing(ne, xR, R.upperY, xRC, xC + g.nodeW, "--graph-nw"); outgoing(et, xR, R.middleY, xRC, xC + g.nodeW, "--graph-wt"); outgoing(se, xR, R.lowerY, xRC, xC + g.nodeW, "--graph-sw"); }
if (st.length) outgoing(st, xC, yCT + g.nodeH + g.vSpacing * 2, xBC, xC + g.nodeW, "--graph-nr"); }
// nxList: список связей под графом; List : [ … ] — скрытые связи
function nxList(box, data) {
const skip = new Set((data.list || []).map(s => s.toLowerCase()));
const code = t => (data.codes || {})[t] || t, center = data.center.name;
const secs = [["NW", "nw", "nwType", "nw"], ["NE", "ne", "nwType", "nw"], ["WT", "wt", "wtType", "wt"], ["ET", "et", "wtType", "wt"],
  ["SW", "sw", "swType", "sw"], ["SE", "se", "swType", "sw"], ["NR", "nr", "nrType", "nr"], ["ST", "st", "nrType", "nr"]]
  .map(([l, s, t, c]) => ({ label: l + "-" + code(data[t]), nodes: data[s], cls: "dag-section-" + c, key: data[t] }))
  .concat((data.extra || []).flatMap(x => [{ label: "DX-" + code(x.type), nodes: x.direct, cls: "dag-section-extra", key: x.type }, { label: "IX-" + code(x.type), nodes: x.inverse, cls: "dag-section-extra", key: x.type }]))
  .filter(s => !skip.has((s.key || "").toLowerCase()));
const root = el("dag-list");
const link = (n, sec) => { const a = document.createElement("a"); a.href = "#"; a.textContent = n.name; a.className = sec ? "dag-link dag-link-secondary" : "dag-link"; a.onclick = e => { e.preventDefault(); nodeGo(n); }; return a; };
secs.forEach(sec => { if (!sec.nodes || !sec.nodes.length) return;
  const div = el("dag-section " + sec.cls), lab = document.createElement("span"); lab.className = "dag-section-label"; lab.textContent = sec.label + ": "; div.append(lab);
  sec.nodes.forEach((n, i) => { if (i) div.append(" "); div.append(link(n));
    const clean = nxVisible(n, center);
    if (clean.length) { div.append(" [ "); clean.forEach((nb, k) => { if (k) { const s = document.createElement("span"); s.className = "dag-link-sep"; s.textContent = " · "; div.append(s); } div.append(link(nb, true)); }); div.append(" ]"); } });
  root.append(div); });
box.append(root); }
