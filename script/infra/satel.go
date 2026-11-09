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
sats := []string{} // нет спутников — пустой список, не null
for _, t := range satTypes {if exists(strings.TrimSuffix(p, ".uic") + "." + t) {sats = append(sats, t)}}
ct := strings.ToLower(setting(p, "Content"))
if ct == "" {ct = "md"}
lines := []string{}
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
files := conceptFiles(src, dst)
for _, f := range files {if exists(f[1]) {return format.Value{}, false}}
for _, f := range files {if err := copyFile(f[0], f[1]); err != nil {return format.Value{}, false}}
q.adopt(name, dst); return format.Value{Kind: format.Term, Text: name}, true }
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
			if v := l[:k] + swap(l[k:]); v != l {lines[i], changed = dedupe(v), true}}
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
	files := conceptFiles(src, dst)
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
// dict: словари концепта (9.21, решено 09.11.2026) — концепты из его Defs; словарь —
// json-спутник из пар «термин : определение» (левая и правая части); Deep — глубина
// нечёткого поиска (первые Deep литер слова, по умолчанию 3, как в ПИ).
func (q *Script) dict(name string) map[string]any {
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
sort.Slice(terms, func(i, j int) bool {return len(terms[i]["term"]) > len(terms[j]["term"])}) // длинные термины — раньше
if terms != nil {out["terms"] = terms}; return out }
// conceptFiles: пары «откуда → куда» для концепта src → dst: файл концепта, спутники,
// картинки img/<имя>.* (conceptFiles ПИ).
func conceptFiles(src, dst string) [][2]string {
sb, db := strings.TrimSuffix(src, ".uic"), strings.TrimSuffix(dst, ".uic")
files := [][2]string{{src, dst}}
for _, t := range satTypes {if exists(sb + "." + t) {files = append(files, [2]string{sb + "." + t, db + "." + t})}}
sn, dn := filepath.Join(filepath.Dir(src), "img", filepath.Base(sb)), filepath.Join(filepath.Dir(dst), "img", filepath.Base(db))
for _, e := range []string{"png", "jpg", "jpeg", "webp", "gif", "ico"} {if exists(sn + "." + e) {files = append(files, [2]string{sn + "." + e, dn + "." + e})}}; return files }
// importC: препозиция «import [ Тема Имя … ]» (решено 09.11.2026; как ImportConcepts
// ПИ): из темы-источника — названные концепты, их синонимы, Taxon и Links (прямые и
// обратные) и ещё уровень связей от них; концепты, что в текущей теме есть, — пропускаются;
// файлы и спутники — по тому же пути; связи — как у clone. Ответ — число перенесённых.
func (q *Script) importC(arg format.Value) (format.Value, bool) {
if q.theme == "" || arg.Kind != format.Array || len(arg.Items) < 2 || arg.Items[0].Kind != format.Term {return format.Value{}, false}
src, ok := q.themeExists(arg.Items[0].Text)
if !ok {q.in.RawErr = fmt.Errorf("темы %s нет", arg.Items[0].Text); return format.Value{}, false}
if src == q.theme {q.in.RawErr = fmt.Errorf("%s — текущая тема", src); return format.Value{}, false}
sdir := filepath.Join(q.themaRoot(), src)
sidx := themeIndex(sdir)
var got []string
seen := map[string]bool{}
add := func(n string) bool {k := strings.ToLower(n); if seen[k] || sidx[k] == "" {return false}; seen[k] = true; got = append(got, k); return true}
links := func(k string) {
	p := sidx[k]
	as, _ := readLink(p, "Alias")
	for _, a := range as {add(a)}
	for _, l := range []string{"Taxon", "Links"} {d, i := readLink(p, l); for _, n := range append(d, i...) {add(n)}}}
for _, v := range arg.Items[1:] {
	if v.Kind != format.Term {return format.Value{}, false}
	if sidx[strings.ToLower(v.Text)] == "" {q.in.RawErr = fmt.Errorf("концепта %s в теме %s нет", v.Text, src); return format.Value{}, false}
	if add(v.Text) {links(strings.ToLower(v.Text))}}
for _, k := range append([]string{}, got...) {links(k)}
var done [][2]string
for _, k := range got {
	if q.cidx[k] != "" {continue}
	rel, _ := filepath.Rel(sdir, sidx[k])
	dst := filepath.Join(q.themaDir(), rel)
	files := conceptFiles(sidx[k], dst)
	if exists(dst) {continue}
	for _, f := range files {if exists(f[1]) {continue}; os.MkdirAll(filepath.Dir(f[1]), 0755); if err := copyFile(f[0], f[1]); err != nil {q.in.RawErr = err; return format.Value{}, false}}
	q.cidx[k] = dst
	done = append(done, [2]string{conceptName(dst, ""), dst})}
for _, d := range done {clearInverse(d[1])} // сначала все обратные пусты, потом прямые — у целей (и у перенесённых друг у друга)
for _, d := range done {q.relink(d[0], d[1])}; return format.Value{Kind: format.Integer, Text: strconv.Itoa(len(done))}, true }
// adopt: скопированный в тему концепт name (файл dst) — в указатель и контекст; обратные
// списки его связей — пусты, у целей прямых — он в обратных (clone).
func (q *Script) adopt(name, dst string) {clearInverse(dst); q.relink(name, dst) }
// clearInverse: обратные списки связей концепта — пусты (на скопированный никто не ссылается).
func clearInverse(dst string) {
b, _ := os.ReadFile(dst)
lines, _ := conceptLines(b)
for _, l := range lines {if _, ok := (&Script{}).linkRoot(lineKey(l)); ok {d, _ := readLink(dst, lineKey(l)); writeLists(dst, lineKey(l), d, nil)}} }
// relink: прямые списки связей концепта — у целей он в обратных; трансклюзии, контекст.
func (q *Script) relink(name, dst string) {
q.cidx[strings.ToLower(name)] = dst
b, _ := os.ReadFile(dst)
lines, _ := conceptLines(b)
for _, l := range lines {
	root, ok := q.linkRoot(lineKey(l))
	if !ok {continue}
	d, inv := readLink(dst, lineKey(l))
	writeLists(dst, lineKey(l), nil, inv) // обратный — как есть (его уже могли пополнить), прямой — заново
	var pred []format.Value
	for _, n := range d {pred = append(pred, format.Value{Kind: format.Term, Text: n})}
	q.writeLink(name, lineKey(l), pred, root, true)}
if ns := transcl(dst); len(ns) > 0 {q.tidx[strings.ToLower(name)] = ns}
on, of := q.in.OnParam, q.in.OnFrame
q.in.OnParam, q.in.OnFrame = nil, nil
q.loadConceptsIn(dst)
q.in.OnParam, q.in.OnFrame = on, of }
// union: препозиция «union [ Главный Дубль … ]» (решено 09.11.2026; как UnionConcepts ПИ,
// дубли — явным списком): строки дублей вливаются в главный (связи и ссылки — объединение,
// Rank — большее, прочее — если у главного нет), json — по ключам (занятый — «ключ-2»),
// md, html, txt, css — дописываются, прочие спутники — если у главного нет; дубли удаляются,
// ссылки на них во всех концептах и в контенте — на главный. Ответ — имя главного.
func (q *Script) union(arg format.Value) (format.Value, bool) {
if q.theme == "" || arg.Kind != format.Array || len(arg.Items) < 2 {return format.Value{}, false}
var paths []string
for _, v := range arg.Items {
	if v.Kind != format.Term {return format.Value{}, false}
	p := q.cidx[strings.ToLower(v.Text)]
	if p == "" {q.in.RawErr = fmt.Errorf("концепта %s в теме нет", v.Text); return format.Value{}, false}
	if has(paths, p) {q.in.RawErr = fmt.Errorf("%s — дважды", v.Text); return format.Value{}, false}
	paths = append(paths, p)}
main, mname := paths[0], conceptName(paths[0], "")
for _, dp := range paths[1:] {
	dname := conceptName(dp, "")
	b, _ := os.ReadFile(dp)
	lines, _ := conceptLines(b)
	mb, _ := os.ReadFile(main)
	mlines, _ := conceptLines(mb)
	for _, l := range lines {
		if !isParam(l) {if !has(mlines, l) {mlines = append(mlines, l)}; continue}
		k := lineKey(l)
		cur := ""
		for _, m := range mlines {if isParam(m) && format.SameTerm(lineKey(m), k) {cur = m}}
		if cur == "" {mlines = append(mlines, l); continue}
		if _, ok := q.linkRoot(k); ok || k == "Alias" {
			writeLines(main, mlines)
			md, mi := readLink(main, k)
			rd, ri := readLink(dp, k)
			for _, n := range rd {if !has(md, n) {md = append(md, n)}}
			for _, n := range ri {if !has(mi, n) {mi = append(mi, n)}}
			if k == "Alias" {writeParam(main, k, []format.Value{block(md)})} else {writeLists(main, k, md, mi)}
			nb, _ := os.ReadFile(main); mlines, _ = conceptLines(nb); continue}
		if k == "Rank" {a, _ := strconv.Atoi(setting(main, k)); r, _ := strconv.Atoi(setting(dp, k)); if r > a {mlines[indexOf(mlines, cur)] = l}; continue}
		if has(refKeys, k) {
			ts := settingList(main, k)
			for _, n := range settingList(dp, k) {if !has(ts, n) {ts = append(ts, n)}}
			mlines[indexOf(mlines, cur)] = k + " : " + format.Format([]format.Value{block(ts)})}}
	if err := writeLines(main, mlines); err != nil {q.in.RawErr = err; return format.Value{}, false}
	if err := mergeSats(main, dp); err != nil {q.in.RawErr = err; return format.Value{}, false}
	for _, f := range conceptFiles(dp, dp) {os.Remove(f[0])}
	delete(q.cidx, strings.ToLower(dname))
	if err := q.replaceRefs(dname, mname); err != nil {q.in.RawErr = err; return format.Value{}, false}}
selfClean(main, mname)
q.indexConcepts()
q.env = disp.NewEnv(nil)
if q.loadAliases() != nil || q.loadConcepts() != nil {return format.Value{}, false}; return format.Value{Kind: format.Term, Text: mname}, true }
func indexOf(ls []string, s string) int {for i, l := range ls {if l == s {return i}}; return -1 }
// block: блок имён «[ A B ]».
func block(ns []string) format.Value {
v := format.Value{Kind: format.Array}
for _, n := range ns {v.Items = append(v.Items, format.Value{Kind: format.Term, Text: n})}; return v }
// mergeSats: спутники дубля dp → главного main (mergeSatellites ПИ).
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
		if err := os.WriteFile(f[1], []byte(string(a)+sep+string(b)), 0644); err != nil {return err}
	case !exists(f[1]):
		os.MkdirAll(filepath.Dir(f[1]), 0755)
		if err := copyFile(f[0], f[1]); err != nil {return err}}}; return nil }
// selfClean: у главного после слияния — без ссылок на себя и повторов в значениях связей и refKeys (uniqueRefs ПИ).
func selfClean(path, name string) {
b, _ := os.ReadFile(path)
lines, _ := conceptLines(b)
for _, l := range lines {
	if !isParam(l) || !has(refKeys, lineKey(l)) {continue}
	k := lineKey(l)
	uniq := func(ns []string) []string {var o []string; for _, n := range ns {if !format.SameTerm(n, name) && !has(o, n) {o = append(o, n)}}; return o}
	if has([]string{"Taxon", "Links", "Attrib", "Preposit", "Prfix", "Sufix", "Radix"}, k) {d, i := readLink(path, k); writeLists(path, k, uniq(d), uniq(i)); continue}
	writeParam(path, k, []format.Value{block(uniq(settingList(path, k)))})} }
// dedupe: строка параметра — без повторов имён в каждом блоке (после замены имени, union).
func dedupe(l string) string {
k := strings.Index(l, ":")
vs, err := format.Parse(strings.TrimSpace(l[k+1:]))
if err != nil {return l}
for i, v := range vs {
	if v.Kind != format.Array {continue}
	var items []format.Value
	for _, x := range v.Items {
		dup := false
		for _, y := range items {if x.Kind == format.Term && y.Kind == format.Term && format.SameTerm(x.Text, y.Text) {dup = true}}
		if !dup {items = append(items, x)}}
	vs[i].Items = items}; return lineKey(l) + " : " + format.Format(vs) }
