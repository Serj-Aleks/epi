// epi/script/infra/qwest.go
package main
import ("os"; "path/filepath"; "sort"; "strconv"; "strings"; "unicode/utf8"; "epi/script/format")
// поиск qwest: qwest слово, qwest "слово & слово" — концепты, у которых каждое слово есть в имени,
// синонимах или тексте спутников md, html, json, xml, txt; qwest "" — все; ответ — число, список — в панели
// qtext: текст спутника без разметки
func qtext(s string) string {return strings.TrimSpace(reSpace.ReplaceAllString(reMarks.ReplaceAllString(reTags.ReplaceAllString(s, " "), " "), " ")) }
// qterms: слова запроса — строчными, разделитель «&»
func qterms(q string) []string {
var ts []string
for _, t := range strings.Split(strings.ToLower(q), "&") {if t = strings.TrimSpace(t); t != "" {ts = append(ts, t)}}; return ts }
// excerpt: отрывок вокруг найденного — по 80 знаков с каждой стороны
func excerpt(text, low, t string) string {
pos := strings.Index(low, t)
rs, rp := []rune(text), utf8.RuneCountInString(low[:pos])
a, b := max(rp-80, 0), min(rp+utf8.RuneCountInString(t)+80, len(rs)); return "…" + string(rs[a:b]) + "…" }
// qwestFind: найденные концепты текущей темы по запросу q
func (q *Script) qwestFind(query string) []map[string]string {return q.qwestIn(q.cidx, q.theme, query) }
// qwizFind: найденные концепты всех тем (qwizt), в каждой строке её тема; нет — пустой список
func (q *Script) qwizFind(query string) []map[string]string {
ts, _ := q.themes()
out := []map[string]string{}
for _, t := range ts {idx := q.cidx; if t != q.theme {idx = themeIndex(filepath.Join(q.themaRoot(), t))}; out = append(out, q.qwestIn(idx, t, query)...)}; return out }
// qwestIn: найденные концепты указателя idx темы theme — taxon и rank (концепт с Rank), число параметров, спутники; нет — пустой список
func (q *Script) qwestIn(idx map[string]string, theme, query string) []map[string]string {
ts := qterms(query)
out := []map[string]string{}
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
	if rk := setting(p, "Rank"); rk != "" {r["taxon"], r["rank"] = "1", rk}
	np, sats := 0, []string{}
	if b, err := os.ReadFile(p); err == nil {ls, _ := conceptLines(b); for _, l := range ls {if isParam(l) {np++}}}
	for _, t := range satTypes {if exists(strings.TrimSuffix(p, ".uic") + "." + t) {sats = append(sats, t)}}
	r["params"], r["sats"] = strconv.Itoa(np), strings.Join(sats, " ")
	if theme == q.theme && q.iconFile(name) != "" {r["icon"] = "/api/icon?name=" + name} else if theme != q.theme {r["icon"] = "/api/icon?name=" + name + "&theme=" + theme}
	out = append(out, r)}
sort.Slice(out, func(i, j int) bool {return aLess(out[i]["name"], out[j]["name"])}); return out }
// qwest: препозиция поиска — операнд не вычисляется (слово, строка, число); ответ — число найденных
func (q *Script) qwest(arg format.Value) (format.Value, bool) {
if q.theme == "" || arg.Kind != format.Term && arg.Kind != format.String && arg.Kind != format.Integer && arg.Kind != format.Decimal {return format.Value{}, false}
return format.Value{Kind: format.Integer, Text: strconv.Itoa(len(q.qwestFind(arg.Text)))}, true }
// qwizt: поиск во всех темах, как qwest; ответ — число найденных
func (q *Script) qwizt(arg format.Value) (format.Value, bool) {
if arg.Kind != format.Term && arg.Kind != format.String && arg.Kind != format.Integer && arg.Kind != format.Decimal {return format.Value{}, false}
return format.Value{Kind: format.Integer, Text: strconv.Itoa(len(q.qwizFind(arg.Text)))}, true }
// iconOther: картинка концепта другой темы — <папка концепта>/<имя>.svg или img/<имя>.*
func (q *Script) iconOther(theme, name string) string {
if _, ok := q.themeExists(theme); !ok || strings.ContainsAny(name, "/\\") {return ""}
p := themeIndex(filepath.Join(q.themaRoot(), theme))[strings.ToLower(name)]
if p == "" {return ""}
base := strings.TrimSuffix(p, ".uic")
if exists(base + ".svg") {return base + ".svg"}
for _, e := range []string{"png", "jpg", "jpeg", "webp", "gif", "ico"} {if f := filepath.Join(filepath.Dir(p), "img", filepath.Base(base)+"."+e); exists(f) {return f}}; return "" }
// findIn: поиск внутри концепта — в имени и спутниках md, html, json, xml, txt концепта и синонимов
// каждое слово запроса (& — и) в одном месте; ответ — концепт, тип спутника, отрывок
func (q *Script) findIn(name, query string) []map[string]string {
p := q.conceptPath(name)
out := []map[string]string{}
ts := qterms(query)
if p == "" || len(ts) == 0 {return out}
paths := []string{p}
as, _ := readLink(p, "Alias")
for _, a := range as {if ap := q.conceptPath(a); ap != "" && !has(paths, ap) {paths = append(paths, ap)}}
for _, cp := range paths {
	n := conceptName(cp, "")
	if every(strings.ToLower(n), ts) {out = append(out, map[string]string{"name": n, "type": "", "excerpt": ""})}
	for _, ty := range qwestTypes {
		b, err := os.ReadFile(strings.TrimSuffix(cp, ".uic") + "." + ty)
		if err != nil {continue}
		text := qtext(string(b))
		low := strings.ToLower(text)
		if every(low, ts) {out = append(out, map[string]string{"name": n, "type": ty, "excerpt": excerpt(text, low, ts[0])})}}}; return out }
// every: все слова ts — в строке s
func every(s string, ts []string) bool {for _, t := range ts {if !strings.Contains(s, t) {return false}}; return true }
