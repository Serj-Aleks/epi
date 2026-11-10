// epi/script/infra/thema.go
package main
// темы автора: тема — папка thema/<Имя> с протоколом prtcl и концептами (путь из букв имени, строки «Параметр : предикация»)
// активная тема одна, системный лексикон lexic — общий
import ("fmt"; "os"; "path/filepath"; "sort"; "strings"; "unicode"; "unicode/utf8"; "epi/script/disp"; "epi/script/format")
// themaRoot, themaDir: папка тем и папка активной темы
func (q *Script) themaRoot() string { return filepath.Join(q.dir, "thema") }
func (q *Script) themaDir() string  { return filepath.Join(q.themaRoot(), q.theme) }
// themes: имена тем по алфавиту
func (q *Script) themes() ([]string, error) {
es, err := os.ReadDir(q.themaRoot())
if os.IsNotExist(err) {return nil, nil}
if err != nil {return nil, err}
var ts []string
for _, e := range es {if e.IsDir() {ts = append(ts, e.Name())}}
sort.Slice(ts, func(i, j int) bool {return aLess(ts[i], ts[j])}); return ts, nil }
// checkName: имя темы — с прописной буквы, любой алфавит, без пробелов и разделителей пути
func checkName(name string) error {
r, _ := utf8.DecodeRuneInString(name)
switch {
case name == "": return fmt.Errorf("имя темы пусто")
case !unicode.IsUpper(r): return fmt.Errorf("имя темы пишется с прописной буквы (объект)")
case utf8.RuneCountInString(name) > 64: return fmt.Errorf("имя темы длиннее 64 знаков")
case strings.ContainsAny(name, "/\\:*?\"<>|") || strings.IndexFunc(name, unicode.IsSpace) >= 0: return fmt.Errorf("в имени темы — пробел или недопустимый знак")}; return nil }
// themeExists: имя занято (без учёта регистра)
func (q *Script) themeExists(name string) (string, bool) {
ts, _ := q.themes()
for _, t := range ts {if format.SameTerm(t, name) {return t, true}}; return "", false }
// createTheme: новая тема; первая забирает протокол прежнего вида script/prtcl
func (q *Script) createTheme(name string) error {
if err := checkName(name); err != nil {return err}
if t, ok := q.themeExists(name); ok {return fmt.Errorf("тема %s уже есть", t)}
ts, err := q.themes()
if err != nil {return err}
dir := filepath.Join(q.themaRoot(), name)
if err := os.MkdirAll(dir, 0755); err != nil {return err}
legacy := filepath.Join(q.dir, "prtcl")
if _, err := os.Stat(legacy); len(ts) == 0 && err == nil {if err := os.Rename(legacy, filepath.Join(dir, "prtcl")); err != nil {return err}} else if err := os.MkdirAll(filepath.Join(dir, "prtcl"), 0755); err != nil {return err}; return q.switchTheme(name) }
// renameTheme: переименование папки темы; ссылок на имя темы внутри нет
func (q *Script) renameTheme(old, name string) error {
if err := checkName(name); err != nil {return err}
cur, ok := q.themeExists(old)
if !ok {return fmt.Errorf("темы %s нет", old)}
if t, ok := q.themeExists(name); ok && t != cur {return fmt.Errorf("тема %s уже есть", t)}
if err := os.Rename(filepath.Join(q.themaRoot(), cur), filepath.Join(q.themaRoot(), name)); err != nil {return err}
if q.theme == cur {q.theme = name}; return nil }
// switchTheme: тема активна — контекст заново (системные алиасы и концепты темы), база нераспознанных темы, последний открытый сеанс
func (q *Script) switchTheme(name string) error {
q.theme, q.cur, q.closed, q.turns = name, "", false, nil
q.indexConcepts()
q.in.Unknown = nil
if err := q.loadIgnor(); err != nil {return err}
q.env = disp.NewEnv(nil)
if err := q.loadAliases(); err != nil {return err}
if err := q.loadConcepts(); err != nil {return err}
ss, err := q.list()
if err != nil {return err}
for _, s := range ss {if !s.Closed {return q.switchTo(s.Name)}}; q.saveState(); return nil }
// startTheme: при запуске — тема и сеанс из state.uic, иначе тема последнего открытого сеанса, иначе первая; тем нет — ни одной
func (q *Script) startTheme() error {
ts, err := q.themes()
if err != nil || len(ts) == 0 {return err}
if t, s := q.loadState(); t != "" {
	if cur, ok := q.themeExists(t); ok {
		if err := q.switchTheme(cur); err != nil {return err}
		if s != "" && s != q.cur && exists(q.seans(s)) {return q.switchTo(s)}; return nil}}
best, last := ts[0], ""
for _, t := range ts {
	q.theme = t
	ss, err := q.list()
	if err != nil {return err}
	for _, s := range ss {if !s.Closed && s.Name > last {best, last = t, s.Name}}}; return q.switchTheme(best) }
// statePath, saveState, loadState: state.uic — тема и сеанс, где автор остановился
func (q *Script) statePath() string {return filepath.Join(q.dir, "state.uic") }
func (q *Script) saveState() {writeLines(q.statePath(), []string{"Thema : " + q.theme, "Seans : " + q.cur}) }
func (q *Script) loadState() (string, string) {
t, s := "", ""
b, err := os.ReadFile(q.statePath())
if err != nil {return "", ""}
for _, l := range strings.Split(string(b), "\n") {
	k, v, _ := strings.Cut(l, " : ")
	if k == "Thema" {t = strings.TrimSpace(v)} else if k == "Seans" {s = strings.TrimSpace(v)}}; return t, s }
// conceptFile: файл концепта темы — по указателю (клон лежит по пути другого концепта), иначе по буквам имени
func (q *Script) conceptFile(name string) string {
if p, ok := q.cidx[strings.ToLower(name)]; ok {return p}; return letterPath(q.themaDir(), name) }
// indexConcepts: указатель концептов темы
func (q *Script) indexConcepts() {q.cidx = themeIndex(q.themaDir()); q.indexTranscl() }
// themeIndex: указатель концептов темы в папке dir — имя (строчными) → файл
func themeIndex(dir string) map[string]string {
idx := map[string]string{}
filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
	if err != nil {return nil}
	if d.IsDir() && d.Name() == "prtcl" {return filepath.SkipDir}
	if !d.IsDir() && strings.HasSuffix(path, ".uic") {idx[strings.ToLower(strings.TrimSuffix(d.Name(), ".uic"))] = path}; return nil}); return idx }
// conceptLines: строки концепта «Параметр : предикация»
func conceptLines(b []byte) ([]string, error) {return disp.Split(string(b)) }
// lineKey: параметр строки «Параметр : …»
func lineKey(line string) string {
k, _, _ := strings.Cut(line, " "); return k }
// writeConcept: параметр фрейма автора строкой в файле концепта, повторное определение заменяет строку; синонимы — группа
func (q *Script) writeConcept(frame, param string, pred []format.Value) error {
if q.theme == "" {return nil}
if root, ok := q.linkRoot(param); ok {return q.writeLink(frame, param, pred, root, false)}
if !format.SameTerm(param, "Alias") {q.known(q.conceptFile(frame)); return writeParam(q.conceptFile(frame), param, pred)}
path := q.conceptFile(frame)
old, _ := readLink(path, "Alias")
if err := writeParam(path, "Alias", pred); err != nil {return err}
q.cidx[strings.ToLower(frame)] = path; return q.syncAlias(frame, old, names(pred)) }
// writeParam: строка «Параметр : предикация» в файле концепта path; новый параметр — перед телом фрейма
func writeParam(path, param string, pred []format.Value) error {
var lines []string
if b, err := os.ReadFile(path); err == nil {if lines, err = conceptLines(b); err != nil {return fmt.Errorf("%s: %w", path, err)}} else if !os.IsNotExist(err) {return err}
text := param + " : " + strings.ReplaceAll(format.Format(pred), "\n", " . ")
found := false
for i, l := range lines {if isParam(l) && format.SameTerm(lineKey(l), param) {lines[i], found = text, true}}
if !found {lines = append(lines, "")
	k := len(lines) - 1
	for k > 0 && !isParam(lines[k-1]) {lines[k] = lines[k-1]; k--}
	lines[k] = text}; return writeLines(path, lines) }
// isParam: строка концепта — параметр «Имя : значение» (имя с прописной, без точки), иначе тело фрейма
func isParam(l string) bool {
f := strings.Fields(l)
if len(f) < 3 || f[1] != ":" || strings.Contains(f[0], ".") {return false}
r, _ := utf8.DecodeRuneInString(f[0]); return unicode.IsUpper(r) }
// writeBody: тело фрейма «Фрейм : [ … ]» строками после параметров концепта
func (q *Script) writeBody(frame string, body []format.Value) error {
if q.theme == "" {return nil}
path := q.conceptFile(frame)
var lines []string
if b, err := os.ReadFile(path); err == nil {if lines, err = conceptLines(b); err != nil {return fmt.Errorf("%s: %w", path, err)}} else if !os.IsNotExist(err) {return err}
var out []string
for _, l := range lines {if isParam(l) {out = append(out, l)}}
preds, err := disp.Split(format.Format(body))
if err != nil {return err}
q.known(path); return writeLines(path, append(out, preds...)) }
// loadConcepts: системные концепты (lexic/constans) и концепты темы в контекст; концепт темы дополняет и переопределяет системный
func (q *Script) loadConcepts() error {
on, of := q.in.OnParam, q.in.OnFrame
q.in.OnParam, q.in.OnFrame = nil, nil
defer func() {q.in.OnParam, q.in.OnFrame = on, of}()
if err := q.loadConceptsIn(filepath.Join(q.dir, "lexic", "constans")); err != nil {return err}; return q.loadConceptsIn(q.themaDir()) }
// loadConceptsIn: концепты в папке root — имя из пути, файл прежнего вида переписывается; у связи в контекст — только прямой список
func (q *Script) loadConceptsIn(root string) error {return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
	if os.IsNotExist(err) && path == root {return nil}
	if err != nil {return err}
	if d.IsDir() && d.Name() == "prtcl" {return filepath.SkipDir}
	if d.IsDir() || !strings.HasSuffix(path, ".uic") {return nil}
	b, err := os.ReadFile(path)
	if err != nil {return err}
	name := conceptName(path, root)
	var lines []string
	if isLegacy(b) {
		_, models, err := readLegacy(path)
		if err != nil {return err}
		for _, m := range models {lines = append(lines, m[0]+" : "+m[1])}
		if err := writeLines(path, lines); err != nil {return err}} else if lines, err = conceptLines(b); err != nil {return fmt.Errorf("%s: %w", path, err)}
	var body []string
	for _, l := range lines {
		if !isParam(l) {body = append(body, l); continue}
		if _, ok := q.linkRoot(lineKey(l)); ok {l = directOnly(l)}
		if _, err := q.in.Run(name+"."+l, q.env); err != nil {return fmt.Errorf("%s: %w", path, err)}}
	if len(body) > 0 {if _, err := q.in.Run(name+" : [ "+strings.Join(body, " . ")+" ]", q.env); err != nil {return fmt.Errorf("%s: %w", path, err)}}; return nil}) }
// conceptName: имя концепта из имени файла с прописной первой буквой (К/В/А/Д/Р/А/Т/квадрат.uic → Квадрат)
func conceptName(path, root string) string { return capital(strings.TrimSuffix(filepath.Base(path), ".uic")) }
// loadIgnor: база нераспознанных темы — строка на предикацию, нечитаемое — строкой; прежний вид «[ … ]» переписывается
func (q *Script) loadIgnor() error {
b, err := os.ReadFile(q.ignor())
if os.IsNotExist(err) {return nil}
if err != nil {return err}
var lines []string
legacy := false
for _, l := range strings.Split(string(b), "\n") {
	if l = strings.TrimSpace(l); l == "" {continue}
	text, line := l, l
	if vs, err := format.Parse(l); err == nil && len(vs) == 1 {
		switch vs[0].Kind {
		case format.Array:
			text, line, legacy = format.Format(vs[0].Items), format.Format(vs[0].Items), true
		case format.String:
			text = vs[0].Text}}
	q.in.Unknown = append(q.in.Unknown, text)
	lines = append(lines, line)}
if legacy {return writeLines(q.ignor(), lines)}; return nil }
// themeIcon: картинка темы — <тема>/<Тема>.svg или <тема>/img/<Тема>.png|… (как в ПИ)
func (q *Script) themeIcon(name string) string {
dir := filepath.Join(q.themaRoot(), name)
for _, n := range []string{name, strings.ToLower(name)} {
	if p := filepath.Join(dir, n+".svg"); exists(p) {return p}
	for _, e := range []string{"png", "jpg", "jpeg", "webp", "gif", "ico"} {if p := filepath.Join(dir, "img", n+"."+e); exists(p) {return p}}}; return "" }
// themesInfo: темы для карточки темы — имя, картинка, число концептов, спутников и сеансов, последний сеанс
func (q *Script) themesInfo() []map[string]any {
ts, _ := q.themes()
out := []map[string]any{}
for _, t := range ts {
	dir := filepath.Join(q.themaRoot(), t)
	nc, ns, nm, last := 0, 0, 0, ""
	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {return nil}
		if d.IsDir() && d.Name() == "prtcl" {
			es, _ := os.ReadDir(path)
			for _, e := range es {if strings.HasPrefix(e.Name(), "seans-") {nm++; if e.Name() > last {last = e.Name()}}}; return filepath.SkipDir}
		if d.IsDir() || filepath.Dir(path) == dir {return nil}
		if strings.HasSuffix(path, ".uic") {nc++} else {ns++}; return nil})
	r := map[string]any{"name": t, "concepts": nc, "sats": ns, "seans": nm, "last": strings.TrimSuffix(strings.TrimPrefix(last, "seans-"), ".uic")}
	if q.themeIcon(t) != "" {r["icon"] = "/api/themeicon?name=" + t}
	out = append(out, r)}; return out }
// known: файл концепта темы в указатель (новый концепт виден сразу), файлы лексикона — нет
func (q *Script) known(path string) {
if q.theme != "" && strings.HasPrefix(path, q.themaDir()+string(filepath.Separator)) {q.cidx[strings.ToLower(strings.TrimSuffix(filepath.Base(path), ".uic"))] = path} }
