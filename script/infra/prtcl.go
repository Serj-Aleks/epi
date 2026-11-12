// epi/script/infra/prtcl.go
package main
import ("fmt"; "log"; "os"; "path/filepath"; "sort"; "strings"; "time"; "epi/script/disp"; "epi/script/format")
// протокол темы: сеансы, ignor, запись шагов диалога, интерпретация с вопросами в браузер, состояние
// ignor, seans: файлы протокола темы
func (q *Medium) ignor() string            { return filepath.Join(q.themaDir(), "prtcl", "ignor.uic") }
func (q *Medium) seans(name string) string { return filepath.Join(q.themaDir(), "prtcl", name) }
// list: сеансы протокола, новые первыми; закрытый — файл только для чтения
func (q *Medium) list() ([]SeansRow, error) {
if q.theme == "" {return nil, nil}
es, err := os.ReadDir(filepath.Join(q.themaDir(), "prtcl"))
if os.IsNotExist(err) {return nil, nil}
if err != nil {return nil, err}
var ss []SeansRow
for _, e := range es {
	n := e.Name()
	if !strings.HasPrefix(n, "seans-") || !strings.HasSuffix(n, ".uic") {continue}
	st, err := e.Info()
	if err != nil {return nil, err}
	ss = append(ss, SeansRow{Name: n, Closed: st.Mode().Perm()&0200 == 0})}
sort.Slice(ss, func(i, j int) bool { return ss[i].Name > ss[j].Name }); return ss, nil }
// switchTo: сделать сеанс текущим; строки сеанса — модели «[ предикация ] -> интерпретация», диалог показывается
// каким был, заново не выполняется; ошибка отмечена «-> !»
func (q *Medium) switchTo(name string) error {
st, err := os.Stat(q.seans(name))
if err != nil {return err}
b, err := os.ReadFile(q.seans(name))
if err != nil {return err}
text, errLines := bangs(string(b))
vs, err := format.Parse(text)
if err != nil {return fmt.Errorf("%s: %w", name, err)}
var turns []Turn
for i := 0; i+2 < len(vs); i += 3 {
	if vs[i+1].Kind != format.Op || vs[i+1].Text != "->" {return fmt.Errorf("%s: строка %d: ожидалась модель «[ … ] -> …»", name, vs[i].Line)}
	t := Turn{Pred: vs[i].Text, Answer: vs[i+2].String()}
	if vs[i].Kind == format.Array {t.Pred = format.Format(vs[i].Items)}
	if vs[i+2].Kind == format.String {t.Answer = vs[i+2].Text}
	t.Error = errLines[vs[i+2].Line]
	turns = append(turns, t)}
q.cur, q.closed, q.turns = name, st.Mode().Perm()&0200 == 0, turns; q.saveState(); return nil }
// newSeans: открыть сеанс seans-<сегодня>.uic; в тот же день сеанс продолжается
func (q *Medium) newSeans() error {
name := "seans-" + time.Now().Format("2006-01-02") + ".uic"
if st, err := os.Stat(q.seans(name)); err == nil {if st.Mode().Perm()&0200 == 0 {return fmt.Errorf("сеанс %s сегодня уже закрыт; его можно открыть для чтения", name)}} else if err := os.WriteFile(q.seans(name), nil, 0644); err != nil {return err}; return q.switchTo(name) }
// record: шаг диалога строкой сеанса «[ предикация ] -> интерпретация»; нечитаемое и ошибка — строками, ошибка с «!»
func record(t Turn) string {
pred := "[ " + t.Pred + " ]"
if _, err := format.Parse(t.Pred); err != nil {pred = format.Value{Kind: format.String, Text: t.Pred}.String()}
ans := t.Answer
if vs, err := format.Parse(ans); t.Error || err != nil || len(vs) != 1 {ans = format.Value{Kind: format.String, Text: t.Answer}.String()}
if t.Error {ans = "! " + ans}
return pred + " -> " + ans + "\n" }
// bangs: номера строк сеанса с ошибкой «-> !»; знак убирается перед разбором (в ASON «!» не токен)
func bangs(s string) (string, map[int]bool) {
m := map[int]bool{}
ls := strings.Split(s, "\n")
for i, l := range ls {if j := strings.LastIndex(l, "-> ! "); j >= 0 {ls[i], m[i+1] = l[:j+3]+l[j+5:], true}}; return strings.Join(ls, "\n"), m }
// appendFile: дописать текст в файл
func appendFile(path, text string) error {
f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
if err != nil {return err}
if _, err := f.WriteString(text); err != nil {f.Close(); return err}; return f.Close() }
// start: интерпретация в отдельной горутине — вопросы диспетчера уходят в браузер, ответы возвращаются
// на каждую предикацию строки свой ответ; непрочитанное — как нераспознанное (вопрос, не ошибка)
// нераспознанное — в ignor.uic как есть, нечитаемое — строкой
func (q *Medium) start(text string) *job {
r := &job{questions: make(chan ask), answers: make(chan string), done: make(chan []Turn, 1)}
known := len(q.in.Unknown)
q.in.Ask = func(question string) bool {r.questions <- ask{Text: question}; return <-r.answers == "y"}
q.choose = func(text string, opts []string) string {r.questions <- ask{Text: text, Options: opts}; return <-r.answers}
go func() {
	preds, err := disp.Split(text)
	var ts []Turn
	if err != nil {
		q.in.Unreadable(text, err)
		ts = append(ts, Turn{Pred: text, Answer: "@none"})}
	for _, p := range preds {
		v, err := q.in.Run(p, q.env)
		t := Turn{Pred: p, Answer: v.String(), Frames: append([]string(nil), q.in.Used...)}
		if err != nil {t.Answer, t.Error = err.Error(), true}
		ts = append(ts, t)}
	for _, u := range q.in.Unknown[known:] {
		line := u
		if _, err := format.Parse(u); err != nil {line = format.Value{Kind: format.String, Text: u}.String()}
		if e := appendFile(q.ignor(), line+"\n"); e != nil {log.Print(e)}}
	r.done <- ts}(); return r }
// wait: следующий вопрос или результат
func (q *Medium) wait(r *job) Reply {
select {
case a := <-r.questions:
	q.pending, q.pQuest, q.pOpts = r, a.Text, a.Options; return Reply{Question: a.Text, Options: a.Options}
case ts := <-r.done:
	q.pending, q.pPred, q.pQuest, q.pOpts = nil, "", "", nil
	for _, t := range ts {
		q.turns = append(q.turns, t)
		if err := appendFile(q.seans(q.cur), record(t)); err != nil {log.Print(err)}}; return Reply{Turns: ts}} }
// state: темы, сеансы, диалог и ожидающий вопрос (страницу можно перезагрузить, не потеряв его)
func (q *Medium) state() map[string]any {
ss, err := q.list()
if err != nil {log.Print(err)}
ts, err := q.themes()
if err != nil {log.Print(err)}; return map[string]any{
	"themes": ts, "theme": q.theme,
	"seanses": ss, "current": q.cur, "closed": q.closed, "turns": q.turns,
	"pending": map[string]any{"pred": q.pPred, "question": q.pQuest, "options": q.pOpts},} }
