// epi/script/infra/links.go
package main
import ("os"; "path/filepath"; "strings"; "epi/script/format")
// Связи концептов (ason-spec 9.17): параметры Taxon (обобщение), Prfix, Sufix,
// Radix (словообразование). В файле концепта — два списка «[ прямой ] [ обратный ]»:
// автор пишет прямой (Жёлтый.Taxon : [ Цвет ]), обратный script ведёт сам —
// у Цвет появляется «Taxon : [ ] [ Жёлтый ]» (самоиндексирование, как в ПИ).
// linkRoot: где лежат цели связи: Taxon — концепты темы; Prfix, Sufix, Radix —
// общий лексикон lexic/prfix, lexic/sufix, lexic/radix.
func (q *Script) linkRoot(param string) (string, bool) {
switch format.Key(param) {
case "Taxon": return q.themaDir(), true
case "Prfix", "Sufix", "Radix": return filepath.Join(q.dir, "lexic", strings.ToLower(param)), true}; return "", false }
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
	tp := letterPath(root, t)
	td, ti := readLink(tp, param)
	if err := writeLists(tp, param, td, without(ti, frame)); err != nil {return err}}
for _, t := range d {
	if has(old, t) {continue}
	tp := letterPath(root, t)
	td, ti := readLink(tp, param)
	if !has(ti, frame) {ti = append(ti, frame)}
	if err := writeLists(tp, param, td, ti); err != nil {return err}}; return nil }
// directOnly: строка связи без обратного списка — для контекста фрейма.
func directOnly(l string) string {
vs, err := format.Parse(strings.TrimSpace(l[strings.Index(l, ":")+1:]))
if err != nil || len(vs) < 2 {return l}; return lineKey(l) + " : " + format.Format(vs[:1]) }
// linkOrder: связи в локальном контексте — порядок осей как в ПИ (префикс,
// [атрибут], суффикс, [препозиция]), затем дополнительные: корень, таксон.
var linkOrder = []string{"Prfix", "Sufix", "Radix", "Taxon"}
// nexus: локальный контекст концепта (9.17) — «nexus Цвет» → { Alias : … Taxon : [ прямой ] [ обратный ] … };
// концепт ищется в теме и в лексиконе (prfix, sufix, radix).
func (q *Script) nexus(arg format.Value) (format.Value, bool) {
if arg.Kind != format.Term || q.theme == "" {return format.Value{}, false}
name := arg.Text
paths := []string{q.conceptFile(name)}
for _, c := range []string{"prfix", "sufix", "radix"} {paths = append(paths, letterPath(filepath.Join(q.dir, "lexic", c), name))}
found := false
var parts []string
for _, p := range paths {if exists(p) {found = true}}
if !found {return format.Value{}, false}
if d, _ := readLink(paths[0], "Alias"); len(d) > 0 {parts = append(parts, "Alias : [ "+strings.Join(d, " ")+" ]")}
for _, l := range linkOrder {
	var d, i []string
	for _, p := range paths {pd, pi := readLink(p, l); d, i = append(d, pd...), append(i, pi...)}
	if len(d)+len(i) == 0 {continue}
	parts = append(parts, l+" : [ "+strings.Join(d, " ")+" ] [ "+strings.Join(i, " ")+" ]")}
vs, err := format.Parse("{ " + strings.Join(parts, " ") + " }")
if err != nil || len(vs) != 1 {return format.Value{}, false}; return vs[0], true }
