// epi/script/infra/trans.go
package main
import ("os"; "path/filepath"; "strings"; "unicode"; "unicode/utf8"; "epi/script/format")
// Транслятор натуральных предикаций (ason-spec 9.16): «trans "текст"» — токены
// текста классифицируются по словарям; результат — блок предикации нашей
// грамматики. Словари: препозиции — lexic/preposit, шум — lexic/xdict, знаки
// определения — lexic/defnt, атрибуты — lexic/atrbt (пока отбрасываются),
// объекты — концепты темы (формы слова — через Alias). Нераспознанный токен —
// вопрос с выбором; ответ пополняет словарь.
// transTokens: слова (буквы и цифры; дефис внутри слова сохраняется) и тире.
func transTokens(text string) []string {
var ts []string
var w strings.Builder
flush := func() {if w.Len() > 0 {ts = append(ts, strings.ToLower(strings.Trim(w.String(), "-"))); w.Reset()}}
for _, r := range text {
	switch {
	case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' && w.Len() > 0: w.WriteRune(r)
	case r == '—' || r == '–' || r == '-': flush(); ts = append(ts, "—")
	default: flush()}}
flush()
var out []string
for _, t := range ts {if t != "" {out = append(out, t)}}; return out }
// dictFile: слово в словаре класса lexic/<класс> — путь из букв, как у концептов.
func (q *Script) dictFile(class, word string) string { return letterPath(filepath.Join(q.dir, "lexic", class), word) }
func exists(path string) bool {_, err := os.Stat(path); return err == nil }
// objIndex: слово → имя концепта темы (само имя и его синонимы Alias).
func (q *Script) objIndex() map[string]string {
idx := map[string]string{}
root := q.themaDir()
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
// classify: класс слова по словарям; для объекта — имя концепта.
func (q *Script) classify(w string, idx map[string]string) (string, string) {
if w == "—" {return "defnt", ""}
if r, _ := utf8.DecodeRuneInString(w); unicode.IsDigit(r) {return "number", ""}
for _, c := range []string{"xdict", "defnt", "atrbt", "preposit"} {if exists(q.dictFile(c, w)) {return c, ""}}
for _, f := range q.in.Frames {if _, ok := f.Kernel[w]; ok {return "preposit", ""}}
if n, ok := idx[w]; ok {return "thema", n}; return "", "" }
// capital: имя объекта — с прописной (9.9).
func capital(w string) string {r, n := utf8.DecodeRuneInString(w); return string(unicode.ToUpper(r)) + w[n:] }
// learn: ответ на вопрос о слове пополняет словарь. «thema:Имя» — объект: имя
// совпадает со словом — новый концепт темы, иначе слово — синоним (Alias) концепта.
func (q *Script) learn(w, answer string, idx map[string]string) (string, string) {
switch answer {
case "preposit", "xdict", "defnt", "atrbt":
	if err := writeLines(q.dictFile(answer, w), nil); err != nil {return "", ""}; return answer, ""}
name, ok := strings.CutPrefix(answer, "thema:")
if !ok || q.theme == "" {return "", ""}
name = capital(strings.TrimSpace(name))
if checkName(name) != nil {return "", ""}
if strings.EqualFold(name, w) {
	if !exists(q.conceptFile(name)) {if err := writeLines(q.conceptFile(name), nil); err != nil {return "", ""}}
	idx[w] = name; return "thema", name}
var al []format.Value // синонимы концепта name и новое слово
for k, n := range idx {if n == name && !strings.EqualFold(k, name) {al = append(al, format.Value{Kind: format.Term, Text: k})}}
al = append(al, format.Value{Kind: format.Term, Text: w})
if err := q.writeConcept(name, "Alias", []format.Value{{Kind: format.Array, Items: al}}); err != nil {return "", ""}
idx[w] = name; return "thema", name }
// trans: ядро интерпретатора trans — «trans "текст"» → [ предикация ].
func (q *Script) trans(a, b *format.Value) (format.Value, bool) {
if a != nil || b == nil || b.Kind != format.String || q.choose == nil {return format.Value{}, false}
idx := q.objIndex()
opts := []string{"preposit", q.theme, "atrbt", "defnt", "xdict", "✕"}
var out []format.Value
skip := map[string]bool{}
for _, w := range transTokens(b.Text) {
	c, name := q.classify(w, idx)
	if c == "" && !skip[w] {
		ans := q.choose("слово «"+w+"» не распознано: к какому словарю отнести? ("+q.theme+" — объект, концепт темы; ✕ — пропустить)", opts)
		if ans == q.theme {ans = "thema:" + capital(w)}
		if c, name = q.learn(w, ans, idx); c == "" {skip[w] = true}}
	switch c {
	case "thema": out = append(out, format.Value{Kind: format.Term, Text: name})
	case "preposit": out = append(out, format.Value{Kind: format.Term, Text: w})
	case "defnt": out = append(out, format.Value{Kind: format.Op, Text: ":"})
	case "number": if vs, err := format.Parse(w); err == nil && len(vs) == 1 {out = append(out, vs[0])}}}
return format.Value{Kind: format.Array, Items: out}, true }
