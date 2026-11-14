// epi/script/infra/lense.go
package main
import ("encoding/json"; "fmt"; "os"; "sort"; "strconv"; "strings")
// данные линз карточки: таксоны, дочерние, процесс и ситуации, словари, алфавит темы
// taxa: таксоны темы — концепты с параметром Rank, имя, цвет, картинка, в алфавите темы
func (q *Medium) taxa() []map[string]string {
out := []map[string]string{}
for _, p := range q.cidx {
	if setting(p, "Rank") == "" {continue}
	n := conceptName(p, "")
	t := map[string]string{"name": n, "color": setting(p, "Color")}
	if q.iconFile(n) != "" {t["icon"] = "/api/icon?name=" + n}
	out = append(out, t)}
sort.Slice(out, func(i, j int) bool {return aLess(out[i]["name"], out[j]["name"])}); return out }
// children: дочерние концепта — обратный список Taxon, только существующие; taxon — дочерний сам таксон (задан Rank)
func (q *Medium) children(name string) []map[string]any {
p := q.conceptPath(name)
if p == "" {return []map[string]any{}}
_, inv := readLink(p, "Taxon")
out := []map[string]any{}
for _, n := range inv {if c := q.conceptPath(n); c != "" {out = append(out, map[string]any{"name": conceptName(c, ""), "taxon": setting(c, "Rank") != ""})}}
sort.SliceStable(out, func(i, j int) bool {return aLess(out[i]["name"].(string), out[j]["name"].(string))}); return out }
// process: ситуации процесса — тройки «объект препозиция объект» из json-спутника концепта
// и концептов I-process; строка — номер, три имени, есть ли концепт, его синонимы; по номеру ситуации
func (q *Medium) process(name string) []map[string]any {
p := q.cidx[strings.ToLower(name)]
out := []map[string]any{}
if p == "" || setting(p, "Process") == "" && len(settingList(p, "Process")) == 0 {return out}
paths := []string{p}
for _, n := range settingList(p, "I-process") {if x := q.cidx[strings.ToLower(n)]; x != "" {paths = append(paths, x)}}
cell := func(n string) map[string]any {
	c := map[string]any{"name": n, "exists": false}
	if x := q.conceptPath(n); x != "" {
		c["name"], c["exists"] = conceptName(x, ""), true
		as, _ := readLink(x, "Alias")
		var ok []string
		for _, a := range as {if y := q.conceptPath(a); y != "" {ok = append(ok, conceptName(y, ""))}}
		c["alias"] = ok}; return c}
for _, cp := range paths {
	b, err := os.ReadFile(strings.TrimSuffix(cp, ".uic") + ".json")
	if err != nil {continue}
	var sat map[string]any
	if json.Unmarshal(b, &sat) != nil {continue}
	keys := make([]string, 0, len(sat))
	for k := range sat {keys = append(keys, k)}
	sort.Slice(keys, func(i, j int) bool {a, ea := sitNum(keys[i]); b, eb := sitNum(keys[j]); if ea && eb {return a < b}; return keys[i] < keys[j]})
	for _, k := range keys {
		s, _ := sat[k].(string)
		if f := strings.Fields(s); len(f) == 3 {out = append(out, map[string]any{"key": k, "a": cell(f[0]), "p": cell(f[1]), "b": cell(f[2])})}}}
sort.SliceStable(out, func(i, j int) bool {a, ea := sitNum(out[i]["key"].(string)); b, eb := sitNum(out[j]["key"].(string)); return ea && (!eb || a < b)}); return out }
// aLess: a раньше b в алфавите темы; литеры вне алфавита — в конце
func aLess(a, b string) bool {
ra, rb := []rune(strings.ToLower(a)), []rune(strings.ToLower(b))
for i := 0; i < len(ra) && i < len(rb); i++ {
	oa, ok := aOrder[ra[i]]
	if !ok {oa = limit}
	ob, ok := aOrder[rb[i]]
	if !ok {ob = limit}
	if oa != ob {return oa < ob}}; return len(ra) < len(rb) }
// dict: словари концепта — концепты из его Defs, словарь — json-спутник «термин : определение»
// Deep — первые литеры слова для нечёткого поиска (по умолчанию 3); длинные термины раньше
func (q *Medium) dict(name string) map[string]any {
p := q.conceptPath(name)
out := map[string]any{"deep": 3, "terms": []map[string]string{}}
if p == "" {return out}
if d, err := strconv.Atoi(setting(p, "Deep")); err == nil && d > 0 {out["deep"] = d}
var terms []map[string]string
for _, n := range settingList(p, "Defs") {
	dp := q.conceptPath(n)
	if dp == "" {continue}
	b, err := os.ReadFile(strings.TrimSuffix(dp, ".uic") + ".json")
	if err != nil {continue}
	var sat map[string]any
	if json.Unmarshal(b, &sat) != nil {continue}
	for k, v := range sat {if s, ok := v.(string); ok && strings.TrimSpace(k) != "" {terms = append(terms, map[string]string{"term": k, "def": s, "dict": conceptName(dp, "")})}}}
sort.Slice(terms, func(i, j int) bool {return len(terms[i]["term"]) > len(terms[j]["term"])})
if terms != nil {out["terms"] = terms}; return out }
// situation: новая ситуация «a p b» в json концепта name — номер на единицу больше наибольшего в теме
func (q *Medium) situation(name, a, p, b string) error {
cp := q.cidx[strings.ToLower(name)]
if cp == "" {return fmt.Errorf("концепта %s в теме нет", name)}
if a == "" || p == "" || b == "" || strings.ContainsAny(a+p+b, " \t") {return fmt.Errorf("ситуация — три слова: объект, препозиция, объект")}
next := 1
for _, x := range q.cidx {
	jb, err := os.ReadFile(strings.TrimSuffix(x, ".uic") + ".json")
	if err != nil {continue}
	var m map[string]any
	if json.Unmarshal(jb, &m) != nil {continue}
	for k, v := range m {if s, ok := v.(string); ok && len(strings.Fields(s)) == 3 {if n, ok := sitNum(k); ok && n >= next {next = n + 1}}}}
jp := strings.TrimSuffix(cp, ".uic") + ".json"
m := map[string]any{}
if jb, err := os.ReadFile(jp); err == nil {if json.Unmarshal(jb, &m) != nil {return fmt.Errorf("%s: json не читается", jp)}}
m[strconv.Itoa(next)] = a + " " + p + " " + b
out, _ := json.MarshalIndent(m, "", "  "); return os.WriteFile(jp, out, 0644) }
// sitNum: номер ситуации по ключу
func sitNum(k string) (int, bool) {n, err := strconv.Atoi(k); return n, err == nil }
// params: имена параметров для подбора в ⊞ — системные и те, что уже есть в концептах темы, в алфавите
func (q *Medium) params() []string {
seen, out := map[string]bool{}, []string{}
add := func(k string) {if l := strings.ToLower(k); !seen[l] {seen[l] = true; out = append(out, k)}}
for _, k := range sysParams {add(k)}
for _, p := range q.cidx {if b, err := os.ReadFile(p); err == nil {ls, _ := conceptLines(b); for _, l := range ls {if isParam(l) {add(lineKey(l))}}}}
sort.Slice(out, func(i, j int) bool {return strings.ToLower(out[i]) < strings.ToLower(out[j])}); return out }
