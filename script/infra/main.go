// epi/script/infra/main.go
package main
import ("encoding/json"; "flag"; "log"; "net/http"; "os"; "path/filepath"; "strings"; "epi/script/disp"; "epi/script/format")
// сервер панели — интерфейс среды: строка предикаций в браузере, ответ диспетчера — следующей строкой диалога
// данные в папке script: lexic — системный лексикон, thema — темы автора (концепты и протокол prtcl)
// запуск из папки script: go run ./infra [-addr localhost:5832]
// open: среда — фреймы-интерпретаторы lexic/preposit, транслятор trans, препозиции без вычисления операнда
// (nexus, qwest, qwizt, clone, rename, import, union); фреймы автора — в теме, алиасы препозиций — общие
// для всех тем (имя интерпретатора не алиас); контекст — системные алиасы и концепты темы
func open(dir string) (*Medium, error) {
q := &Medium{dir: dir}
q.in = disp.New(limit, func(string) bool { return false })
for _, f := range q.in.Frames {if err := q.syncFrame(f); err != nil {return nil, err}}
q.in.FrameOf("trans").Kernel["trans"] = q.trans
q.in.Raw = map[string]func(format.Value) (format.Value, bool){"nexus": q.nexus, "qwest": q.qwest, "clone": q.clone, "rename": q.rename, "qwizt": q.qwizt, "import": q.importC, "union": q.union}
q.in.OnParam = func(frame, param string, pred []format.Value) {if err := q.writeConcept(frame, param, pred); err != nil {log.Print(err)}}
q.in.OnFrame = func(name string, body []format.Value) {if err := q.writeBody(name, body); err != nil {log.Print(err)}}
q.in.OnAlias = func(name, target string) {
	if q.in.FrameOf(name) != nil {return}
	if err := writeLines(q.frameFile("preposit", name), []string{name + " : " + target}); err != nil {log.Print(err)}}
q.env = disp.NewEnv(nil)
if err := q.loadAliases(); err != nil {return nil, err}
if err := q.startTheme(); err != nil {return nil, err}; return q, nil }
// serve: запросы панели /api/…; Choice — ответ на вопрос с выбором, Ext — тип спутника
// situation: Text — «объект препозиция объект»; sat: GET читает спутник, POST пишет; img — <папка концепта>/img/<файл>, без name — <тема>/img/<файл>
// icon с theme — концепт другой темы (qwizt); в новой теме сразу открывается сеанс; закрытый сеанс — файл только для чтения
func (q *Medium) serve(w http.ResponseWriter, req *http.Request) {
w.Header().Set("Content-Type", "application/json")
q.mu.Lock()
defer q.mu.Unlock()
var body struct {
	Text   string `json:"text"`
	To     string `json:"to"`
	Answer bool   `json:"answer"`
	Choice string `json:"choice"`
	Name   string `json:"name"`
	Ext    string `json:"ext"`}
if req.Method != http.MethodGet {if err := json.NewDecoder(req.Body).Decode(&body); err != nil {http.Error(w, err.Error(), http.StatusBadRequest); return}}
fail := func(code int, msg string) { http.Error(w, msg, code) }
if req.URL.Path == "/api/nexus" {json.NewEncoder(w).Encode(q.nexusGraph(req.URL.Query().Get("name"))); return}
if req.URL.Path == "/api/card" {json.NewEncoder(w).Encode(q.card(req.URL.Query().Get("name"))); return}
if req.URL.Path == "/api/qwest" {json.NewEncoder(w).Encode(q.qwestFind(req.URL.Query().Get("q"))); return}
if req.URL.Path == "/api/find" {json.NewEncoder(w).Encode(q.findIn(req.URL.Query().Get("name"), req.URL.Query().Get("q"))); return}
if req.URL.Path == "/api/qwizt" {json.NewEncoder(w).Encode(q.qwizFind(req.URL.Query().Get("q"))); return}
if req.URL.Path == "/api/taxa" {json.NewEncoder(w).Encode(q.taxa()); return}
if req.URL.Path == "/api/dict" {json.NewEncoder(w).Encode(q.dict(req.URL.Query().Get("name"))); return}
if req.URL.Path == "/api/themes" {json.NewEncoder(w).Encode(q.themesInfo()); return}
if req.URL.Path == "/api/themeicon" {n := req.URL.Query().Get("name"); if _, ok := q.themeExists(n); !ok || q.themeIcon(n) == "" {http.NotFound(w, req); return}; w.Header().Del("Content-Type"); http.ServeFile(w, req, q.themeIcon(n)); return}
if req.URL.Path == "/api/process" {json.NewEncoder(w).Encode(q.process(req.URL.Query().Get("name"))); return}
if req.URL.Path == "/api/children" {json.NewEncoder(w).Encode(q.children(req.URL.Query().Get("name"))); return}
if req.URL.Path == "/api/situation" {
	f := strings.Fields(body.Text)
	if len(f) != 3 {fail(http.StatusBadRequest, "ситуация — три слова: объект, препозиция, объект"); return}
	if err := q.situation(body.Name, f[0], f[1], f[2]); err != nil {fail(http.StatusBadRequest, err.Error()); return}
	json.NewEncoder(w).Encode(q.process(body.Name)); return}
if req.URL.Path == "/api/sat" {
	if req.Method == http.MethodGet {
		p := q.satFile(req.URL.Query().Get("name"), req.URL.Query().Get("ext"))
		b, err := os.ReadFile(p)
		if p == "" || err != nil && !os.IsNotExist(err) {fail(http.StatusNotFound, "нет спутника"); return}
		json.NewEncoder(w).Encode(map[string]string{"text": string(b)}); return}
	if err := q.writeSat(body.Name, body.Ext, body.Text); err != nil {fail(http.StatusBadRequest, err.Error()); return}
	json.NewEncoder(w).Encode(q.card(body.Name)); return}
if req.URL.Path == "/api/img" {
	n, f, dir := req.URL.Query().Get("name"), req.URL.Query().Get("f"), q.themaDir()
	if n != "" {p := q.conceptPath(n); if p == "" {http.NotFound(w, req); return}; dir = filepath.Dir(p)}
	if f == "" || f != filepath.Base(f) {http.NotFound(w, req); return}
	w.Header().Del("Content-Type"); http.ServeFile(w, req, filepath.Join(dir, "img", f)); return}
if req.URL.Path == "/api/icon" {
	p := q.iconFile(req.URL.Query().Get("name"))
	if t := req.URL.Query().Get("theme"); t != "" && t != q.theme {p = q.iconOther(t, req.URL.Query().Get("name"))}
	if p == "" {http.NotFound(w, req); return}
	w.Header().Del("Content-Type"); http.ServeFile(w, req, p); return}
if q.pending != nil && req.URL.Path != "/api/answer" && req.URL.Path != "/api/state" {fail(http.StatusConflict, "ждёт ответа на вопрос"); return}
switch req.URL.Path {
case "/api/state":
	json.NewEncoder(w).Encode(q.state())
case "/api/thema/new":
	if err := q.createTheme(strings.TrimSpace(body.Name)); err != nil {fail(http.StatusBadRequest, err.Error()); return}
	if q.cur == "" {
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
case "/api/new":
	if q.theme == "" {fail(http.StatusConflict, "темы нет: сначала создайте тему"); return}
	if err := q.newSeans(); err != nil {fail(http.StatusConflict, err.Error()); return}
	json.NewEncoder(w).Encode(q.state())
case "/api/open":
	if filepath.Base(body.Name) != body.Name || !strings.HasPrefix(body.Name, "seans-") {fail(http.StatusBadRequest, "нет такого сеанса"); return}
	if err := q.switchTo(body.Name); err != nil {fail(http.StatusNotFound, err.Error()); return}
	json.NewEncoder(w).Encode(q.state())
case "/api/close":
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
// main: сервер панели — из папки script
func main() {
addr := flag.String("addr", "localhost:5832", "адрес")
flag.Parse()
dir, err := os.Getwd()
if err != nil {log.Fatal(err)}
if _, err := os.Stat(filepath.Join(dir, "go.mod")); err != nil {log.Fatal("запуск — из папки script: go run ./infra")}
q, err := open(dir)
if err != nil {log.Fatal(err)}
http.HandleFunc("/api/", q.serve)
http.Handle("/", noCache(http.FileServer(http.FS(page))))
log.Printf("script: http://%s (данные: %s)", *addr, dir)
log.Fatal(http.ListenAndServe(*addr, nil)) }
// noCache: страница панели всегда свежая — браузер не берёт прежние стили и код из кэша после обновления
func noCache(h http.Handler) http.Handler {return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {w.Header().Set("Cache-Control", "no-cache"); h.ServeHTTP(w, r)}) }
