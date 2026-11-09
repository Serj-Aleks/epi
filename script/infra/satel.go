// epi/script/infra/satel.go
package main
import ("encoding/json"; "fmt"; "os"; "path/filepath"; "regexp"; "sort"; "strconv"; "strings"; "epi/script/disp"; "epi/script/format")
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
if src == "" {q.in.RawErr = fmt.Errorf("концепта %s в теме нет", arg.Items[0].Text); return format.Value{}, false}
if checkName(name) != nil {q.in.RawErr = fmt.Errorf("имя %s — с прописной буквы, без пробелов", name); return format.Value{}, false}
if q.conceptPath(name) != "" {q.in.RawErr = fmt.Errorf("концепт %s уже есть", name); return format.Value{}, false}
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
// refKeys: параметры концепта, в значениях которых — имена концептов (refSlots ПИ).
var refKeys = []string{"Alias", "Attrib", "Preposit", "Prfix", "Sufix", "Radix", "Taxon", "Links", "Defs", "I-process", "I-sort"}
var reToken, reRefMark = regexp.MustCompile(`[\p{L}\p{N}_-]+`), regexp.MustCompile(`(\[\[|\{\{)([^\]\}]+)(\]\]|\}\})`)
// replaceRefs: имя old → name во всех концептах темы и лексикона связей (prfix, sufix,
// radix): в значениях параметров refKeys и в ссылках и трансклюзиях текстовых спутников
// (как replaceRefs ПИ).
func (q *Script) replaceRefs(old, name string) error {
swap := func(s string) string {return reToken.ReplaceAllStringFunc(s, func(t string) string {if format.SameTerm(t, old) {return name}; return t})}
mark := func(s string) string {return reRefMark.ReplaceAllStringFunc(s, func(m string) string {
	sub := reRefMark.FindStringSubmatch(m)
	if format.SameTerm(strings.TrimSpace(sub[2]), old) && (sub[1] == "[[") == (sub[3] == "]]") {return sub[1] + name + sub[3]}; return m})}
roots := []string{q.themaDir()}
for _, c := range []string{"prfix", "sufix", "radix"} {roots = append(roots, filepath.Join(q.dir, "lexic", c))}
for _, r := range roots {
	err := filepath.WalkDir(r, func(path string, d os.DirEntry, err error) error {
		if err != nil {return nil}
		if d.IsDir() && d.Name() == "prtcl" {return filepath.SkipDir}
		if d.IsDir() || !strings.HasSuffix(path, ".uic") {return nil}
		b, err := os.ReadFile(path)
		if err != nil {return err}
		lines, err := conceptLines(b)
		if err != nil {return nil}
		changed := false
		for i, l := range lines {
			if !isParam(l) || !has(refKeys, lineKey(l)) {continue}
			k := strings.Index(l, ":")
			if v := l[:k] + swap(l[k:]); v != l {lines[i], changed = v, true}}
		if changed {if err := writeLines(path, lines); err != nil {return err}}
		for _, t := range qwestTypes {
			sp := strings.TrimSuffix(path, ".uic") + "." + t
			sb, err := os.ReadFile(sp)
			if err != nil {continue}
			if v := mark(string(sb)); v != string(sb) {if err := os.WriteFile(sp, []byte(v), 0644); err != nil {return err}}}; return nil})
	if err != nil {return err}}; return nil }
// rename: препозиция «rename [ Старое Новое ]» (имя — пользователь, 09.11.2026; как
// RenameConcept ПИ): файлы концепта, спутников и img/<имя>.* — под новым именем в той же
// папке; имя заменяется во всех связях и в контенте; контекст темы — заново. Ответ — новое имя.
func (q *Script) rename(arg format.Value) (format.Value, bool) {
if q.theme == "" || arg.Kind != format.Array || len(arg.Items) != 2 || arg.Items[0].Kind != format.Term || arg.Items[1].Kind != format.Term {return format.Value{}, false}
src, name := q.cidx[strings.ToLower(arg.Items[0].Text)], capital(arg.Items[1].Text)
if src == "" {q.in.RawErr = fmt.Errorf("концепта %s в теме нет", arg.Items[0].Text); return format.Value{}, false}
if checkName(name) != nil {q.in.RawErr = fmt.Errorf("имя %s — с прописной буквы, без пробелов", name); return format.Value{}, false}
if p := q.conceptPath(name); p != "" && !strings.EqualFold(arg.Items[0].Text, name) {q.in.RawErr = fmt.Errorf("концепт %s уже есть", name); return format.Value{}, false}
old := conceptName(src, "")
dst := filepath.Join(filepath.Dir(src), strings.ToLower(name)+".uic")
if dst != src {
	sb, db := strings.TrimSuffix(src, ".uic"), strings.TrimSuffix(dst, ".uic")
	files := [][2]string{{src, dst}}
	for _, t := range satTypes {if exists(sb + "." + t) {files = append(files, [2]string{sb + "." + t, db + "." + t})}}
	sn, dn := filepath.Join(filepath.Dir(src), "img", filepath.Base(sb)), filepath.Join(filepath.Dir(src), "img", filepath.Base(db))
	for _, e := range []string{"png", "jpg", "jpeg", "webp", "gif", "ico"} {if exists(sn + "." + e) {files = append(files, [2]string{sn + "." + e, dn + "." + e})}}
	for _, f := range files {if exists(f[1]) {return format.Value{}, false}}
	for _, f := range files {if err := os.Rename(f[0], f[1]); err != nil {return format.Value{}, false}}}
if err := q.replaceRefs(old, name); err != nil {return format.Value{}, false}
q.indexConcepts()
q.env = disp.NewEnv(nil)
if q.loadAliases() != nil || q.loadConcepts() != nil {return format.Value{}, false}; return format.Value{Kind: format.Term, Text: name}, true }
// tags: теги темы — концепты с параметром Rank (как в ПИ: rank задан — концепт-тег);
// имя, цвет, картинка; по имени.
func (q *Script) tags() []map[string]string {
out := []map[string]string{}
for _, p := range q.cidx {
	if setting(p, "Rank") == "" {continue}
	n := conceptName(p, "")
	t := map[string]string{"name": n, "color": setting(p, "Color")}
	if q.iconFile(n) != "" {t["icon"] = "/api/icon?name=" + n}
	out = append(out, t)}
sort.Slice(out, func(i, j int) bool {return aLess(out[i]["name"], out[j]["name"])}); return out }
// children: дочерние концепта — обратный список Taxon, только существующие (GetChildren ПИ);
// tag — дочерний сам тег (задан Rank).
func (q *Script) children(name string) []map[string]any {
p := q.conceptPath(name)
if p == "" {return []map[string]any{}}
_, inv := readLink(p, "Taxon")
out := []map[string]any{}
for _, n := range inv {if c := q.conceptPath(n); c != "" {out = append(out, map[string]any{"name": conceptName(c, ""), "tag": setting(c, "Rank") != ""})}}
sort.SliceStable(out, func(i, j int) bool {return aLess(out[i]["name"].(string), out[j]["name"].(string))}); return out }
// process: ситуации процесса концепта (как GetConceptProcess ПИ) — тройки «объект
// препозиция объект» из значений json-спутника концепта и концептов I-process; строка —
// номер, три имени, есть ли концепт, его синонимы-концепты.
func (q *Script) process(name string) []map[string]any {
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
	sort.Slice(keys, func(i, j int) bool {a, ea := strconv.Atoi(keys[i]); b, eb := strconv.Atoi(keys[j]); if ea == nil && eb == nil {return a < b}; return keys[i] < keys[j]})
	for _, k := range keys {
		s, _ := sat[k].(string)
		if f := strings.Fields(s); len(f) == 3 {out = append(out, map[string]any{"key": len(out) + 1, "a": cell(f[0]), "p": cell(f[1]), "b": cell(f[2])})}}}; return out }
// aOrder: место литеры в алфавите темы; пробел, «-», «,» — раньше всех (BuildCharOrder ПИ).
var aOrder = func() map[rune]int {
m := map[rune]int{' ': -1, '-': -1, ',': -1}
pos := 0
for _, c := range aSort {if c != ' ' {m[c] = pos; pos++}}; return m}()
// aLess: a раньше b в алфавите темы (CompareByASort ПИ); литеры вне алфавита — в конце.
func aLess(a, b string) bool {
ra, rb := []rune(strings.ToLower(a)), []rune(strings.ToLower(b))
for i := 0; i < len(ra) && i < len(rb); i++ {
	oa, ok := aOrder[ra[i]]
	if !ok {oa = limit}
	ob, ok := aOrder[rb[i]]
	if !ok {ob = limit}
	if oa != ob {return oa < ob}}; return len(ra) < len(rb) }
