// epi/script/infra/trans.go
package main
import ("os"; "path/filepath"; "strings"; "unicode"; "unicode/utf8"; "epi/script/format")
// транслятор trans "текст": слова по словарям — preposit, xdict (шум), defnt (определение), atrbt (атрибуты),
// объекты — концепты темы (формы слова — Alias); результат — блок предикации, нераспознанное — вопрос с выбором
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
// classify: класс слова по словарям; для объекта и атрибута — имя концепта
func (q *Medium) classify(w string, d dicts) (string, string) {
if w == "—" {return "defnt", ""}
if r, _ := utf8.DecodeRuneInString(w); unicode.IsDigit(r) {return "number", ""}
for _, c := range []string{"xdict", "defnt", "preposit"} {if exists(q.dictFile(c, w)) {return c, ""}}
for _, f := range q.in.Frames {if _, ok := f.Kernel[w]; ok {return "preposit", ""}}
if n, ok := d.atr[w]; ok {return "atrbt", n}
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
// learn: ответ пополняет словарь — preposit, xdict, defnt: слово в словарь; thema:Имя — объект; atrbt:Имя — атрибут
func (q *Medium) learn(w, answer string, d dicts) (string, string) {
switch answer {
case "preposit", "xdict", "defnt":
	if err := writeLines(q.dictFile(answer, w), nil); err != nil {return "", ""}; return answer, ""}
if name, ok := strings.CutPrefix(answer, "thema:"); ok && q.theme != "" {
	name = capital(strings.TrimSpace(name))
	if learnName(q.themaDir(), w, name, d.obj) {return "thema", name}}
if name, ok := strings.CutPrefix(answer, "atrbt:"); ok {
	name = capital(strings.TrimSpace(name))
	if learnName(filepath.Join(q.dir, "lexic", "atrbt"), w, name, d.atr) {return "atrbt", name}}; return "", "" }
// trans: ядро интерпретатора trans "текст" → [ предикация ]; атрибут — параметр следующего объекта, иначе предыдущего
// («прямыми углами» → Угол.Прямой)
func (q *Medium) trans(a, b *format.Value) (format.Value, bool) {
if a != nil || b == nil || b.Kind != format.String || q.choose == nil {return format.Value{}, false}
d := dicts{obj: conceptIndex(q.themaDir()), atr: conceptIndex(filepath.Join(q.dir, "lexic", "atrbt"))}
opts := []string{"preposit", q.theme, "atrbt", "defnt", "xdict", "✕"}
var out []format.Value
var attrs []string
lastObj := -1
skip := map[string]bool{}
// end: конец предложения — атрибуты без следующего объекта к предыдущему («Угол — прямой» → Угол.Прямой); разделитель « . »
end := func() {
	if len(attrs) > 0 && lastObj >= 0 {out[lastObj].Text += "." + strings.Join(attrs, "."); out[lastObj].Kind = format.Field}
	if n := len(out); n > 0 && out[n-1].Kind == format.Op && out[n-1].Text == ":" {out = out[:n-1]}
	if n := len(out); n > 0 && !(out[n-1].Kind == format.Op && out[n-1].Text == ".") {out = append(out, format.Value{Kind: format.Op, Text: "."})}
	attrs, lastObj = nil, -1}
for _, w := range transTokens(b.Text) {
	if w == "." {end(); continue}
	c, name := q.classify(w, d)
	if c == "" && !skip[w] {
		ans := q.choose("слово «"+w+"» не распознано: к какому словарю отнести? ("+q.theme+" — объект, концепт темы; atrbt — атрибут; ✕ — пропустить)", opts)
		switch ans {
		case q.theme: ans = "thema:" + capital(w)
		case "atrbt": ans = "atrbt:" + capital(w)}
		if c, name = q.learn(w, ans, d); c == "" {skip[w] = true}}
	switch c {
	case "thema":
		out = append(out, format.Value{Kind: format.Term, Text: strings.Join(append([]string{name}, attrs...), ".")})
		if len(attrs) > 0 {out[len(out)-1].Kind = format.Field}
		attrs, lastObj = nil, len(out)-1
	case "atrbt": attrs = append(attrs, name)
	case "preposit": out = append(out, format.Value{Kind: format.Term, Text: w})
	case "defnt": out = append(out, format.Value{Kind: format.Op, Text: ":"})
	case "number": if vs, err := format.Parse(w); err == nil && len(vs) == 1 {out = append(out, vs[0])}}}
end()
if n := len(out); n > 0 && out[n-1].Kind == format.Op && out[n-1].Text == "." {out = out[:n-1]}
return format.Value{Kind: format.Array, Items: out}, true }
