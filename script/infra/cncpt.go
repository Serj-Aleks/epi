// epi/script/infra/cncpt.go
package main
import ("archive/tar"; "compress/gzip"; "encoding/json"; "fmt"; "io"; "log"; "net/http"; "os"; "path/filepath"; "sort"; "strconv"; "strings"; "unicode"; "unicode/utf8"; "epi/script/disp"; "epi/script/format")
// файлы концептов: модель — одна строка, ключей и индексов нет; путь — из букв имени: <тема>/<буквы>/<имя>.uic
//
//	интерпретатор  lexic/preposit/A/R/I/F/M/arifm.uic
//	               Preposit : [ * + - / ]
//	алиас          lexic/preposit/P/L/U/S/plus.uic
//	               plus : +
//	концепт автора thema/<Тема>/X/x.uic
//	               A : 5
// letterPath: <корень>/<путь из букв имени — системный фрейм Sdir>/<имя>.uic; Sdir не ответил — пусто (в журнал)
func (q *Medium) letterPath(root, name string) string {
v, err := q.core.Invoke("Sdir", []format.Value{{Kind: format.String, Text: name}}, q.cenv)
if err != nil || v.Kind != format.String {log.Printf("Sdir %q: %v", name, err); return ""}; return filepath.Join(root, filepath.FromSlash(v.Text), strings.ToLower(name)+".uic") }
// writeLines: файл целиком через временный файл
func writeLines(path string, lines []string) error {
if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {return err}
tmp := path + ".tmp"
if err := os.WriteFile(tmp, []byte(strings.Join(lines, "\n")+"\n"), 0644); err != nil {return err}; return os.Rename(tmp, path) }
// conceptFile: файл концепта темы — по указателю (клон лежит по пути другого концепта), иначе по буквам имени
func (q *Medium) conceptFile(name string) string {
if p, ok := q.cidx[strings.ToLower(name)]; ok {return p}; return q.letterPath(q.themaDir(), name) }
// conceptLines: строки концепта «Параметр : предикация»
func conceptLines(b []byte) ([]string, error) {return disp.Split(string(b)) }
// lineKey: параметр строки «Параметр : …»
func lineKey(line string) string {
k, _, _ := strings.Cut(line, " "); return k }
// writeConcept: параметр фрейма автора строкой в файле концепта (темы или корня lexic/radix) — у системного фрейма (sdict) в концепт темы, повторное определение заменяет строку; синонимы — группа
func (q *Medium) writeConcept(frame, param string, pred []format.Value) error {
if q.theme == "" {return nil}
if root, ok := q.linkRoot(param); ok {return q.writeLink(frame, param, pred, root, false)}
if format.SameTerm(param, "Rank") {return q.writeRank(frame, pred)}
if format.SameTerm(param, "Locus") {return q.writeLocus(frame, pred)}
path := q.conceptPath(frame)
if path == "" || q.inSdict(path) {path = q.conceptFile(frame)}
if !format.SameTerm(param, "Alias") {q.known(path); return writeParam(path, param, pred)}
old, _ := readLink(path, "Alias")
if err := writeParam(path, "Alias", pred); err != nil {return err}
q.known(path); return q.syncAlias(frame, old, names(pred)) }
// writeRank: ранг концепта; пустой ранг (Rank :) — концепт больше не таксон, он убирается из Taxon всех дочерних
func (q *Medium) writeRank(frame string, pred []format.Value) error {
path := q.conceptFile(frame); q.known(path)
was := setting(path, "Rank") != ""
if err := writeParam(path, "Rank", pred); err != nil || !was || setting(path, "Rank") != "" {return err}
_, kids := readLink(path, "Taxon")
root, _ := q.linkRoot("Taxon")
for _, k := range kids {
	kp := q.conceptPath(k); if kp == "" {continue}
	d, _ := readLink(kp, "Taxon")
	var ns []format.Value
	for _, n := range d {if !format.SameTerm(n, frame) {ns = append(ns, format.Value{Kind: format.Term, Text: n})}}
	if err := q.writeLink(conceptName(kp, ""), "Taxon", ns, root, false); err != nil {return err}}; return nil }
// writeParam: строка «Параметр : предикация» в файле концепта path (пустое значение — «Параметр :»); новый параметр — перед телом фрейма; повтор строки параметра убирается
func writeParam(path, param string, pred []format.Value) error {
var lines []string
if b, err := os.ReadFile(path); err == nil {if lines, err = conceptLines(b); err != nil {return fmt.Errorf("%s: %w", path, err)}} else if !os.IsNotExist(err) {return err}
text := strings.TrimRight(param + " : " + strings.ReplaceAll(format.Canon(pred), "\n", " . "), " ")
found := false
out := lines[:0]
for _, l := range lines {if isParam(l) && format.SameTerm(lineKey(l), param) {if found {continue}; l, found = text, true}; out = append(out, l)}
lines = out
if !found {lines = append(lines, "")
	k := len(lines) - 1
	for k > 0 && !isParam(lines[k-1]) {lines[k] = lines[k-1]; k--}
	lines[k] = text}; return writeLines(path, lines) }
// isParam: строка концепта — параметр «Имя : значение» или «Имя :» (имя с прописной, без точки), иначе тело фрейма
func isParam(l string) bool {
f := strings.Fields(l)
if len(f) < 2 || f[1] != ":" || strings.Contains(f[0], ".") {return false}
r, _ := utf8.DecodeRuneInString(f[0]); return unicode.IsUpper(r) }
// writeBody: тело фрейма «Фрейм : [ … ]» — в json концепта ключом Corpus (строка — предикация, порядок — порядок импликаций); в файле концепта — только параметры
func (q *Medium) writeBody(frame string, body []format.Value) error {
if q.theme == "" {return nil}
path := q.conceptFile(frame)
var lines []string
if b, err := os.ReadFile(path); err == nil {if lines, err = conceptLines(b); err != nil {return fmt.Errorf("%s: %w", path, err)}} else if !os.IsNotExist(err) {return err}
var out []string
for _, l := range lines {if isParam(l) {out = append(out, l)}}
preds, err := disp.Split(format.Canon(body))
if err != nil {return err}
if err := writeLines(path, out); err != nil {return err}
if err := writeCorpus(path, preds); err != nil {return err}; q.known(path); return nil }
// readCorpus: тело концепта path из json (ключ Corpus); ok — ключ есть
func readCorpus(path string) ([]string, bool) {
b, err := os.ReadFile(strings.TrimSuffix(path, ".uic") + ".json")
if err != nil {return nil, false}
var m map[string]any
if json.Unmarshal(b, &m) != nil {return nil, false}
l, ok := m[corpusKey].([]any)
if !ok {return nil, false}
var out []string
for _, x := range l {if s, ok := x.(string); ok {out = append(out, s)}}; return out, true }
// writeCorpus: тело в json концепта path ключом Corpus; пустое тело убирает ключ, json без ключей не заводится
func writeCorpus(path string, preds []string) error {
jp := strings.TrimSuffix(path, ".uic") + ".json"
m := map[string]any{}
jb, err := os.ReadFile(jp)
if err == nil {if json.Unmarshal(jb, &m) != nil {return fmt.Errorf("%s: json не читается", jp)}}
if len(preds) == 0 {delete(m, corpusKey)} else {m[corpusKey] = preds}
if len(m) == 0 && err != nil {return nil}
out, _ := json.MarshalIndent(m, "", "  "); return os.WriteFile(jp, out, 0644) }
// bodyLines: тело концепта — Corpus из json, иначе строки файла концепта после параметров
func bodyLines(path string, lines []string) []string {
if c, ok := readCorpus(path); ok {return c}
var body []string
for _, l := range lines {if !isParam(l) {body = append(body, l)}}; return body }
// loadConcepts: системные фреймы (sdict), концепты системной темы (ACTUS) и концепты текущей темы в контекст;
// концепт темы дополняет и переопределяет системный; вопросов при загрузке нет
func (q *Medium) loadConcepts() error {
on, of, ak := q.in.OnParam, q.in.OnFrame, q.in.Ask
q.in.OnParam, q.in.OnFrame, q.in.Ask = nil, nil, nil
defer func() {q.in.OnParam, q.in.OnFrame, q.in.Ask = on, of, ak}()
if err := q.loadConceptsIn(q.in, q.env, filepath.Join(q.dir, "sdict")); err != nil {return err}
if t, ok := q.themeExists(sysTheme); ok && !format.SameTerm(t, q.theme) {if err := q.loadConceptsIn(q.in, q.env, filepath.Join(q.themaRoot(), t)); err != nil {return err}}; return q.loadConceptsIn(q.in, q.env, q.themaDir()) }
// loadConceptsIn: концепты в папке root в контекст env интерпретатора in — имя из пути; у связи в контекст — только прямой список; тело — Corpus json,
// тело строками файла концепта (прежняя запись) один раз переносится в Corpus
func (q *Medium) loadConceptsIn(in *disp.Interp, env *disp.Env, root string) error {return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
	if os.IsNotExist(err) && path == root {return nil}
	if err != nil {return err}
	if d.IsDir() && d.Name() == "prtcl" {return filepath.SkipDir}
	if d.IsDir() || !strings.HasSuffix(path, ".uic") {return nil}
	b, err := os.ReadFile(path)
	if err != nil {return err}
	name := conceptName(path, root)
	lines, err := conceptLines(b)
	if err != nil {return fmt.Errorf("%s: %w", path, err)}
	var params []string
	for _, l := range lines {
		if !isParam(l) {continue}
		params = append(params, l)
		if _, ok := q.linkRoot(lineKey(l)); ok {l = directOnly(l)}
		if _, err := in.Run(name+"."+l, env); err != nil {return fmt.Errorf("%s: %w", path, err)}}
	body := bodyLines(path, lines)
	if _, ok := readCorpus(path); !ok && len(body) > 0 && len(params) < len(lines) {if err := writeCorpus(path, body); err == nil {writeLines(path, params)}}
	if len(body) > 0 {if _, err := in.Run(name+" : [ "+strings.Join(body, " . ")+" ]", env); err != nil {return fmt.Errorf("%s: %w", path, err)}}; return nil}) }
// conceptName: имя концепта из имени файла с прописной первой буквой (К/В/А/Д/Р/А/Т/квадрат.uic → Квадрат)
func conceptName(path, root string) string { return capital(strings.TrimSuffix(filepath.Base(path), ".uic")) }
// known: файл концепта темы в указатель (новый концепт виден сразу, ссылки на него из контента — в его обратный Links), файлы лексикона — нет
func (q *Medium) known(path string) {
if q.theme == "" || !strings.HasPrefix(path, q.themaDir()+string(filepath.Separator)) {return}
k := strings.ToLower(strings.TrimSuffix(filepath.Base(path), ".uic"))
_, was := q.cidx[k]; q.cidx[k] = path
if !was {q.backRefs(path)} }
// backRefs: новый концепт получает в обратный список Links тех, кто уже ссылался на него в контенте ([[Имя]] до его появления)
func (q *Medium) backRefs(path string) {
name := conceptName(path, ""); var add []string
for _, p := range q.cidx {if p == path {continue}; d, _ := readLink(p, "Links"); for _, n := range d {if strings.EqualFold(n, name) {add = append(add, conceptName(p, ""))}}}
if len(add) == 0 {return}
d, i := readLink(path, "Links")
for _, n := range add {if !has(i, n) {i = append(i, n)}}
sort.Strings(i); writeLists(path, "Links", d, i) }
// locus: место концепта темы — папка его файла от папки темы (N/U/M); у концепта лексикона — пусто
func (q *Medium) locus(p string) string {
rel, err := filepath.Rel(q.themaDir(), filepath.Dir(p))
if q.theme == "" || err != nil || strings.HasPrefix(rel, "..") {return ""}; return filepath.ToSlash(rel) }
// writeLocus: Концепт.Locus : путь — концепт со спутниками, img/<имя>.* и картинками контента переезжает в папку темы
// путь — внутри темы, не в prtcl и img, файла с тем же именем там нет; Locus в файл не пишется, связи по имени не меняются
// картинки контента: остались ли в старой папке другие концепты — копируются, нет — переезжают; пустые папки убираются
func (q *Medium) writeLocus(frame string, pred []format.Value) error {
src := q.conceptPath(frame)
if src == "" || q.locus(src) == "" {return fmt.Errorf("Locus: нет концепта темы %s", frame)}
loc := strings.Trim(strings.ReplaceAll(strings.Trim(format.Canon(pred), `" `), " ", ""), "/")
parts := strings.Split(loc, "/")
for _, s := range parts {if s == "" || s == "." || s == ".." || strings.EqualFold(s, "prtcl") || strings.EqualFold(s, "img") {return fmt.Errorf("Locus: путь %q не годится — внутри темы, без prtcl, img и ..", loc)}}
dir := filepath.Join(append([]string{q.themaDir()}, parts...)...)
dst := filepath.Join(dir, filepath.Base(src))
if dst == src {return nil}
if exists(dst) {return fmt.Errorf("Locus: в %s уже есть %s", loc, filepath.Base(src))}
old := filepath.Dir(src)
files := conceptFiles(src, dst)
for _, t := range []string{"md", "html"} {if b, err := os.ReadFile(strings.TrimSuffix(src, ".uic") + "." + t); err == nil {for _, m := range reImg.FindAllStringSubmatch(string(b), -1) {f := m[1] + m[2]; if f == filepath.Base(f) && exists(filepath.Join(old, "img", f)) {files = append(files, [2]string{filepath.Join(old, "img", f), filepath.Join(dir, "img", f)})}}}}
alone := true
if es, err := os.ReadDir(old); err == nil {for _, e := range es {if strings.HasSuffix(e.Name(), ".uic") && e.Name() != filepath.Base(src) {alone = false}}}
for i, f := range files {
	if exists(f[1]) {continue}
	if err := os.MkdirAll(filepath.Dir(f[1]), 0755); err != nil {return err}
	if i > 0 && !alone && strings.Contains(f[0], string(filepath.Separator)+"img"+string(filepath.Separator)) && !strings.HasPrefix(filepath.Base(f[0]), strings.TrimSuffix(filepath.Base(src), ".uic")+".") {if err := copyFile(f[0], f[1]); err != nil {return err}; continue}
	if err := os.Rename(f[0], f[1]); err != nil {return err}}
for d := filepath.Join(old, "img"); ; d = filepath.Dir(d) {if d == q.themaDir() || os.Remove(d) != nil && d != filepath.Join(old, "img") {break}}
q.cidx[strings.ToLower(strings.TrimSuffix(filepath.Base(dst), ".uic"))] = dst; return nil }
// setting: значение настройки концепта (Nord : Links) без кавычек
func setting(path, key string) string {
b, err := os.ReadFile(path)
if err != nil {return ""}
lines, _ := conceptLines(b)
for _, l := range lines {
	if !isParam(l) || !format.SameTerm(lineKey(l), key) {continue}
	vs, err := format.Parse(strings.TrimSpace(l[strings.Index(l, ":")+1:]))
	if err == nil && len(vs) == 1 {return vs[0].Text}}; return "" }
// sdictFile: файл системного фрейма — sdict/<первая буква>/<имя>.uic; нет — пусто
func (q *Medium) sdictFile(name string) string {
r, n := utf8.DecodeRuneInString(name)
if n == 0 {return ""}
if p := filepath.Join(q.dir, "sdict", strings.ToUpper(string(r)), strings.ToLower(name)+".uic"); exists(p) {return p}; return "" }
// inSdict: файл лежит в системной папке sdict (не меняется в script)
func (q *Medium) inSdict(p string) bool { return p != "" && strings.HasPrefix(p, filepath.Join(q.dir, "sdict")+string(filepath.Separator)) }
// conceptPath: файл существующего концепта в теме, в лексиконе (radix) или системный фрейм (sdict); "" — концепта нет
func (q *Medium) conceptPath(name string) string {
if p := q.conceptFile(name); exists(p) {return p}
if p := q.letterPath(filepath.Join(q.dir, "lexic", "radix"), name); exists(p) {return p}; return q.sdictFile(name) }
// iconFile: картинка концепта — <папка>/<имя>.svg или <папка>/img/<имя>.png|jpg|jpeg|webp|gif|ico; системный фрейм — своя svg или общая sdict/sdict.svg
func (q *Medium) iconFile(name string) string {
if name == "" || strings.ContainsAny(name, "/\\") {return ""}
roots := []string{q.themaDir()}
if p := q.sdictFile(name); p != "" {if s := strings.TrimSuffix(p, ".uic") + ".svg"; exists(s) {return s}; if s := filepath.Join(q.dir, "sdict", "sdict.svg"); exists(s) {return s}}
for _, c := range []string{"radix", "preposit"} {roots = append(roots, filepath.Join(q.dir, "lexic", c))}
base := strings.ToLower(name)
for _, r := range roots {
	dir := filepath.Dir(q.letterPath(r, name))
	if r == q.themaDir() {dir = filepath.Dir(q.conceptFile(name))}
	if p := filepath.Join(dir, base+".svg"); exists(p) {return p}
	for _, ext := range []string{"png", "jpg", "jpeg", "webp", "gif", "ico"} {if p := filepath.Join(dir, "img", base+"."+ext); exists(p) {return p}}}; return "" }
// settingList: имена в настройке-блоке концепта (Graph : [ nr ], List : [ Taxon ])
func settingList(path, key string) []string {
b, err := os.ReadFile(path)
if err != nil {return nil}
lines, _ := conceptLines(b)
for _, l := range lines {
	if !isParam(l) || !format.SameTerm(lineKey(l), key) {continue}
	vs, err := format.Parse(strings.TrimSpace(l[strings.Index(l, ":")+1:]))
	if err == nil {return names(vs)}}; return nil }
// exists: файл есть
func exists(path string) bool {_, err := os.Stat(path); return err == nil }
// capital: имя объекта с прописной
func capital(w string) string {r, n := utf8.DecodeRuneInString(w); return string(unicode.ToUpper(r)) + w[n:] }
// sdictList: системные фреймы для карточки темы — имя, картинка, число параметров, спутники; в алфавите темы
func (q *Medium) sdictList() []map[string]string {
out := []map[string]string{}
filepath.WalkDir(filepath.Join(q.dir, "sdict"), func(p string, d os.DirEntry, err error) error {
	if err != nil || d.IsDir() || !strings.HasSuffix(p, ".uic") {return nil}
	name, np, sats := conceptName(p, ""), 0, []string{}
	if b, err := os.ReadFile(p); err == nil {ls, _ := conceptLines(b); for _, l := range ls {if isParam(l) {np++}}}
	for _, t := range satTypes {if exists(strings.TrimSuffix(p, ".uic") + "." + t) {sats = append(sats, t)}}
	r := map[string]string{"name": name, "params": strconv.Itoa(np), "sats": strings.Join(sats, " ")}
	if q.iconFile(name) != "" {r["icon"] = "/api/icon?name=" + name}
	out = append(out, r); return nil})
sort.Slice(out, func(i, j int) bool {return aLess(out[i]["name"], out[j]["name"])}); return out }
// sdictUpdate: sdict из репозитория — архив ветки, из него только script/sdict во временную папку; проверка: загрузка и Sdir;
// не прошла — sdict прежний, ошибка; прошла — папка заменяется целиком, контекст ядра и темы заново; ответ — число фреймов
func (q *Medium) sdictUpdate() (int, error) {
resp, err := http.Get(sdictSource)
if err != nil {return 0, fmt.Errorf("репозиторий недоступен: %v — проверьте сеть", err)}
defer resp.Body.Close()
if resp.StatusCode != http.StatusOK {return 0, fmt.Errorf("репозиторий ответил %s", resp.Status)}
gz, err := gzip.NewReader(resp.Body)
if err != nil {return 0, fmt.Errorf("архив не читается: %v", err)}
dir, tmp := filepath.Join(q.dir, "sdict"), filepath.Join(q.dir, "sdict.new")
os.RemoveAll(tmp)
defer os.RemoveAll(tmp)
tr, n := tar.NewReader(gz), 0
for {
	h, err := tr.Next()
	if err == io.EOF {break}
	if err != nil {return 0, fmt.Errorf("архив не читается: %v", err)}
	_, rel, _ := strings.Cut(h.Name, "/")
	rel, ok := strings.CutPrefix(rel, "script/sdict/")
	if !ok || h.Typeflag != tar.TypeReg || rel == "" || strings.Contains(rel, "..") {continue}
	dst := filepath.Join(tmp, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {return 0, err}
	b, err := io.ReadAll(tr)
	if err != nil {return 0, err}
	if err := os.WriteFile(dst, b, 0644); err != nil {return 0, err}
	if strings.HasSuffix(rel, ".uic") {n++}}
if n == 0 {return 0, fmt.Errorf("в репозитории нет фреймов sdict — sdict не изменён")}
core, cenv := disp.New(limit, nil), disp.NewEnv(nil)
if err := q.loadConceptsIn(core, cenv, tmp); err != nil {return 0, fmt.Errorf("новый sdict не загрузился: %v — sdict не изменён", err)}
if _, err := core.Invoke("Sdir", []format.Value{{Kind: format.String, Text: "A"}}, cenv); err != nil {return 0, fmt.Errorf("в новом sdict не работает Sdir: %v — sdict не изменён", err)}
old := filepath.Join(q.dir, "sdict.old")
os.RemoveAll(old)
if err := os.Rename(dir, old); err != nil && !os.IsNotExist(err) {return 0, err}
if err := os.Rename(tmp, dir); err != nil {os.Rename(old, dir); return 0, err}
os.RemoveAll(old)
q.core, q.cenv, q.env = core, cenv, disp.NewEnv(nil)
if err := q.loadAliases(); err != nil {return n, err}
if q.theme != "" {if err := q.loadConcepts(); err != nil {return n, err}}; return n, nil }
