// epi/script/infra/kater.go
package main
import ("fmt"; "io"; "os"; "path/filepath"; "regexp"; "strings"; "epi/script/format")
// Перенос данных ПИ (папка kater) в темы EPI (ason-spec 9.18): тема ПИ → thema/<тема>,
// концепт — по тому же пути из букв, файл — строки «Параметр : значение» (0.23).
// Связи ПИ — с обоими списками: alias → Alias, tags → Taxon, prefix → Prfix,
// suffix → Sufix, links → Links, attrib → Attrib, preposit → Preposit; настройки
// показа — параметрами с именами ПИ (List, Rank, Content, Graph, Color, Nord,
// S-west, West, N-west, Deep, Defs, Process, I-process, Tabs, I-sort, Themes).
// Пустое не переносится; name, uri, counts и индекс соседей выводятся заново.
// Спутники (md, svg, json, …) копируются; существующие темы не трогаются.
var katerLinks = map[string]string{"m": "Alias", "y": "Taxon", "f": "Prfix", "v": "Sufix", "j": "Links", "b": "Attrib", "p": "Preposit"}
var katerOrder = []string{"m", "y", "f", "v", "j", "b", "p", "l", "r", "h", "g", "k", "c", "s", "z", "t", "d", "w", "u", "o", "a", "e", "i"}
var katerShow = map[string]string{"l": "List", "r": "Rank", "h": "Content", "g": "Graph", "k": "Color", "c": "Nord", "s": "S-west", "z": "West", "t": "N-west", "d": "Deep", "w": "Defs", "u": "Process", "o": "I-process", "a": "Tabs", "e": "I-sort", "i": "Themes"}
// katerValue: значение настройки — как есть, если это одно значение (терм, число,
// блок), иначе строкой («+», «-nr»).
func katerValue(s string) string {
if vs, err := format.Parse(s); err == nil && len(vs) == 1 && vs[0].Kind != format.Op && vs[0].Kind != format.Literal {return format.Format(vs)}
return format.Value{Kind: format.String, Text: s}.String() }
// katerLists: «[ a b ] [ c ]» → прямой и обратный списки.
func katerLists(s string) ([]string, []string) {
parts := strings.SplitN(s, "]", 2)
list := func(p string) []string {return strings.Fields(strings.NewReplacer("[", " ", "]", " ").Replace(p))}
if len(parts) < 2 {return list(s), nil}; return list(parts[0]), list(parts[1]) }
// katerConcept: строки концепта EPI по слотам концепта ПИ.
func katerConcept(sys map[string]string) []string {
var lines []string
for _, k := range katerOrder {
	v := strings.TrimSpace(sys[k])
	if v == "" {continue}
	switch {
	case k == "m": lines = append(lines, "Alias : "+blk(strings.Fields(v)))
	case katerLinks[k] != "":
		d, i := katerLists(v)
		if len(d)+len(i) == 0 {continue}
		lines = append(lines, katerLinks[k]+" : "+blk(d)+" "+blk(i))
	default: lines = append(lines, piNames(katerShow[k]+" : "+katerValue(v)))}}; return lines }
func copyFile(src, dst string) error {
if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {return err}
in, err := os.Open(src)
if err != nil {return err}
defer in.Close()
out, err := os.Create(dst)
if err != nil {return err}
if _, err = io.Copy(out, in); err != nil {out.Close(); return err}; return out.Close() }
// importKater: темы ПИ из src в thema; отчёт — по теме: концептов, спутников.
func (q *Script) importKater(src string) ([]string, error) {
es, err := os.ReadDir(src)
if err != nil {return nil, err}
var report []string
for _, e := range es {
	if !e.IsDir() || e.Name() == "img" {continue}
	name := e.Name()
	if checkName(name) != nil {report = append(report, name+": имя темы не подходит — пропущена"); continue}
	if _, ok := q.themeExists(name); ok {report = append(report, name+": тема уже есть — пропущена"); continue}
	from, to := filepath.Join(src, name), filepath.Join(q.themaRoot(), name)
	nc, ns := 0, 0
	err := filepath.WalkDir(from, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {return err}
		rel, _ := filepath.Rel(from, path)
		dst := filepath.Join(to, rel)
		if !strings.HasSuffix(path, ".uic") {ns++; return copyFile(path, dst)}
		sys, _, err := readLegacy(path)
		if err != nil {return err}
		nc++; return writeLines(dst, katerConcept(sys))})
	if err != nil {return report, fmt.Errorf("%s: %w", name, err)}
	if err := os.MkdirAll(filepath.Join(to, "prtcl"), 0755); err != nil {return report, err}
	report = append(report, fmt.Sprintf("%s: концептов %d, спутников %d", name, nc, ns))}; return report, nil }
// blk: список имён — блок в каноническом виде: [ a b ] или [].
func blk(ns []string) string {if len(ns) == 0 {return "[]"}; return "[ " + strings.Join(ns, " ") + " ]" }
// piNames: значение настройки осей, графа и списка (N-west, West, S-west, Nord, Graph,
// List) — имена слотов ПИ заменяются именами связей EPI (tags → Taxon, prefix → Prfix …;
// 08.11.2026); гашение ПИ «"-nr -tags"» — блоком без кавычек и минусов «[ nr Taxon ]» (09.11.2026).
var piLinkNames = map[string]string{"tags": "Taxon", "links": "Links", "attrib": "Attrib", "preposit": "Preposit", "prefix": "Prfix", "suffix": "Sufix", "themes": "Themes"}
var rePiName = regexp.MustCompile(`\b(tags|links|attrib|preposit|prefix|suffix|themes)\b`)
func piNames(line string) string {
k := lineKey(line)
if !isParam(line) || !has([]string{"N-west", "West", "S-west", "Nord", "Graph", "List"}, k) {return line}
i := strings.Index(line, ":")
line = line[:i] + rePiName.ReplaceAllStringFunc(line[i:], func(s string) string {return piLinkNames[s]})
v := strings.TrimSpace(line[i+1:])
if k != "Graph" && k != "List" || !strings.HasPrefix(v, "\"") {return line}
var ts []string
for _, t := range strings.Fields(strings.Trim(v, "\"")) {if t = strings.TrimLeft(t, "-"); t != "" {ts = append(ts, t)}}
return k + " : [ " + strings.Join(ts, " ") + " ]" }
