// epi/script/infra/satel.go
package main
import ("os"; "path/filepath"; "regexp"; "sort"; "strings"; "epi/script/format")
// Спутники концепта (0.22, 9.18): файлы рядом с концептом — то же имя, другое
// расширение (md, json, svg, css, xml, html, txt); Content — тип спутника, открываемого
// по умолчанию (md). Карточка концепта — панель с линзами (вариант B, 08.11.2026).
var satTypes = []string{"md", "json", "svg", "css", "xml", "html", "txt"}
// satFile: файл спутника ext концепта name; "" — концепта нет или тип не спутника.
func (q *Script) satFile(name, ext string) string {
p := q.conceptPath(name)
if p == "" || !has(satTypes, ext) {return ""}; return strings.TrimSuffix(p, ".uic") + "." + ext }
// card: карточка концепта — имя, картинка, тип по умолчанию, есть ли спутники, строки концепта.
func (q *Script) card(name string) map[string]any {
p := q.conceptPath(name)
if p == "" {return map[string]any{"found": false}}
var sats []string
for _, t := range satTypes {if exists(strings.TrimSuffix(p, ".uic") + "." + t) {sats = append(sats, t)}}
ct := strings.ToLower(setting(p, "Content"))
if ct == "" {ct = "md"}
var lines []string
if b, err := os.ReadFile(p); err == nil {lines, _ = conceptLines(b)}
c := map[string]any{"found": true, "name": conceptName(p, ""), "content": ct, "sats": sats, "lines": lines}
if q.iconFile(name) != "" {c["icon"] = "/api/icon?name=" + name}; return c }
// reLink: ссылка [[Имя]] в контенте (вне блоков кода) — как SyncContentLinks в ПИ.
var reLink, reFence = regexp.MustCompile(`\[\[([^\]]+)\]\]`), regexp.MustCompile("(?s)```.*?```")
// writeSat: записать спутник; у md связь Links выводится из ссылок [[ ]] контента
// (как в ПИ): прямой список — ссылки, обратные у целей ведёт writeLink.
func (q *Script) writeSat(name, ext, text string) error {
p := q.satFile(name, ext)
if p == "" {return os.ErrNotExist}
if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {return err}
if err := os.WriteFile(p, []byte(text), 0644); err != nil {return err}
if k := strings.ToLower(name); q.cidx[k] != "" {if ns := transcl(q.cidx[k]); len(ns) > 0 {q.tidx[k] = ns} else {delete(q.tidx, k)}}
if ext != "md" || !strings.HasPrefix(q.conceptPath(name), q.themaDir()) {return nil}
seen := map[string]bool{}
var ns []string
for _, m := range reLink.FindAllStringSubmatch(reFence.ReplaceAllString(text, ""), -1) {
	n := strings.TrimSpace(m[1])
	if k := strings.ToLower(n); n != "" && !seen[k] {seen[k] = true; ns = append(ns, n)}}
sort.Strings(ns)
old, _ := readLink(q.conceptFile(name), "Links")
if len(old) == len(ns) {same := true; for _, n := range ns {if !has(old, n) {same = false}}; if same {return nil}}
var pred []format.Value
for _, n := range ns {pred = append(pred, format.Value{Kind: format.Term, Text: n})}; return q.writeLink(conceptName(q.conceptFile(name), ""), "Links", pred, q.themaDir(), true) }
// reIncl: трансклюзия {{Имя}} в контенте (вне блоков кода).
var reIncl = regexp.MustCompile(`\{\{([^{}]+)\}\}`)
// transcl: трансклюзии концепта по файлу p — из спутников md, html, json (в стилях и svg их нет).
func transcl(p string) []string {
var ns []string
for _, t := range []string{"md", "html", "json"} {
	b, err := os.ReadFile(strings.TrimSuffix(p, ".uic") + "." + t)
	if err != nil {continue}
	for _, m := range reIncl.FindAllStringSubmatch(reFence.ReplaceAllString(string(b), ""), -1) {if n := strings.TrimSpace(m[1]); n != "" && !has(ns, n) {ns = append(ns, n)}}}; return ns }
// indexTranscl: указатель трансклюзий темы — вычисляемая связь (9.19): в файлы не пишется.
func (q *Script) indexTranscl() {
q.tidx = map[string][]string{}
for k, p := range q.cidx {if ns := transcl(p); len(ns) > 0 {q.tidx[k] = ns}} }
// inclLists: прямой (кого включает) и обратный (кто включает) списки трансклюзий концепта.
func (q *Script) inclLists(name string) ([]string, []string) {
k := strings.ToLower(name)
var inv []string
for o, ns := range q.tidx {if o != k && has(ns, name) {inv = append(inv, conceptName(q.cidx[o], ""))}}
sort.Strings(inv); return q.tidx[k], inv }
// clone: препозиция «clone [ Исходный Новый ]» (имя — пользователь, 08.11.2026; как
// CloneConcept в ПИ): файл концепта, спутники и картинки img/<имя>.* копируются под
// новым именем в папку исходного; прямые списки связей — те же, у их целей клон
// добавляется в обратные; обратные списки клона пусты (на него ещё никто не ссылается).
// Ответ — имя клона.
func (q *Script) clone(arg format.Value) (format.Value, bool) {
if q.theme == "" || arg.Kind != format.Array || len(arg.Items) != 2 || arg.Items[0].Kind != format.Term || arg.Items[1].Kind != format.Term {return format.Value{}, false}
src, name := q.cidx[strings.ToLower(arg.Items[0].Text)], capital(arg.Items[1].Text)
if src == "" || q.conceptPath(name) != "" || checkName(name) != nil {return format.Value{}, false}
dst := filepath.Join(filepath.Dir(src), strings.ToLower(name)+".uic")
sb, db := strings.TrimSuffix(src, ".uic"), strings.TrimSuffix(dst, ".uic")
files := [][2]string{{src, dst}}
for _, t := range satTypes {if exists(sb + "." + t) {files = append(files, [2]string{sb + "." + t, db + "." + t})}}
sn, dn := filepath.Join(filepath.Dir(src), "img", filepath.Base(sb)), filepath.Join(filepath.Dir(src), "img", filepath.Base(db))
for _, e := range []string{"png", "jpg", "jpeg", "webp", "gif", "ico"} {if exists(sn + "." + e) {files = append(files, [2]string{sn + "." + e, dn + "." + e})}}
for _, f := range files {if exists(f[1]) {return format.Value{}, false}}
for _, f := range files {if err := copyFile(f[0], f[1]); err != nil {return format.Value{}, false}}
q.cidx[strings.ToLower(name)] = dst
b, _ := os.ReadFile(dst)
lines, _ := conceptLines(b)
for _, l := range lines {
	root, ok := q.linkRoot(lineKey(l))
	if !ok {continue}
	d, _ := readLink(dst, lineKey(l))
	writeLists(dst, lineKey(l), nil, nil)
	var pred []format.Value
	for _, n := range d {pred = append(pred, format.Value{Kind: format.Term, Text: n})}
	q.writeLink(name, lineKey(l), pred, root, true)}
if ns := transcl(dst); len(ns) > 0 {q.tidx[strings.ToLower(name)] = ns}
on, of := q.in.OnParam, q.in.OnFrame
q.in.OnParam, q.in.OnFrame = nil, nil
q.loadConceptsIn(dst)
q.in.OnParam, q.in.OnFrame = on, of; return format.Value{Kind: format.Term, Text: name}, true }
