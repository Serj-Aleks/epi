// script — интерфейс среды (ason-spec 0.3; до 02.11.2026 — Qwest): строка
// взаимодействия с диспетчером предикаций в браузере
// (claude/ason-spec.md, раздел 9.11). Предикация вводится в строке, ответ
// диспетчера — следующей строкой диалога.
//
// Данные — рядом с кодом, в папке script (ason-spec 0.20): lexic — системный
// лексикон (класс preposit: алиасы plus, minus, фреймы-интерпретаторы arifm,
// logic, мета-файлы .uic); thema — темы автора (9.14): в каждой концепты и
// протокол prtcl: ignor.uic —
// нераспознанные предикации, сеансы seans-<дата открытия>.uic, строки —
// модели «[ предикация ] -> интерпретация». Сеанс открывается и закрывается
// автором (кнопками); открытый можно продолжать, любой — читать. Закрытый
// сеанс — файл только для чтения.
//
// Запуск из папки script: go run ./infra [-addr localhost:1944]
// epi/script/infra/main.go
package main
import ("encoding/json"; "flag"; "fmt"; "log"; "net/http"; "os"; "path/filepath"; "sort"; "strings"; "time"; "epi/script/disp"; "epi/script/format")
func open(dir string) (*Script, error) {
q := &Script{dir: dir}
q.in = disp.New(limit, func(string) bool { return false })
// фреймы-интерпретаторы: lexic/preposit (9.13) — только системная часть
for _, f := range q.in.Frames {if err := q.syncFrame(f); err != nil {return nil, err}}
q.in.Frame("trans").Kernel["trans"] = q.trans // транслятор (9.16): ядро — словари среды
q.in.Raw = map[string]func(format.Value) (format.Value, bool){"nexus": q.nexus, "qwest": q.qwest, "clone": q.clone, "rename": q.rename, "qwizt": q.qwizt} // локальный контекст (9.17), поиск (9.20), клон и переименование (9.19, 9.21)
// фреймы автора сохраняются в активной теме (9.14)
q.in.OnParam = func(frame, param string, pred []format.Value) {if err := q.writeConcept(frame, param, pred); err != nil {log.Print(err)}}
q.in.OnFrame = func(name string, body []format.Value) {if err := q.writeBody(name, body); err != nil {log.Print(err)}}
// алиасы препозиций — системный лексикон, общий для всех тем (9.15)
q.in.OnAlias = func(name, target string) {
	if q.in.Frame(name) != nil {return} // имя интерпретатора — не алиас
	if err := writeLines(q.frameFile("preposit", name), []string{name + " : " + target}); err != nil {log.Print(err)}}
// контекст — у темы (9.14): системные алиасы и концепты темы
q.env = disp.NewEnv(nil)
if err := q.loadAliases(); err != nil {return nil, err}
if err := q.startTheme(); err != nil {return nil, err}; return q, nil }
func (q *Script) ignor() string            { return filepath.Join(q.themaDir(), "prtcl", "ignor.uic") }
func (q *Script) seans(name string) string { return filepath.Join(q.themaDir(), "prtcl", name) }
// list: сеансы протокола, новые первыми; закрытый — файл только для чтения.
func (q *Script) list() ([]Seans, error) {
if q.theme == "" {return nil, nil}
es, err := os.ReadDir(filepath.Join(q.themaDir(), "prtcl"))
if os.IsNotExist(err) {return nil, nil}
if err != nil {return nil, err}
var ss []Seans
for _, e := range es {
	n := e.Name()
	if !strings.HasPrefix(n, "seans-") || !strings.HasSuffix(n, ".uic") {continue}
	st, err := e.Info()
	if err != nil {return nil, err}
	ss = append(ss, Seans{Name: n, Closed: st.Mode().Perm()&0200 == 0})}
sort.Slice(ss, func(i, j int) bool { return ss[i].Name > ss[j].Name }); return ss, nil }
// switchTo: сделать сеанс текущим. Контекст сеанса восстанавливается:
// его предикации интерпретируются заново без вопросов.
func (q *Script) switchTo(name string) error {
st, err := os.Stat(q.seans(name))
if err != nil {return err}
b, err := os.ReadFile(q.seans(name))
if err != nil {return err}
text, errLines := bangs(string(b)) // ошибка — «-> !» (09.11.2026)
vs, err := format.Parse(text)
if err != nil {return fmt.Errorf("%s: %w", name, err)}
if len(vs) > 0 && !hasArrow(vs) { // сеанс прежнего вида — только предикации
	if vs, err = q.migrateSeans(name, vs, st.Mode().Perm()); err != nil {return err}}
// сеанс заново не выполняется (9.13.4а): строки — модели
// «[ предикация ] -> интерпретация», диалог показывается, каким был
var turns []Turn
for i := 0; i+2 < len(vs); i += 3 {
	if vs[i+1].Kind != format.Op || vs[i+1].Text != "->" {return fmt.Errorf("%s: строка %d: ожидалась модель «[ … ] -> …»", name, vs[i].Line)}
	t := Turn{Pred: vs[i].Text, Answer: vs[i+2].String()}
	if vs[i].Kind == format.Array {t.Pred = format.Format(vs[i].Items)}
	if vs[i+2].Kind == format.String {t.Answer = vs[i+2].Text} // интерпретация — строка (в том числе сообщение об ошибке)
	t.Error = errLines[vs[i+2].Line]
	turns = append(turns, t)}
q.cur, q.closed, q.turns = name, st.Mode().Perm()&0200 == 0, turns; q.saveState(); return nil }
// newSeans: открыть сеанс seans-<сегодня>.uic. Новый сеанс — на следующий
// день; в тот же день сеанс продолжается.
func (q *Script) newSeans() error {
name := "seans-" + time.Now().Format("2006-01-02") + ".uic"
if st, err := os.Stat(q.seans(name)); err == nil {if st.Mode().Perm()&0200 == 0 {return fmt.Errorf("сеанс %s сегодня уже закрыт; его можно открыть для чтения", name)}} else if err := os.WriteFile(q.seans(name), nil, 0644); err != nil {return err}; return q.switchTo(name) }
// record: шаг диалога как строка сеанса — модель «[ предикация ] -> интерпретация»;
// нечитаемая предикация и сообщение об ошибке — строками.
func record(t Turn) string {
pred := "[ " + t.Pred + " ]"
if _, err := format.Parse(t.Pred); err != nil {pred = format.Value{Kind: format.String, Text: t.Pred}.String()}
ans := t.Answer
if vs, err := format.Parse(ans); t.Error || err != nil || len(vs) != 1 {ans = format.Value{Kind: format.String, Text: t.Answer}.String()}
if t.Error {ans = "! " + ans} // ошибка — знак «!» перед ответом (09.11.2026)
return pred + " -> " + ans + "\n" }
// bangs: строки сеанса с ошибкой «-> !» — номера строк; знак убирается перед разбором
// (в ASON «!» не токен, это отметка записи сеанса).
func bangs(s string) (string, map[int]bool) {
m := map[int]bool{}
ls := strings.Split(s, "\n")
for i, l := range ls {if j := strings.LastIndex(l, "-> ! "); j >= 0 {ls[i], m[i+1] = l[:j+3]+l[j+5:], true}}; return strings.Join(ls, "\n"), m }
func appendFile(path, text string) error {
f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
if err != nil {return err}
if _, err := f.WriteString(text); err != nil {f.Close(); return err}; return f.Close() }
// start: интерпретация в отдельной горутине; вопросы диспетчера (0.7, 0.16)
// уходят в браузер, ответы возвращаются.
func (q *Script) start(text string) *run {
r := &run{questions: make(chan ask), answers: make(chan string), done: make(chan []Turn, 1)}
known := len(q.in.Unknown)
q.in.Ask = func(question string) bool {r.questions <- ask{Text: question}; return <-r.answers == "y"}
q.choose = func(text string, opts []string) string {r.questions <- ask{Text: text, Options: opts}; return <-r.answers}
go func() {
	// на каждую предикацию строки — свой ответ (9.11.8)
	preds, err := disp.Split(text)
	var ts []Turn
	if err != nil { // не прочитано — как нераспознанное (0.7): вопрос, не ошибка
		q.in.Unreadable(text, err)
		ts = append(ts, Turn{Pred: text, Answer: "@none"})}
	for _, p := range preds {
		v, err := q.in.Run(p, q.env)
		t := Turn{Pred: p, Answer: v.String(), Frames: append([]string(nil), q.in.Used...)}
		if err != nil {t.Answer, t.Error = err.Error(), true}
		ts = append(ts, t)}
	for _, u := range q.in.Unknown[known:] {
		line := u                                  // ignor.uic: предикация как есть, строкой файла
		if _, err := format.Parse(u); err != nil {line = format.Value{Kind: format.String, Text: u}.String()} // нечитаемое — строкой
		if e := appendFile(q.ignor(), line+"\n"); e != nil {log.Print(e)}}
	r.done <- ts}(); return r }
// wait: следующий вопрос или результат.
func (q *Script) wait(r *run) Reply {
select {
case a := <-r.questions:
	q.pending, q.pQuest, q.pOpts = r, a.Text, a.Options; return Reply{Question: a.Text, Options: a.Options}
case ts := <-r.done:
	q.pending, q.pPred, q.pQuest, q.pOpts = nil, "", "", nil
	for _, t := range ts {
		q.turns = append(q.turns, t)
		if err := appendFile(q.seans(q.cur), record(t)); err != nil {log.Print(err)}}; return Reply{Turns: ts}} }
func (q *Script) state() map[string]any {
ss, err := q.list()
if err != nil {log.Print(err)}
ts, err := q.themes()
if err != nil {log.Print(err)}; return map[string]any{
	"themes": ts, "theme": q.theme,
	"seanses": ss, "current": q.cur, "closed": q.closed, "turns": q.turns,
	// ожидающий вопрос отдаётся тоже: страницу можно перезагрузить, не потеряв его
	"pending": map[string]any{"pred": q.pPred, "question": q.pQuest, "options": q.pOpts},} }
func (q *Script) handle(w http.ResponseWriter, req *http.Request) {
w.Header().Set("Content-Type", "application/json")
q.mu.Lock()
defer q.mu.Unlock()
var body struct {
	Text   string `json:"text"`
	To     string `json:"to"`
	Answer bool   `json:"answer"`
	Choice string `json:"choice"` // ответ на вопрос с выбором
	Name   string `json:"name"`
	Ext    string `json:"ext"`} // тип спутника (md, json …)
if req.Method != http.MethodGet {if err := json.NewDecoder(req.Body).Decode(&body); err != nil {http.Error(w, err.Error(), http.StatusBadRequest); return}}
fail := func(code int, msg string) { http.Error(w, msg, code) }
if req.URL.Path == "/api/nexus" {json.NewEncoder(w).Encode(q.nexusGraph(req.URL.Query().Get("name"))); return} // граф локального контекста (9.17)
if req.URL.Path == "/api/card" {json.NewEncoder(w).Encode(q.card(req.URL.Query().Get("name"))); return} // карточка концепта (панель, 9.19)
if req.URL.Path == "/api/qwest" {json.NewEncoder(w).Encode(q.qwestFind(req.URL.Query().Get("q"))); return} // поиск: список для панели (9.20)
if req.URL.Path == "/api/qwizt" {json.NewEncoder(w).Encode(q.qwizFind(req.URL.Query().Get("q"))); return} // поиск во всех темах
if req.URL.Path == "/api/tags" {json.NewEncoder(w).Encode(q.tags()); return} // теги темы (линза тегов, 9.21)
if req.URL.Path == "/api/themes" {json.NewEncoder(w).Encode(q.themesInfo()); return} // темы со счётчиками (карточка темы)
if req.URL.Path == "/api/themeicon" {n := req.URL.Query().Get("name"); if _, ok := q.themeExists(n); !ok || q.themeIcon(n) == "" {http.NotFound(w, req); return}; w.Header().Del("Content-Type"); http.ServeFile(w, req, q.themeIcon(n)); return}
if req.URL.Path == "/api/process" {json.NewEncoder(w).Encode(q.process(req.URL.Query().Get("name"))); return} // ситуации процесса (линза процесса, 9.21)
if req.URL.Path == "/api/children" {json.NewEncoder(w).Encode(q.children(req.URL.Query().Get("name"))); return} // дочерние (обратный Taxon)
if req.URL.Path == "/api/sat" { // спутник концепта: прочитать (GET) или записать (POST)
	if req.Method == http.MethodGet {
		p := q.satFile(req.URL.Query().Get("name"), req.URL.Query().Get("ext"))
		b, err := os.ReadFile(p)
		if p == "" || err != nil && !os.IsNotExist(err) {fail(http.StatusNotFound, "нет спутника"); return}
		json.NewEncoder(w).Encode(map[string]string{"text": string(b)}); return}
	if err := q.writeSat(body.Name, body.Ext, body.Text); err != nil {fail(http.StatusBadRequest, err.Error()); return}
	json.NewEncoder(w).Encode(q.card(body.Name)); return}
if req.URL.Path == "/api/img" { // картинка контента: <папка концепта>/img/<файл> (как в ПИ)
	p, f := q.conceptPath(req.URL.Query().Get("name")), req.URL.Query().Get("f")
	if p == "" || f == "" || f != filepath.Base(f) {http.NotFound(w, req); return}
	w.Header().Del("Content-Type"); http.ServeFile(w, req, filepath.Join(filepath.Dir(p), "img", f)); return}
if req.URL.Path == "/api/icon" { // картинка узла графа (спутник концепта)
	p := q.iconFile(req.URL.Query().Get("name"))
	if t := req.URL.Query().Get("theme"); t != "" && t != q.theme {p = q.iconOther(t, req.URL.Query().Get("name"))} // концепт другой темы (qwizt)
	if p == "" {http.NotFound(w, req); return}
	w.Header().Del("Content-Type"); http.ServeFile(w, req, p); return}
if q.pending != nil && req.URL.Path != "/api/answer" && req.URL.Path != "/api/state" {fail(http.StatusConflict, "ждёт ответа на вопрос"); return}
switch req.URL.Path {
case "/api/state":
	json.NewEncoder(w).Encode(q.state())
case "/api/thema/new": // новая тема (9.14)
	if err := q.createTheme(strings.TrimSpace(body.Name)); err != nil {fail(http.StatusBadRequest, err.Error()); return}
	if q.cur == "" { // в новой теме сразу открывается сеанс
		if err := q.newSeans(); err != nil {fail(http.StatusConflict, err.Error()); return}}
	json.NewEncoder(w).Encode(q.state())
case "/api/thema/open":
	t, ok := q.themeExists(body.Name)
	if !ok {fail(http.StatusNotFound, "нет такой темы"); return}
	if err := q.switchTheme(t); err != nil {fail(http.StatusInternalServerError, err.Error()); return}
	json.NewEncoder(w).Encode(q.state())
case "/api/thema/rename":
	if err := q.renameTheme(body.Name, strings.TrimSpace(body.To)); err != nil {fail(http.StatusBadRequest, err.Error()); return}
	json.NewEncoder(w).Encode(q.state())
case "/api/new": // открыть новый сеанс: seans-<сегодня>.uic
	if q.theme == "" {fail(http.StatusConflict, "темы нет: сначала создайте тему"); return}
	if err := q.newSeans(); err != nil {fail(http.StatusConflict, err.Error()); return}
	json.NewEncoder(w).Encode(q.state())
case "/api/open": // открыть сеанс: продолжить (не закрыт) или читать (закрыт)
	if filepath.Base(body.Name) != body.Name || !strings.HasPrefix(body.Name, "seans-") {fail(http.StatusBadRequest, "нет такого сеанса"); return}
	if err := q.switchTo(body.Name); err != nil {fail(http.StatusNotFound, err.Error()); return}
	json.NewEncoder(w).Encode(q.state())
case "/api/close": // закрыть текущий сеанс: файл только для чтения
	if q.cur == "" || q.closed {fail(http.StatusConflict, "открытого сеанса нет"); return}
	if err := os.Chmod(q.seans(q.cur), 0444); err != nil {fail(http.StatusInternalServerError, err.Error()); return}
	q.closed = true
	json.NewEncoder(w).Encode(q.state())
case "/api/eval":
	text := strings.TrimSpace(body.Text)
	if q.cur == "" || q.closed {fail(http.StatusConflict, "открытого сеанса нет"); return}
	if text == "" {fail(http.StatusBadRequest, "пусто"); return}
	q.pPred = text
	json.NewEncoder(w).Encode(q.wait(q.start(text)))
case "/api/answer":
	if q.pending == nil {fail(http.StatusConflict, "вопроса нет"); return}
	r := q.pending
	a := map[bool]string{true: "y", false: "n"}[body.Answer]
	if body.Choice != "" {a = body.Choice}
	r.answers <- a
	json.NewEncoder(w).Encode(q.wait(r))
default:
	http.NotFound(w, req)} }
func main() {
addr := flag.String("addr", "localhost:1944", "адрес")
catlg := flag.String("catlg", "", "перенести темы ПИ из папки каталога ПИ — kater (ason-spec 9.18) и выйти")
flag.Parse()
dir, err := os.Getwd()
if err != nil {log.Fatal(err)}
if _, err := os.Stat(filepath.Join(dir, "go.mod")); err != nil {log.Fatal("запуск — из папки script: go run ./infra")}
q, err := open(dir)
if err != nil {log.Fatal(err)}
if *catlg != "" {r, err := q.importKater(*catlg); for _, l := range r {fmt.Println(l)}; if err != nil {log.Fatal(err)}; return}
http.HandleFunc("/api/", q.handle)
http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(page)})
log.Printf("script: http://%s (данные: %s)", *addr, dir)
log.Fatal(http.ListenAndServe(*addr, nil)) }
func hasArrow(vs []format.Value) bool {
for _, v := range vs {if v.Kind == format.Op && v.Text == "->" {return true}}; return false }
// migrateSeans: сеанс прежнего вида (только предикации автора) один раз
// интерпретируется заново без вопросов и переписывается моделями
// «[ предикация ] -> интерпретация»; права файла (закрыт / открыт) сохраняются.
func (q *Script) migrateSeans(name string, vs []format.Value, perm os.FileMode) ([]format.Value, error) {
ask := q.in.Ask
q.in.Ask = func(string) bool { return false }
defer func() { q.in.Ask = ask }()
known := len(q.in.Unknown)
var sb strings.Builder
for _, v := range vs {
	text := v.Text
	if v.Kind == format.Array {text = format.Format(v.Items)}
	val, err := q.in.Run(text, q.env)
	t := Turn{Pred: text, Answer: val.String()}
	if err != nil {t.Answer, t.Error = err.Error(), true}
	sb.WriteString(record(t))}
q.in.Unknown = q.in.Unknown[:known]
path := q.seans(name)
tmp := path + ".tmp"
if err := os.WriteFile(tmp, []byte(sb.String()), perm); err != nil {return nil, err}
if err := os.Rename(tmp, path); err != nil {return nil, err}
out, err := format.Parse(sb.String())
if err != nil {return nil, fmt.Errorf("%s после переноса: %w", name, err)}; return out, nil }
