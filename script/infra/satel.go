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
