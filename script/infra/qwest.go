// epi/script/infra/qwest.go
package main
import ("os"; "path/filepath"; "regexp"; "sort"; "strconv"; "strings"; "unicode/utf8"; "epi/script/format")
// Поиск (ason-spec 9.20, имя — пользователь 08.11.2026): препозиция qwest, как поиск ПИ
// (qwest.go, bxcli.js): «qwest слово» или «qwest "слово & слово"» — концепты темы, у
// которых каждое слово есть в имени, синонимах или тексте спутников md, html, json,
// xml, txt (сначала тип Content); «qwest ""» — все концепты темы. Ответ — число
// найденных; список — в панели (линза темы): картинка, имя, отрывок, тип спутника.
var qwestTypes = []string{"md", "html", "json", "xml", "txt"}
var reTags, reMarks, reSpace = regexp.MustCompile(`<[^>]*>`), regexp.MustCompile("[#*_`\\[\\]{}|>]+"), regexp.MustCompile(`\s+`)
// qtext: текст спутника без разметки (как StripContent в ПИ).
func qtext(s string) string {return strings.TrimSpace(reSpace.ReplaceAllString(reMarks.ReplaceAllString(reTags.ReplaceAllString(s, " "), " "), " ")) }
// qterms: слова запроса — строчными, разделитель «&».
func qterms(q string) []string {
var ts []string
for _, t := range strings.Split(strings.ToLower(q), "&") {if t = strings.TrimSpace(t); t != "" {ts = append(ts, t)}}; return ts }
// excerpt: отрывок вокруг найденного — по 80 знаков с каждой стороны.
func excerpt(text, low, t string) string {
pos := strings.Index(low, t)
rs, rp := []rune(text), utf8.RuneCountInString(low[:pos])
a, b := rp-80, rp+utf8.RuneCountInString(t)+80
if a < 0 {a = 0}
if b > len(rs) {b = len(rs)}; return "…" + string(rs[a:b]) + "…" }
// qwestFind: найденные концепты текущей темы по запросу q.
func (q *Script) qwestFind(query string) []map[string]string {return q.qwestIn(q.cidx, q.theme, query) }
// qwizFind: найденные концепты всех тем (qwizt, 9.20) — в каждой строке её тема.
func (q *Script) qwizFind(query string) []map[string]string {
ts, _ := q.themes()
var out []map[string]string
for _, t := range ts {idx := q.cidx; if t != q.theme {idx = themeIndex(filepath.Join(q.themaRoot(), t))}; out = append(out, q.qwestIn(idx, t, query)...)}; return out }
// qwestIn: найденные концепты указателя idx темы theme.
func (q *Script) qwestIn(idx map[string]string, theme, query string) []map[string]string {
ts := qterms(query)
var out []map[string]string
for k, p := range idx {
	name := conceptName(p, "")
	alias, _ := readLink(p, "Alias")
	head := strings.ToLower(name + " " + strings.Join(alias, " "))
	ct := strings.ToLower(setting(p, "Content"))
	types := append([]string{}, qwestTypes...)
	if has(types, ct) {types = append([]string{ct}, without(types, ct)...)}
	texts := map[string]string{}
	ok, hit, typ := true, "", ""
	for _, t := range ts {
		if strings.Contains(head, t) {continue}
		found := false
		for _, ty := range types {
			if _, read := texts[ty]; !read {b, _ := os.ReadFile(strings.TrimSuffix(p, ".uic") + "." + ty); texts[ty] = qtext(string(b))}
			if low := strings.ToLower(texts[ty]); strings.Contains(low, t) {found = true; if hit == "" {hit, typ = excerpt(texts[ty], low, t), ty}; break}}
		if !found {ok = false; break}}
	if !ok {continue}
	r := map[string]string{"name": name, "key": k, "excerpt": hit, "type": typ, "theme": theme}
	if theme == q.theme && q.iconFile(name) != "" {r["icon"] = "/api/icon?name=" + name} else if theme != q.theme {r["icon"] = "/api/icon?name=" + name + "&theme=" + theme}
	out = append(out, r)}
sort.Slice(out, func(i, j int) bool {return aLess(out[i]["name"], out[j]["name"])}); return out }
// qwest: препозиция поиска — операнд не вычисляется (слово, строка, число); ответ — число найденных.
func (q *Script) qwest(arg format.Value) (format.Value, bool) {
if q.theme == "" || arg.Kind != format.Term && arg.Kind != format.String && arg.Kind != format.Integer && arg.Kind != format.Decimal {return format.Value{}, false}
return format.Value{Kind: format.Integer, Text: strconv.Itoa(len(q.qwestFind(arg.Text)))}, true }
// qwizt: поиск во всех темах (имя — пользователь, 09.11.2026) — как qwest; ответ — число найденных.
func (q *Script) qwizt(arg format.Value) (format.Value, bool) {
if arg.Kind != format.Term && arg.Kind != format.String && arg.Kind != format.Integer && arg.Kind != format.Decimal {return format.Value{}, false}
return format.Value{Kind: format.Integer, Text: strconv.Itoa(len(q.qwizFind(arg.Text)))}, true }
// iconOther: картинка концепта другой темы — <папка концепта>/<имя>.svg или img/<имя>.*.
func (q *Script) iconOther(theme, name string) string {
if _, ok := q.themeExists(theme); !ok || strings.ContainsAny(name, "/\\") {return ""}
p := themeIndex(filepath.Join(q.themaRoot(), theme))[strings.ToLower(name)]
if p == "" {return ""}
base := strings.TrimSuffix(p, ".uic")
if exists(base + ".svg") {return base + ".svg"}
for _, e := range []string{"png", "jpg", "jpeg", "webp", "gif", "ico"} {if f := filepath.Join(filepath.Dir(p), "img", filepath.Base(base)+"."+e); exists(f) {return f}}; return "" }
