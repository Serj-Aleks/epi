// epi/script/infra/trans.go
package main
import ("encoding/json"; "fmt"; "os"; "path/filepath"; "strconv"; "strings"; "unicode"; "unicode/utf8"; "epi/script/format")
// транслятор trans "текст": слова по словарям — preposit, xdict (шум), defnt (определение), numer (числительные: слово : число),
// объекты — концепты темы (формы слова — Alias); результат — блок предикации; нераспознанное в одном предложении — вопрос с выбором,
// в нескольких — отметка ?слово в результате и запись в словарь нераспознанных incog темы
// transTokens: слова (буквы, цифры, дефис внутри слова), тире и конец предложения
func transTokens(text string) []string {
var ts []string
var w strings.Builder
flush := func() {if w.Len() > 0 {ts = append(ts, strings.ToLower(strings.Trim(w.String(), "-"))); w.Reset()}}
for _, r := range text {
	switch {
	case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' && w.Len() > 0: w.WriteRune(r)
	case r == '—' || r == '–' || r == '-': flush(); ts = append(ts, "—")
	case r == '.' || r == '!' || r == '?' || r == ';': flush(); ts = append(ts, ".")
	default: flush()}}
flush()
var out []string
for _, t := range ts {if t != "" {out = append(out, t)}}; return out }
// transSentences: предложения текста — те же концы, что у transTokens
func transSentences(text string) []string {
var out []string
for _, s := range strings.FieldsFunc(text, func(r rune) bool {return r == '.' || r == '!' || r == '?' || r == ';'}) {if s = strings.TrimSpace(s); s != "" {out = append(out, s)}}; return out }
// incog: нераспознанное слово w в словаре prtcl/incog.uic темы — «Слово : [ счётчик "первое предложение" ]»
func (q *Medium) incog(w, sentence string) error {
path := filepath.Join(q.themaDir(), "prtcl", "incog.uic")
n, s := 0, sentence
if b, err := os.ReadFile(path); err == nil {
	ls, _ := conceptLines(b)
	for _, l := range ls {if isParam(l) && format.SameTerm(lineKey(l), capital(w)) {if vs, err := format.Parse(strings.TrimSpace(l[strings.Index(l, ":")+1:])); err == nil && len(vs) == 1 && len(vs[0].Items) == 2 {n, _ = strconv.Atoi(vs[0].Items[0].Text); s = vs[0].Items[1].Text}}}}
return writeParam(path, capital(w), []format.Value{{Kind: format.Array, Items: []format.Value{{Kind: format.Integer, Text: strconv.Itoa(n + 1)}, {Kind: format.String, Text: s}}}}) }
// dictFile: слово в словаре класса lexic/<класс> — путь из букв, как у концептов
func (q *Medium) dictFile(class, word string) string { return letterPath(filepath.Join(q.dir, "lexic", class), word) }
// conceptIndex: слово → имя концепта в папке root (имя — из файла, синонимы — Alias)
func conceptIndex(root string) map[string]string {
idx := map[string]string{}
filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
	if err != nil {return nil}
	if d.IsDir() && d.Name() == "prtcl" {return filepath.SkipDir}
	if d.IsDir() || !strings.HasSuffix(path, ".uic") {return nil}
	name := conceptName(path, root)
	idx[strings.ToLower(name)] = name
	b, err := os.ReadFile(path)
	if err != nil {return nil}
	for _, l := range strings.Split(string(b), "\n") {
		if !isParam(l) || !format.SameTerm(lineKey(l), "Alias") {continue}
		vs, err := format.Parse(strings.TrimSpace(l[strings.Index(l, ":")+1:]))
		if err != nil {continue}
		for _, v := range vs {
			if v.Kind == format.Array {for _, it := range v.Items {idx[strings.ToLower(it.Text)] = name}} else {idx[strings.ToLower(v.Text)] = name}}}; return nil}); return idx }
// classify: класс слова по словарям; для объекта — имя концепта, для числительного — число
func (q *Medium) classify(w string, d dicts) (string, string) {
if w == "—" {return "defnt", ""}
if r, _ := utf8.DecodeRuneInString(w); unicode.IsDigit(r) {return "number", ""}
for _, c := range []string{"xdict", "defnt", "preposit"} {if exists(q.dictFile(c, w)) {return c, ""}}
for _, f := range q.in.Frames {if _, ok := f.Kernel[w]; ok {return "preposit", ""}}
if p := q.dictFile("numer", w); exists(p) {if l, err := os.ReadFile(p); err == nil {if _, n, ok := strings.Cut(string(l), ":"); ok {return "numer", strings.TrimSpace(n)}}}
if n, ok := d.obj[w]; ok {return "thema", n}; return "", "" }
// learnName: слово w — концепт name в папке root: совпало с именем — новый концепт, иначе синоним (Alias)
func learnName(root, w, name string, idx map[string]string) bool {
if checkName(name) != nil {return false}
path := letterPath(root, name)
if strings.EqualFold(name, w) {
	if !exists(path) {if err := writeLines(path, nil); err != nil {return false}}
	idx[w] = name; return true}
var al []format.Value
for k, n := range idx {if n == name && !strings.EqualFold(k, name) {al = append(al, format.Value{Kind: format.Term, Text: k})}}
al = append(al, format.Value{Kind: format.Term, Text: w})
if err := writeParam(path, "Alias", []format.Value{{Kind: format.Array, Items: al}}); err != nil {return false}
idx[w], idx[strings.ToLower(name)] = name, name; return true }
// learn: ответ пополняет словарь — preposit, xdict, defnt: слово в словарь; preposit:п — синоним препозиции п (слово : п),
// само слово — новая препозиция, тире — знак определения (defnt); thema:Имя — объект; numer:N — числительное (слово : N)
func (q *Medium) learn(w, answer string, d dicts) (string, string) {
switch answer {
case "preposit", "xdict", "defnt":
	if err := writeLines(q.dictFile(answer, w), nil); err != nil {return "", ""}; return answer, ""}
if t, ok := strings.CutPrefix(answer, "preposit:"); ok {
	switch t = strings.TrimSpace(t); {
	case t == "" || strings.ContainsAny(t, " \t"): return "", ""
	case strings.EqualFold(t, w): return q.learn(w, "preposit", d)
	case t == "—" || t == "–" || t == "-": return q.learn(w, "defnt", d)}
	if err := writeLines(q.dictFile("preposit", w), []string{w + " : " + t}); err != nil {return "", ""}; return "preposit", ""}
if name, ok := strings.CutPrefix(answer, "thema:"); ok && q.theme != "" {
	name = capital(strings.TrimSpace(name))
	if learnName(q.themaDir(), w, name, d.obj) {q.indexConcepts(); return "thema", name}}
if n, ok := strings.CutPrefix(answer, "numer:"); ok {
	n = strings.TrimSpace(n)
	if vs, err := format.Parse(n); err != nil || len(vs) != 1 || vs[0].Kind != format.Integer && vs[0].Kind != format.Decimal {return "", ""}
	if err := writeLines(q.dictFile("numer", w), []string{w + " : " + n}); err != nil {return "", ""}; return "numer", n}; return "", "" }
// trans: ядро интерпретатора trans "текст" → [ предикация ]; атрибут — параметр следующего объекта, иначе предыдущего;
// объект перед объектом без препозиции — атрибут второго («прямыми углами» → Угол.Прямой); числительное — число перед объектом
func (q *Medium) trans(a, b *format.Value) (format.Value, bool) {
if a != nil || b == nil || b.Kind != format.String || q.choose == nil {return format.Value{}, false}
d := dicts{obj: conceptIndex(q.themaDir())}
opts := []string{"preposit", q.theme, "numer", "xdict", "✕"}
var out []format.Value
var attrs []string
lastObj, seg := -1, 0
skip := map[string]bool{}
sents := transSentences(b.Text)
ask, si, dot := len(sents) < 2, 0, false
// end: конец предложения — атрибуты без следующего объекта к предыдущему («Угол — прямой» → Угол.Прямой); разделитель « . »
end := func() {
	if len(attrs) > 0 && lastObj >= 0 {out[lastObj].Text += "." + strings.Join(attrs, "."); out[lastObj].Kind = format.Field}
	if n := len(out); n > 0 && out[n-1].Kind == format.Op && out[n-1].Text == ":" {out = out[:n-1]}
	if len(out) > seg {q.sense(out[seg:])}
	if n := len(out); n > 0 && !(out[n-1].Kind == format.Op && out[n-1].Text == ".") {out = append(out, format.Value{Kind: format.Op, Text: "."})}
	attrs, lastObj, seg = nil, -1, len(out)}
for _, w := range transTokens(b.Text) {
	if w == "." {end(); if !dot {si++}; dot = true; continue}
	dot = false
	c, name := q.classify(w, d)
	if c == "" && !ask {
		sent := ""
		if si < len(sents) {sent = sents[si]}
		if q.theme != "" {q.incog(w, sent)}
		out = append(out, format.Value{Kind: format.Term, Text: "?" + w}); continue}
	if c == "" && !skip[w] {
		ans := q.choose("слово «"+w+"» не распознано: к какому словарю отнести? ("+q.theme+" — объект, концепт темы; preposit — препозиция или синоним; numer — числительное; xdict — шум; ✕ — пропустить)", opts)
		switch ans {
		case q.theme: ans = "thema:" + capital(w)}
		if c, name = q.learn(w, ans, d); c == "" {skip[w] = true}}
	switch c {
	case "thema":
		if lastObj >= 0 && lastObj == len(out)-1 {attrs = append(strings.Split(out[lastObj].Text, "."), attrs...); out = out[:lastObj]}
		out = append(out, format.Value{Kind: format.Term, Text: strings.Join(append([]string{name}, attrs...), ".")})
		if len(attrs) > 0 {out[len(out)-1].Kind = format.Field}
		attrs, lastObj = nil, len(out)-1
	case "numer": if vs, err := format.Parse(name); err == nil && len(vs) == 1 {out = append(out, vs[0])}
	case "preposit": out = append(out, format.Value{Kind: format.Term, Text: w})
	case "defnt": out = append(out, format.Value{Kind: format.Op, Text: ":"})
	case "number": if vs, err := format.Parse(w); err == nil && len(vs) == 1 {out = append(out, vs[0])}}}
end()
if n := len(out); n > 0 && out[n-1].Kind == format.Op && out[n-1].Text == "." {out = out[:n-1]}
return format.Value{Kind: format.Array, Items: out}, true }
// sense: смысл распознанного предложения — вопросы ✓/✕: атрибут (Объект.Атрибут → связь Attrib), определение
// (Объект : … → json объекта), ситуация (Объект препозиция Объект → json левого объекта); с ?словом — без вопросов
func (q *Medium) sense(seg []format.Value) {
obj := func(v format.Value) bool {return (v.Kind == format.Term || v.Kind == format.Field) && !format.IsPrepTerm(v.Text)}
for _, v := range seg {if strings.HasPrefix(v.Text, "?") {return}}
for _, v := range seg {
	if v.Kind != format.Field || !obj(v) {continue}
	ps := strings.Split(v.Text, ".")
	old, _ := readLink(q.conceptFile(ps[0]), "Attrib")
	var add []string
	for _, a := range ps[1:] {if !has(old, a) && !has(add, a) {add = append(add, a)}}
	if len(add) == 0 || !q.in.Ask("записать атрибут: "+ps[0]+".Attrib : [ "+strings.Join(append(old, add...), " ")+" ] ?") {continue}
	var vs []format.Value
	for _, a := range append(old, add...) {vs = append(vs, format.Value{Kind: format.Term, Text: a})}
	root, _ := q.linkRoot("Attrib")
	if q.writeLink(ps[0], "Attrib", []format.Value{{Kind: format.Array, Items: vs}}, root, false) == nil {q.indexConcepts()}}
if len(seg) > 2 && obj(seg[0]) && seg[1].Kind == format.Op && seg[1].Text == ":" {
	x, r := strings.Split(seg[0].Text, ".")[0], format.Canon(seg[2:])
	if q.in.Ask("записать определение: "+x+" : "+r+" — в json "+x+" ?") {q.define(x, r)}; return}
if len(seg) == 3 && obj(seg[0]) && obj(seg[2]) && seg[1].Kind == format.Term && format.IsPrepTerm(seg[1].Text) {
	x := strings.Split(seg[0].Text, ".")[0]
	if q.in.Ask("записать ситуацию: "+seg[0].Text+" "+seg[1].Text+" "+seg[2].Text+" — в json "+x+" ?") {q.situation(x, seg[0].Text, seg[1].Text, seg[2].Text)}} }
// define: определение r термина name в json концепта name — «Термин : [ правые части ]»; строка-значение становится списком
func (q *Medium) define(name, r string) error {
cp := q.cidx[strings.ToLower(name)]
if cp == "" {return fmt.Errorf("концепта %s в теме нет", name)}
jp := strings.TrimSuffix(cp, ".uic") + ".json"
m := map[string]any{}
if jb, err := os.ReadFile(jp); err == nil {if json.Unmarshal(jb, &m) != nil {return fmt.Errorf("%s: json не читается", jp)}}
var list []any
switch v := m[name].(type) {
case string: list = []any{v}
case []any: list = v}
for _, x := range list {if x == r {return nil}}
m[name] = append(list, r)
out, _ := json.MarshalIndent(m, "", "  "); return os.WriteFile(jp, out, 0644) }
