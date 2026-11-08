// epi/script/infra/links.go
package main
import ("os"; "path/filepath"; "strings"; "epi/script/format")
// Связи концептов (ason-spec 9.17): параметры Taxon (обобщение), Links, Prfix, Sufix,
// Radix (словообразование). В файле концепта — два списка «[ прямой ] [ обратный ]»:
// автор пишет прямой (Жёлтый.Taxon : [ Цвет ]), обратный script ведёт сам —
// у Цвет появляется «Taxon : [ ] [ Жёлтый ]» (самоиндексирование, как в ПИ).
// linkRoot: где лежат цели связи: Taxon, Links, Attrib, Preposit — концепты темы;
// Prfix, Sufix, Radix — концепт темы, если он есть (термин — и самостоятельный
// фрейм), иначе общий лексикон lexic/prfix, lexic/sufix, lexic/radix (9.17, 9.18).
func (q *Script) linkRoot(param string) (string, bool) {
switch format.Key(param) {
case "Taxon", "Links", "Attrib", "Preposit": return q.themaDir(), true
case "Prfix", "Sufix", "Radix": return filepath.Join(q.dir, "lexic", strings.ToLower(param)), true}; return "", false }
// targetPath: файл цели t связи: концепт темы, если он есть, иначе — в root.
func (q *Script) targetPath(root, t string) string {
if p := q.conceptFile(t); exists(p) {return p}; return letterPath(root, t) }
// names: имена из предикации — блок [ A B ] или термы.
func names(vs []format.Value) []string {
var ns []string
for _, v := range vs {
	if v.Kind == format.Array {ns = append(ns, names(v.Items)...)} else if v.Kind == format.Term {ns = append(ns, v.Text)}}; return ns }
// readLink: прямой и обратный списки связи param в файле концепта path.
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
// writeLists: строка связи «Param : [ прямой ] [ обратный ]».
func writeLists(path, param string, d, i []string) error {
block := func(ns []string) format.Value {
	v := format.Value{Kind: format.Array}
	for _, n := range ns {v.Items = append(v.Items, format.Value{Kind: format.Term, Text: n})}; return v}
return writeParam(path, param, []format.Value{block(d), block(i)}) }
func has(ns []string, n string) bool {for _, x := range ns {if format.SameTerm(x, n) {return true}}; return false }
func without(ns []string, n string) []string {
var out []string
for _, x := range ns {if !format.SameTerm(x, n) {out = append(out, x)}}; return out }
// writeLink: прямой список связи у frame; у целей — обратный: добавленные
// получают frame, убранные его теряют.
func (q *Script) writeLink(frame, param string, pred []format.Value, root string) error {
path := q.conceptFile(frame)
old, inv := readLink(path, param)
d := names(pred)
if err := writeLists(path, param, d, inv); err != nil {return err}
for _, t := range old {
	if has(d, t) {continue}
	tp := q.targetPath(root, t)
	td, ti := readLink(tp, param)
	if err := writeLists(tp, param, td, without(ti, frame)); err != nil {return err}}
for _, t := range d {
	if has(old, t) {continue}
	tp := q.targetPath(root, t)
	td, ti := readLink(tp, param)
	if !has(ti, frame) {ti = append(ti, frame)}
	if err := writeLists(tp, param, td, ti); err != nil {return err}}; return nil }
// directOnly: строка связи без обратного списка — для контекста фрейма.
func directOnly(l string) string {
vs, err := format.Parse(strings.TrimSpace(l[strings.Index(l, ":")+1:]))
if err != nil || len(vs) < 2 {return l}; return lineKey(l) + " : " + format.Format(vs[:1]) }
// linkOrder: связи в локальном контексте — порядок осей как в ПИ (префикс,
// [атрибут], суффикс, [препозиция]), затем дополнительные: корень, таксон.
var linkOrder = []string{"Prfix", "Attrib", "Sufix", "Preposit", "Radix", "Taxon", "Links"}
// nexusData: связи концепта name — по файлам темы и лексикона (prfix, sufix,
// radix): синонимы и для каждой связи прямой и обратный списки; найден ли концепт.
func (q *Script) nexusData(name string) ([]string, map[string][2][]string, bool) {
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
// nexus: локальный контекст концепта (9.17) — «nexus Цвет» → { Alias : … Taxon : [ прямой ] [ обратный ] … }.
func (q *Script) nexus(arg format.Value) (format.Value, bool) {
if arg.Kind != format.Term || q.theme == "" {return format.Value{}, false}
alias, links, found := q.nexusData(arg.Text)
if !found {return format.Value{}, false}
var parts []string
if len(alias) > 0 {parts = append(parts, "Alias : [ "+strings.Join(alias, " ")+" ]")}
for _, l := range linkOrder {if s, ok := links[l]; ok {parts = append(parts, l+" : [ "+strings.Join(s[0], " ")+" ] [ "+strings.Join(s[1], " ")+" ]")}}
vs, err := format.Parse("{ " + strings.Join(parts, " ") + " }")
if err != nil || len(vs) != 1 {return format.Value{}, false}; return vs[0], true }
// piLink: имя слота связи ПИ в настройках осей (Nord, West, N-west, S-west) → связь EPI.
var piLink = map[string]string{"prefix": "Prfix", "attrib": "Attrib", "suffix": "Sufix", "preposit": "Preposit", "tags": "Taxon", "links": "Links", "radix": "Radix"}
// setting: значение настройки концепта (Nord : links) — текст без кавычек.
func setting(path, key string) string {
b, err := os.ReadFile(path)
if err != nil {return ""}
lines, _ := conceptLines(b)
for _, l := range lines {
	if !isParam(l) || !format.SameTerm(lineKey(l), key) {continue}
	vs, err := format.Parse(strings.TrimSpace(l[strings.Index(l, ":")+1:]))
	if err == nil && len(vs) == 1 {return vs[0].Text}}; return "" }
// nexusGraph: локальный контекст для графа — оси как в ПИ: северо-запад/северо-восток —
// prefix, запад/восток — attrib, юго-запад/юго-восток — suffix, север/юг — preposit;
// концепт переназначает оси настройками N-west, West, S-west, Nord (значения — имена
// слотов ПИ). Пустая ось занимается первой незанятой связью с данными (Taxon, Links,
// Radix). Слева и сверху — обратные списки, справа и снизу — прямые; у Taxon наоборот
// по вертикали: обобщения (прямой) — сверху.
func (q *Script) nexusGraph(name string) map[string]any {
alias, links, found := q.nexusData(name)
axes := []string{"nw", "wt", "sw", "nr"}
set := map[string]string{"nw": "N-west", "wt": "West", "sw": "S-west", "nr": "Nord"}
def := map[string]string{"nw": "Prfix", "wt": "Attrib", "sw": "Sufix", "nr": "Preposit"}
on := map[string]string{}
used := map[string]bool{}
for _, a := range axes {
	l := def[a]
	if v := piLink[strings.ToLower(setting(q.conceptFile(name), set[a]))]; v != "" {l = v}
	on[a], used[l] = l, true}
for _, a := range axes {
	if _, ok := links[on[a]]; ok {continue}
	for _, x := range []string{"Taxon", "Links", "Radix", "Prfix", "Sufix", "Attrib", "Preposit"} {
		if _, ok := links[x]; ok && !used[x] {used[on[a]], used[x], on[a] = false, true, x; break}}}
out := map[string]any{"center": name, "found": found, "alias": alias, "types": on}
pair := map[string]string{"nw": "ne", "wt": "et", "sw": "se", "nr": "st"}
for _, a := range axes {
	s := links[on[a]]
	in, dir := s[1], s[0]
	if a == "nr" && on[a] == "Taxon" {in, dir = s[0], s[1]}
	out[a], out[pair[a]] = in, dir}; return out }
// iconFile: картинка концепта — спутник (0.22), как в ПИ: <папка концепта>/<имя>.svg
// или <папка концепта>/img/<имя>.png|jpg|jpeg|webp|gif|ico; концепт — в теме или в лексиконе.
func (q *Script) iconFile(name string) string {
if name == "" || strings.ContainsAny(name, "/\\") {return ""}
roots := []string{q.themaDir()}
for _, c := range []string{"prfix", "sufix", "radix", "preposit", "atrbt", "constans"} {roots = append(roots, filepath.Join(q.dir, "lexic", c))}
base := strings.ToLower(name)
for _, r := range roots {
	dir := filepath.Dir(letterPath(r, name))
	if p := filepath.Join(dir, base+".svg"); exists(p) {return p}
	for _, ext := range []string{"png", "jpg", "jpeg", "webp", "gif", "ico"} {if p := filepath.Join(dir, "img", base+"."+ext); exists(p) {return p}}}; return "" }
