// epi/script/infra/satel.go
package main
import ("encoding/json"; "fmt"; "io"; "os"; "path/filepath"; "sort"; "strings"; "epi/script/format")
// спутники концепта — файлы рядом с ним: то же имя, другое расширение (md, json, svg, css, xml, html, txt)
// Content — тип спутника по умолчанию (md); карточка концепта, включения {{ }}, слияние спутников при union
// satFile: файл спутника ext концепта name; "" — концепта нет или тип не спутника
func (q *Script) satFile(name, ext string) string {
p := q.conceptPath(name)
if p == "" || !has(satTypes, ext) {return ""}; return strings.TrimSuffix(p, ".uic") + "." + ext }
// card: карточка концепта — имя, картинка, цвет, тип по умолчанию, спутники (нет — пустой список), строки
func (q *Script) card(name string) map[string]any {
p := q.conceptPath(name)
if p == "" {return map[string]any{"found": false}}
sats := []string{}
for _, t := range satTypes {if exists(strings.TrimSuffix(p, ".uic") + "." + t) {sats = append(sats, t)}}
ct := strings.ToLower(setting(p, "Content"))
if ct == "" {ct = "md"}
lines := []string{}
if b, err := os.ReadFile(p); err == nil {lines, _ = conceptLines(b)}
c := map[string]any{"found": true, "name": conceptName(p, ""), "content": ct, "sats": sats, "lines": lines, "color": setting(p, "Color")}
if q.iconFile(name) != "" {c["icon"] = "/api/icon?name=" + name}; return c }
// writeSat: записать спутник; прямой список Links — ссылки [[ ]] спутников md и html вместе, обратные у целей ведёт writeLink
func (q *Script) writeSat(name, ext, text string) error {
p := q.satFile(name, ext)
if p == "" {return os.ErrNotExist}
if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {return err}
if err := os.WriteFile(p, []byte(text), 0644); err != nil {return err}
if k := strings.ToLower(name); q.cidx[k] != "" {if ns := transcl(q.cidx[k]); len(ns) > 0 {q.tidx[k] = ns} else {delete(q.tidx, k)}}
if (ext != "md" && ext != "html") || !strings.HasPrefix(q.conceptPath(name), q.themaDir()) {return nil}
text = ""
for _, t := range []string{"md", "html"} {if b, err := os.ReadFile(q.satFile(name, t)); err == nil {text += string(b) + "\n"}}
seen := map[string]bool{}
var ns []string
for _, m := range reRefs.FindAllStringSubmatch(reFence.ReplaceAllString(text, ""), -1) {
	n := strings.TrimSpace(m[1])
	if k := strings.ToLower(n); n != "" && !seen[k] {seen[k] = true; ns = append(ns, n)}}
sort.Strings(ns)
old, _ := readLink(q.conceptFile(name), "Links")
if len(old) == len(ns) {same := true; for _, n := range ns {if !has(old, n) {same = false}}; if same {return nil}}
var pred []format.Value
for _, n := range ns {pred = append(pred, format.Value{Kind: format.Term, Text: n})}; return q.writeLink(conceptName(q.conceptFile(name), ""), "Links", pred, q.themaDir(), true) }
// transcl: трансклюзии концепта по файлу p — из спутников md, html, json (в стилях и svg их нет)
func transcl(p string) []string {
var ns []string
for _, t := range []string{"md", "html", "json"} {
	b, err := os.ReadFile(strings.TrimSuffix(p, ".uic") + "." + t)
	if err != nil {continue}
	for _, m := range reIncl.FindAllStringSubmatch(reFence.ReplaceAllString(string(b), ""), -1) {if n := strings.TrimSpace(m[1]); n != "" && !has(ns, n) {ns = append(ns, n)}}}; return ns }
// indexTranscl: указатель трансклюзий темы — вычисляемая связь, в файлы не пишется
func (q *Script) indexTranscl() {
q.tidx = map[string][]string{}
for k, p := range q.cidx {if ns := transcl(p); len(ns) > 0 {q.tidx[k] = ns}} }
// inclLists: прямой (кого включает) и обратный (кто включает) списки трансклюзий концепта
func (q *Script) inclLists(name string) ([]string, []string) {
k := strings.ToLower(name)
var inv []string
for o, ns := range q.tidx {if o != k && has(ns, name) {inv = append(inv, conceptName(q.cidx[o], ""))}}
sort.Strings(inv); return q.tidx[k], inv }
// mergeSats: спутники дубля dp → главного main; текст, который у главного уже есть, не дописывается
func mergeSats(main, dp string) error {
seps := map[string]string{"md": "\n\n---\n\n", "html": "\n", "txt": "\n", "css": "\n"}
for _, f := range conceptFiles(dp, main)[1:] {
	ext := strings.TrimPrefix(filepath.Ext(f[0]), ".")
	switch sep, app := seps[ext]; {
	case ext == "json" && exists(f[1]):
		var s, d map[string]any
		sb, _ := os.ReadFile(f[0]); db, _ := os.ReadFile(f[1])
		if json.Unmarshal(sb, &s) != nil || json.Unmarshal(db, &d) != nil {continue}
		for k, v := range s {
			if dv, ok := d[k]; ok {if fmt.Sprint(dv) == fmt.Sprint(v) {continue}; n := 2; for {if _, t := d[fmt.Sprintf("%s-%d", k, n)]; !t {break}; n++}; k = fmt.Sprintf("%s-%d", k, n)}
			d[k] = v}
		out, _ := json.MarshalIndent(d, "", "  ")
		if err := os.WriteFile(f[1], out, 0644); err != nil {return err}
	case app && exists(f[1]):
		a, _ := os.ReadFile(f[1]); b, _ := os.ReadFile(f[0])
		if strings.Contains(string(a), strings.TrimSpace(string(b))) {continue}
		if err := os.WriteFile(f[1], []byte(string(a)+sep+string(b)), 0644); err != nil {return err}
	case !exists(f[1]):
		os.MkdirAll(filepath.Dir(f[1]), 0755)
		if err := copyFile(f[0], f[1]); err != nil {return err}}}; return nil }
// copyFile: копия файла src в dst, папки создаются (clone, import, union)
func copyFile(src, dst string) error {
if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {return err}
in, err := os.Open(src)
if err != nil {return err}
defer in.Close()
out, err := os.Create(dst)
if err != nil {return err}
if _, err = io.Copy(out, in); err != nil {out.Close(); return err}; return out.Close() }
