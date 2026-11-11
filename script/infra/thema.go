// epi/script/infra/thema.go
package main
import ("fmt"; "os"; "path/filepath"; "sort"; "strings"; "unicode"; "unicode/utf8"; "epi/script/disp"; "epi/script/format")
// темы автора: тема — папка thema/<Имя> с протоколом prtcl и концептами (путь из букв имени, строки «Параметр : предикация»)
// активная тема одна, системный лексикон lexic — общий
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
// createTheme: новая тема с протоколом prtcl
func (q *Script) createTheme(name string) error {
if err := checkName(name); err != nil {return err}
if t, ok := q.themeExists(name); ok {return fmt.Errorf("тема %s уже есть", t)}
if err := os.MkdirAll(filepath.Join(q.themaRoot(), name, "prtcl"), 0755); err != nil {return err}; return q.switchTheme(name) }
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
// indexConcepts: указатель концептов темы
func (q *Script) indexConcepts() {q.cidx = themeIndex(q.themaDir()); q.indexTranscl() }
// themeIndex: указатель концептов темы в папке dir — имя (строчными) → файл
func themeIndex(dir string) map[string]string {
idx := map[string]string{}
filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
	if err != nil {return nil}
	if d.IsDir() && d.Name() == "prtcl" {return filepath.SkipDir}
	if !d.IsDir() && strings.HasSuffix(path, ".uic") {idx[strings.ToLower(strings.TrimSuffix(d.Name(), ".uic"))] = path}; return nil}); return idx }
// loadIgnor: база нераспознанных темы — строка на предикацию, нечитаемое — строкой
func (q *Script) loadIgnor() error {
b, err := os.ReadFile(q.ignor())
if os.IsNotExist(err) {return nil}
if err != nil {return err}
for _, l := range strings.Split(string(b), "\n") {
	if l = strings.TrimSpace(l); l == "" {continue}
	text := l
	if vs, err := format.Parse(l); err == nil && len(vs) == 1 && vs[0].Kind == format.String {text = vs[0].Text}
	q.in.Unknown = append(q.in.Unknown, text)}; return nil }
// themeIcon: картинка темы — <тема>/<Тема>.svg или <тема>/img/<Тема>.png|…
func (q *Script) themeIcon(name string) string {
dir := filepath.Join(q.themaRoot(), name)
for _, n := range []string{name, strings.ToLower(name)} {
	if p := filepath.Join(dir, n+".svg"); exists(p) {return p}
	for _, e := range []string{"png", "jpg", "jpeg", "webp", "gif", "ico"} {if p := filepath.Join(dir, "img", n+"."+e); exists(p) {return p}}}; return "" }
// themesInfo: темы для карточки темы — имя, картинка, число концептов, таксонов, спутников и сеансов, последний сеанс
func (q *Script) themesInfo() []map[string]any {
ts, _ := q.themes()
out := []map[string]any{}
for _, t := range ts {
	dir := filepath.Join(q.themaRoot(), t)
	nc, nt, ns, nm, last := 0, 0, 0, 0, ""
	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {return nil}
		if d.IsDir() && d.Name() == "prtcl" {
			es, _ := os.ReadDir(path)
			for _, e := range es {if strings.HasPrefix(e.Name(), "seans-") {nm++; if e.Name() > last {last = e.Name()}}}; return filepath.SkipDir}
		if d.IsDir() || filepath.Dir(path) == dir {return nil}
		if strings.HasSuffix(path, ".uic") {nc++; if setting(path, "Rank") != "" {nt++}} else {ns++}; return nil})
	r := map[string]any{"name": t, "concepts": nc, "taxa": nt, "sats": ns, "seans": nm, "last": strings.TrimSuffix(strings.TrimPrefix(last, "seans-"), ".uic")}
	if q.themeIcon(t) != "" {r["icon"] = "/api/themeicon?name=" + t}
	out = append(out, r)}; return out }
