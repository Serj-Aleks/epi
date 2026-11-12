// epi/script/infra/nexus.go
package main
import ("os"; "path/filepath"; "sort"; "strconv"; "strings"; "epi/script/format")
// связи концептов — Taxon, Links, Attrib, Preposit, Prfix, Sufix, Radix; в файле два списка «[ прямой ] [ обратный ]»
// автор пишет прямой (Жёлтый.Taxon : [ Цвет ]), обратный script ведёт сам (Цвет: Taxon : [ ] [ Жёлтый ])
// linkRoot: где цели связи — Taxon, Links, Attrib, Preposit в теме; Prfix, Sufix, Radix — в теме, если есть, иначе в lexic
func (q *Medium) linkRoot(param string) (string, bool) {
switch format.Key(param) {
case "Taxon", "Links", "Attrib", "Preposit": return q.themaDir(), true
case "Prfix", "Sufix", "Radix": return filepath.Join(q.dir, "lexic", strings.ToLower(param)), true}; return "", false }
// targetPath: файл цели t связи: концепт темы, если он есть, иначе — в root
func (q *Medium) targetPath(root, t string) string {
if p := q.conceptFile(t); exists(p) {return p}; return letterPath(root, t) }
// names: имена из предикации — блок [ A B ] или термы
func names(vs []format.Value) []string {
var ns []string
for _, v := range vs {
	if v.Kind == format.Array {ns = append(ns, names(v.Items)...)} else if v.Kind == format.Term {ns = append(ns, v.Text)}}; return ns }
// readLink: прямой и обратный списки связи param в файле концепта path
func readLink(path, param string) ([]string, []string) {
b, err := os.ReadFile(path)
if err != nil {return nil, nil}
lines, err := conceptLines(b)
if err != nil {return nil, nil}
for _, l := range lines {
	if !isParam(l) || !format.SameTerm(lineKey(l), param) {continue}
	vs, err := format.Parse(strings.TrimSpace(l[strings.Index(l, ":")+1:]))
	if err != nil {return nil, nil}
	var d, i []string
	if len(vs) > 0 {d = names(vs[:1])}
	if len(vs) > 1 {i = names(vs[1:2])}; return d, i}; return nil, nil }
// writeLists: строка связи «Param : [ прямой ] [ обратный ]»
func writeLists(path, param string, d, i []string) error {return writeParam(path, param, []format.Value{listBlock(d), listBlock(i)}) }
// has, without: имя в списке; список без имени
func has(ns []string, n string) bool {for _, x := range ns {if format.SameTerm(x, n) {return true}}; return false }
func without(ns []string, n string) []string {
var out []string
for _, x := range ns {if !format.SameTerm(x, n) {out = append(out, x)}}; return out }
// writeLink: прямой список связи у frame, у целей обратный — добавленные получают frame, убранные теряют
// strict — только у существующих целей (ссылки контента)
func (q *Medium) writeLink(frame, param string, pred []format.Value, root string, strict bool) error {
path := q.conceptFile(frame)
old, inv := readLink(path, param)
d := names(pred)
if err := writeLists(path, param, d, inv); err != nil {return err}
for _, t := range old {
	if has(d, t) {continue}
	tp := q.targetPath(root, t)
	if strict && !exists(tp) {continue}
	q.known(tp)
	td, ti := readLink(tp, param)
	if err := writeLists(tp, param, td, without(ti, frame)); err != nil {return err}}
for _, t := range d {
	if has(old, t) {continue}
	tp := q.targetPath(root, t)
	if strict && !exists(tp) {continue}
	q.known(tp)
	td, ti := readLink(tp, param)
	if !has(ti, frame) {ti = append(ti, frame)}
	if err := writeLists(tp, param, td, ti); err != nil {return err}}; return nil }
// directOnly: строка связи без обратного списка (для контекста фрейма)
func directOnly(l string) string {
vs, err := format.Parse(strings.TrimSpace(l[strings.Index(l, ":")+1:]))
if err != nil || len(vs) < 2 {return l}; return lineKey(l) + " : " + format.Canon(vs[:1]) }
// nexusData: синонимы и связи концепта name (прямой и обратный списки) по файлам темы и лексикона; найден ли концепт
func (q *Medium) nexusData(name string) ([]string, map[string][2][]string, bool) {
paths := []string{q.conceptFile(name)}
for _, c := range []string{"prfix", "sufix", "radix"} {paths = append(paths, letterPath(filepath.Join(q.dir, "lexic", c), name))}
found := false
for _, p := range paths {if exists(p) {found = true}}
alias, _ := readLink(paths[0], "Alias")
links := map[string][2][]string{}
for _, l := range linkOrder {
	var d, i []string
	for _, p := range paths {pd, pi := readLink(p, l); d, i = append(d, pd...), append(i, pi...)}
	if len(d)+len(i) > 0 {links[l] = [2][]string{d, i}}}; return alias, links, found }
// nexus: nexus Имя — ответ имя концепта, контекст показывает панель (граф и список)
func (q *Medium) nexus(arg format.Value) (format.Value, bool) {
if arg.Kind != format.Term || q.theme == "" {return format.Value{}, false}
p := q.conceptPath(arg.Text)
if p == "" {return format.Value{}, false}; return format.Value{Kind: format.Term, Text: conceptName(p, "")}, true }
// graphType: имя связи графа по значению настройки (без учёта регистра); "" — не связь
func graphType(v string) string {for _, t := range graphTypes {if strings.EqualFold(t, v) {return t}}; return "" }
// nexusGraph: граф — оси nw, wt, sw, nr: Prfix, Attrib, Sufix, Preposit или по N-west, West, S-west, Nord
// слева и сверху обратные списки, справа и снизу прямые, остальное — extra; узлы — только существующие концепты
// соседи узла по той же связи — у входящего его прямой список, у исходящего обратный (в обоих есть центр), у центра — синонимы; Transcl вычисляется; Nord : Themes — ось тем
func (q *Medium) nexusGraph(name string) map[string]any {
path := q.conceptPath(name)
if path == "" {return map[string]any{"found": false}}
mk := func(n string) (gnode, string, bool) {
	p := q.conceptPath(n)
	if p == "" {return gnode{}, "", false}
	g := gnode{Name: conceptName(p, ""), Color: setting(p, "Color")}
	if q.iconFile(n) != "" {g.Icon = "/api/icon?name=" + n}; return g, p, true}
resolve := func(ns []string, slot string, inv bool) []gnode {
	out := []gnode{}
	for _, n := range ns {
		g, p, ok := mk(n)
		if !ok {continue}
		if slot != "" {nd, ni := readLink(p, slot); if slot == "Transcl" {nd, ni = q.inclLists(n)}; if inv {nd = ni}; for _, x := range nd {if xg, _, ok := mk(x); ok {g.Neighbors = append(g.Neighbors, gnode{Name: xg.Name})}}}
		out = append(out, g)}; return out}
alias, links, _ := q.nexusData(name)
if d, i := q.inclLists(name); len(d)+len(i) > 0 {links["Transcl"] = [2][]string{d, i}}
center, _, _ := mk(name)
center.Neighbors = resolve(alias, "", false)
spec := map[string]string{"nw": "Prfix", "wt": "Attrib", "sw": "Sufix", "nr": "Preposit"}
for a, k := range map[string]string{"nw": "N-west", "wt": "West", "sw": "S-west", "nr": "Nord"} {if t := graphType(setting(path, k)); t != "" {spec[a] = t}}
themes := strings.EqualFold(setting(path, "Nord"), "Themes")
if themes {spec["nr"] = "Themes"}
out := map[string]any{"found": true, "center": center, "codes": graphCodes, "graph": settingList(path, "Graph"), "list": settingList(path, "List"),
	"nwType": spec["nw"], "wtType": spec["wt"], "swType": spec["sw"], "nrType": spec["nr"]}
pair := map[string]string{"nw": "ne", "wt": "et", "sw": "se", "nr": "st"}
used := map[string]bool{}
for a, t := range spec {if t == "Themes" {continue}; s := links[t]; out[a], out[pair[a]] = resolve(s[1], t, false), resolve(s[0], t, true); used[t] = true}
if themes {out["nr"], out["st"] = q.themeNodes(path)}
var extra []map[string]any
for _, t := range graphTypes {
	if used[t] {continue}
	s := links[t]
	d, i := resolve(s[0], t, true), resolve(s[1], t, false)
	if len(d)+len(i) > 0 {extra = append(extra, map[string]any{"type": t, "direct": d, "inverse": i})}}
out["extra"] = extra; return out }
// syncAlias: группа синонимов — у каждого в Alias все остальные, выбывший теряет frame
// имена, которые не концепты темы, остаются; контекст участников заново, у frame его задаёт текущая предикация
func (q *Medium) syncAlias(frame string, old, now []string) error {
exist := func(n string) string {return q.cidx[strings.ToLower(n)]}
group := []string{conceptName(q.conceptFile(frame), "")}
add := func(n string) {if p := exist(n); p != "" && !has(group, conceptName(p, "")) {group = append(group, conceptName(p, ""))}}
for _, a := range now {add(a)}
for _, a := range now {if exist(a) != "" {as, _ := readLink(exist(a), "Alias"); for _, x := range as {add(x)}}}
put := func(n string, as []string) error {
	v := listBlock(as)
	if err := writeParam(exist(n), "Alias", []format.Value{v}); err != nil || format.SameTerm(n, frame) {return err}
	on := q.in.OnParam
	q.in.OnParam = nil
	defer func() {q.in.OnParam = on}()
	_, err := q.in.Run(n+".Alias : "+format.Canon([]format.Value{v}), q.env); return err}
for _, a := range old {
	if exist(a) == "" || has(group, a) {continue}
	as, _ := readLink(exist(a), "Alias")
	if err := put(conceptName(exist(a), ""), without(as, frame)); err != nil {return err}}
for _, m := range group {
	cur := now
	if !format.SameTerm(m, frame) {cur, _ = readLink(exist(m), "Alias")}
	var as []string
	for _, o := range group {if !format.SameTerm(o, m) {as = append(as, o)}}
	for _, x := range cur {if exist(x) == "" && !has(as, x) {as = append(as, x)}}
	if err := put(m, as); err != nil {return err}}; return nil }
// tokens: слова имени для нечёткого поиска: строчными, разделители — пробел, «-», «_»
func tokens(s string) []string {return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {return r == ' ' || r == '-' || r == '_'}) }
// overlap: слова совпадают по первым deep литерам, короче deep не сравниваются
func overlap(a, b [][]string, deep int) bool {
for _, x := range a {for _, y := range b {for _, s := range x {for _, t := range y {
	rs, rt := []rune(s), []rune(t)
	if len(rs) >= deep && len(rt) >= deep && string(rs[:deep]) == string(rt[:deep]) {return true}}}}}; return false }
// nameTokens: слова имени концепта и его синонимов
func nameTokens(p string) [][]string {
out := [][]string{tokens(conceptName(p, ""))}
as, _ := readLink(p, "Alias")
for _, a := range as {out = append(out, tokens(a))}; return out }
// themeNodes: темы на оси север–юг — север: слово названия совпадает со словом имени концепта
// юг: в теме концепты с совпадающими именами (соседи темы); глубина Deep (3), своя тема — нет
func (q *Medium) themeNodes(path string) ([]gnode, []gnode) {
deep := 3
if d, err := strconv.Atoi(setting(path, "Deep")); err == nil && d > 0 {deep = d}
me := nameTokens(path)
ts, _ := q.themes()
north, south := []gnode{}, []gnode{}
for _, t := range ts {
	if t == q.theme {continue}
	icon := ""
	if q.themeIcon(t) != "" {icon = "/api/themeicon?name=" + t}
	if overlap([][]string{tokens(t)}, me, deep) {north = append(north, gnode{Name: t, Icon: icon, Type: "theme"})}
	var hits []gnode
	for _, p := range themeIndex(filepath.Join(q.themaRoot(), t)) {if overlap(nameTokens(p), me, deep) {hits = append(hits, gnode{Name: conceptName(p, ""), Theme: t})}}
	sort.Slice(hits, func(i, j int) bool {return aLess(hits[i].Name, hits[j].Name)})
	if len(hits) > 0 {south = append(south, gnode{Name: t, Icon: icon, Type: "theme", Neighbors: hits})}}; return north, south }
