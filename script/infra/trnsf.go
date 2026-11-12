// epi/script/infra/trnsf.go
package main
import ("fmt"; "os"; "path/filepath"; "strconv"; "strings"; "epi/script/disp"; "epi/script/format")
// трансформации концептов: clone, rename, import, union — файлы, спутники и связи
// clone: clone [ Исходный Новый ] — файл, спутники и img/<имя>.* под новым именем в папку исходного
// прямые списки — те же, у целей клон в обратных, обратные клона пусты; ответ — имя клона
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
// replaceRefs: имя old → name во всех концептах темы и лексикона связей (prfix, sufix, radix) — в значениях refKeys,
// ссылках и трансклюзиях текстовых спутников
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
// rename: rename [ Старое Новое ] — файлы концепта, спутников и img/<имя>.* под новым именем
// в той же папке, имя заменяется во всех связях и в контенте, контекст темы заново; ответ — новое имя
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
// conceptFiles: пары «откуда → куда» для концепта src → dst — файл, спутники, img/<имя>.*
func conceptFiles(src, dst string) [][2]string {
sb, db := strings.TrimSuffix(src, ".uic"), strings.TrimSuffix(dst, ".uic")
files := [][2]string{{src, dst}}
for _, t := range satTypes {if exists(sb + "." + t) {files = append(files, [2]string{sb + "." + t, db + "." + t})}}
sn, dn := filepath.Join(filepath.Dir(src), "img", filepath.Base(sb)), filepath.Join(filepath.Dir(dst), "img", filepath.Base(db))
for _, e := range []string{"png", "jpg", "jpeg", "webp", "gif", "ico"} {if exists(sn + "." + e) {files = append(files, [2]string{sn + "." + e, dn + "." + e})}}; return files }
// importC: import [ Тема Имя … ] — названные концепты, их синонимы, Taxon и Links (прямые
// и обратные) и ещё уровень связей; имеющиеся в теме пропускаются; файлы по тому же пути
// сначала все обратные списки пусты, потом прямые — у целей; ответ — число перенесённых
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
for _, d := range done {clearInverse(d[1])}
for _, d := range done {q.relink(d[0], d[1])}; return format.Value{Kind: format.Integer, Text: strconv.Itoa(len(done))}, true }
// adopt: скопированный концепт name (файл dst) — в указатель и контекст, обратные списки пусты, у целей — в обратных
func (q *Script) adopt(name, dst string) {clearInverse(dst); q.relink(name, dst) }
// clearInverse: обратные списки связей концепта — пусты (на скопированный никто не ссылается)
func clearInverse(dst string) {
b, _ := os.ReadFile(dst)
lines, _ := conceptLines(b)
for _, l := range lines {if _, ok := (&Script{}).linkRoot(lineKey(l)); ok {d, _ := readLink(dst, lineKey(l)); writeLists(dst, lineKey(l), d, nil)}} }
// relink: у целей прямых списков концепт в обратных (свой обратный — как есть, прямой — заново); трансклюзии, контекст
func (q *Script) relink(name, dst string) {
q.cidx[strings.ToLower(name)] = dst
b, _ := os.ReadFile(dst)
lines, _ := conceptLines(b)
for _, l := range lines {
	root, ok := q.linkRoot(lineKey(l))
	if !ok {continue}
	d, inv := readLink(dst, lineKey(l))
	writeLists(dst, lineKey(l), nil, inv)
	var pred []format.Value
	for _, n := range d {pred = append(pred, format.Value{Kind: format.Term, Text: n})}
	q.writeLink(name, lineKey(l), pred, root, true)}
if ns := transcl(dst); len(ns) > 0 {q.tidx[strings.ToLower(name)] = ns}
on, of := q.in.OnParam, q.in.OnFrame
q.in.OnParam, q.in.OnFrame = nil, nil
q.loadConceptsIn(dst)
q.in.OnParam, q.in.OnFrame = on, of }
// union: union [ Главный Дубль … ] — строки дублей в главный: связи — объединение, Rank — большее,
// прочее — если у главного нет; json по ключам (занятый — «ключ-2»), md, html, txt, css дописываются, прочие — если нет
// дубли удаляются, ссылки на них — на главный; ответ — имя главного
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
			if k == "Alias" {writeParam(main, k, []format.Value{listBlock(md)})} else {writeLists(main, k, md, mi)}
			nb, _ := os.ReadFile(main); mlines, _ = conceptLines(nb); continue}
		if k == "Rank" {a, _ := strconv.Atoi(setting(main, k)); r, _ := strconv.Atoi(setting(dp, k)); if r > a {mlines[indexOf(mlines, cur)] = l}; continue}
		if has(refKeys, k) {
			ts := settingList(main, k)
			for _, n := range settingList(dp, k) {if !has(ts, n) {ts = append(ts, n)}}
			mlines[indexOf(mlines, cur)] = k + " : " + format.Format([]format.Value{listBlock(ts)})}}
	if err := writeLines(main, mlines); err != nil {q.in.RawErr = err; return format.Value{}, false}
	if err := mergeSats(main, dp); err != nil {q.in.RawErr = err; return format.Value{}, false}
	for _, f := range conceptFiles(dp, dp) {os.Remove(f[0])}
	delete(q.cidx, strings.ToLower(dname))
	if err := q.replaceRefs(dname, mname); err != nil {q.in.RawErr = err; return format.Value{}, false}}
selfClean(main, mname)
q.indexConcepts()
q.env = disp.NewEnv(nil)
if q.loadAliases() != nil || q.loadConcepts() != nil {return format.Value{}, false}; return format.Value{Kind: format.Term, Text: mname}, true }
// indexOf: место строки s в ls, нет — -1
func indexOf(ls []string, s string) int {for i, l := range ls {if l == s {return i}}; return -1 }
// listBlock: блок имён «[ A B ]»
func listBlock(ns []string) format.Value {
v := format.Value{Kind: format.Array}
for _, n := range ns {v.Items = append(v.Items, format.Value{Kind: format.Term, Text: n})}; return v }
// selfClean: у главного после слияния — без ссылок на себя и повторов в значениях связей и refKeys
func selfClean(path, name string) {
b, _ := os.ReadFile(path)
lines, _ := conceptLines(b)
for _, l := range lines {
	if !isParam(l) || !has(refKeys, lineKey(l)) {continue}
	k := lineKey(l)
	uniq := func(ns []string) []string {var o []string; for _, n := range ns {if !format.SameTerm(n, name) && !has(o, n) {o = append(o, n)}}; return o}
	if has([]string{"Taxon", "Links", "Attrib", "Preposit", "Prfix", "Sufix", "Radix"}, k) {d, i := readLink(path, k); writeLists(path, k, uniq(d), uniq(i)); continue}
	writeParam(path, k, []format.Value{listBlock(uniq(settingList(path, k)))})} }
// dedupe: строка параметра — без повторов имён в каждом блоке (после замены имени, union)
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
