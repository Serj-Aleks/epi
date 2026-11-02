// Qwest — строка взаимодействия с диспетчером предикаций в браузере
// (claude/ason-spec.md, раздел 9.11). Предикация вводится в строке, ответ
// диспетчера — следующей строкой диалога.
//
// Данные — рядом с кодом, в папке script (ason-spec 0.20): lexic — темы и
// концепты (в теме preposit — алиасы plus, minus и фреймы-интерпретаторы
// arifm, logic, мета-файлы .uic); prtcl — протокол: ignor.uic —
// нераспознанные предикации, сеансы seans-<дата открытия>.uic, строки —
// модели «[ предикация ] -> интерпретация». Сеанс открывается и закрывается
// автором (кнопками); открытый можно продолжать, любой — читать. Закрытый
// сеанс — файл только для чтения.
//
// Запуск из папки script: go run ./infra [-addr localhost:1944]
package main

import (
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"epi/script/disp"
	"epi/script/format"
)

//go:embed index.html
var page []byte

// Turn — шаг диалога: предикация, интерпретация, фреймы-интерпретаторы.
type Turn struct {
	Pred   string   `json:"pred"`
	Answer string   `json:"answer"`
	Frames []string `json:"frames"`
	Error  bool     `json:"error,omitempty"`
}

// Reply — ответ сервера: шаг завершён (Turn) или диспетчер спрашивает.
type Reply struct {
	Turns    []Turn `json:"turns,omitempty"`
	Question string `json:"question,omitempty"`
}

// Seans — сеанс протокола.
type Seans struct {
	Name   string `json:"name"`
	Closed bool   `json:"closed"`
}

type run struct {
	questions chan string
	answers   chan bool
	done      chan []Turn
}

// Qwest — состояние: фреймы-интерпретаторы общие, контекст — у сеанса.
type Qwest struct {
	mu      sync.Mutex
	in      *disp.Interp
	dir     string
	cur     string // текущий сеанс (имя файла) или ""
	closed  bool
	env     *disp.Env
	turns   []Turn
	pending *run
	pPred   string // предикация, по которой ждёт ответа вопрос
	pQuest  string
}

const limit = 10000

func open(dir string) (*Qwest, error) {
	if err := os.MkdirAll(filepath.Join(dir, "prtcl"), 0755); err != nil {
		return nil, err
	}
	q := &Qwest{dir: dir}
	q.in = disp.New(limit, func(string) bool { return false })
	// фреймы-интерпретаторы: lexic/preposit (9.13); прежние файлы переносятся
	for _, f := range q.in.Frames {
		if err := q.loadFrame(f); err != nil {
			return nil, err
		}
		if err := q.migrateFrame(f); err != nil {
			return nil, err
		}
	}
	if b, err := os.ReadFile(q.ignor()); err == nil {
		vs, err := format.Parse(string(b))
		if err != nil {
			return nil, fmt.Errorf("ignor.uic: %w", err)
		}
		for _, v := range vs {
			if v.Kind == format.Array {
				q.in.Unknown = append(q.in.Unknown, format.Format(v.Items))
			}
		}
	}
	q.in.OnModel = func(f *disp.Frame, m disp.Model) {
		if err := q.writeFrame(f); err != nil {
			log.Print(err)
		}
	}
	// контекст общий (9.13.4а): определения не принадлежат сеансу;
	// алиасы препозиций — концепты темы preposit
	q.env = disp.NewEnv(nil)
	if err := q.loadAliases(); err != nil {
		return nil, err
	}
	// при запуске — последний открытый сеанс, если есть
	ss, err := q.list()
	if err != nil {
		return nil, err
	}
	for _, s := range ss {
		if !s.Closed {
			if err := q.switchTo(s.Name); err != nil {
				return nil, err
			}
			break
		}
	}
	return q, nil
}

func (q *Qwest) ignor() string            { return filepath.Join(q.dir, "prtcl", "ignor.uic") }
func (q *Qwest) seans(name string) string { return filepath.Join(q.dir, "prtcl", name) }

// list: сеансы протокола, новые первыми; закрытый — файл только для чтения.
func (q *Qwest) list() ([]Seans, error) {
	es, err := os.ReadDir(filepath.Join(q.dir, "prtcl"))
	if err != nil {
		return nil, err
	}
	var ss []Seans
	for _, e := range es {
		n := e.Name()
		if !strings.HasPrefix(n, "seans-") || !strings.HasSuffix(n, ".uic") {
			continue
		}
		st, err := e.Info()
		if err != nil {
			return nil, err
		}
		ss = append(ss, Seans{Name: n, Closed: st.Mode().Perm()&0200 == 0})
	}
	sort.Slice(ss, func(i, j int) bool { return ss[i].Name > ss[j].Name })
	return ss, nil
}

// switchTo: сделать сеанс текущим. Контекст сеанса восстанавливается:
// его предикации интерпретируются заново без вопросов.
func (q *Qwest) switchTo(name string) error {
	st, err := os.Stat(q.seans(name))
	if err != nil {
		return err
	}
	b, err := os.ReadFile(q.seans(name))
	if err != nil {
		return err
	}
	vs, err := format.Parse(string(b))
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	if len(vs) > 0 && !hasArrow(vs) { // сеанс прежнего вида — только предикации
		if vs, err = q.migrateSeans(name, vs, st.Mode().Perm()); err != nil {
			return err
		}
	}
	// сеанс заново не выполняется (9.13.4а): строки — модели
	// «[ предикация ] -> интерпретация», диалог показывается, каким был
	var turns []Turn
	for i := 0; i+2 < len(vs); i += 3 {
		if vs[i+1].Kind != format.Op || vs[i+1].Text != "->" {
			return fmt.Errorf("%s: строка %d: ожидалась модель «[ … ] -> …»", name, vs[i].Line)
		}
		t := Turn{Pred: vs[i].Text, Answer: vs[i+2].String()}
		if vs[i].Kind == format.Array {
			t.Pred = format.Format(vs[i].Items)
		}
		if vs[i+2].Kind == format.String { // интерпретация — строка (в том числе сообщение об ошибке)
			t.Answer = vs[i+2].Text
		}
		turns = append(turns, t)
	}
	q.cur, q.closed, q.turns = name, st.Mode().Perm()&0200 == 0, turns
	return nil
}

// record: шаг диалога как строка сеанса — модель «[ предикация ] -> интерпретация»;
// нечитаемая предикация и сообщение об ошибке — строками.
func record(t Turn) string {
	pred := "[ " + t.Pred + " ]"
	if _, err := format.Parse(t.Pred); err != nil {
		pred = format.Value{Kind: format.String, Text: t.Pred}.String()
	}
	ans := t.Answer
	if vs, err := format.Parse(ans); t.Error || err != nil || len(vs) != 1 {
		ans = format.Value{Kind: format.String, Text: t.Answer}.String()
	}
	return pred + " -> " + ans + "\n"
}
func appendFile(path, text string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	if _, err := f.WriteString(text); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// start: интерпретация в отдельной горутине; вопросы диспетчера (0.7, 0.16)
// уходят в браузер, ответы возвращаются.
func (q *Qwest) start(text string) *run {
	r := &run{questions: make(chan string), answers: make(chan bool), done: make(chan []Turn, 1)}
	known := len(q.in.Unknown)
	q.in.Ask = func(question string) bool {
		r.questions <- question
		return <-r.answers
	}
	go func() {
		// на каждую предикацию строки — свой ответ (9.11.8)
		preds, err := disp.Split(text)
		if err != nil {
			preds = []string{text} // нечитаемый ввод — одна строка с сообщением
		}
		var ts []Turn
		for _, p := range preds {
			v, err := q.in.Run(p, q.env)
			t := Turn{Pred: p, Answer: v.String(), Frames: append([]string(nil), q.in.Used...)}
			if err != nil {
				t.Answer, t.Error = err.Error(), true
			}
			ts = append(ts, t)
		}
		for _, u := range q.in.Unknown[known:] {
			if e := appendFile(q.ignor(), "[ "+u+" ]\n"); e != nil {
				log.Print(e)
			}
		}
		r.done <- ts
	}()
	return r
}

// wait: следующий вопрос или результат.
func (q *Qwest) wait(r *run) Reply {
	select {
	case question := <-r.questions:
		q.pending, q.pQuest = r, question
		return Reply{Question: question}
	case ts := <-r.done:
		q.pending, q.pPred, q.pQuest = nil, "", ""
		for _, t := range ts {
			q.turns = append(q.turns, t)
			if err := appendFile(q.seans(q.cur), record(t)); err != nil {
				log.Print(err)
			}
		}
		return Reply{Turns: ts}
	}
}

func (q *Qwest) state() map[string]any {
	ss, err := q.list()
	if err != nil {
		log.Print(err)
	}
	return map[string]any{
		"seanses": ss, "current": q.cur, "closed": q.closed, "turns": q.turns,
		// ожидающий вопрос отдаётся тоже: страницу можно перезагрузить, не потеряв его
		"pending": map[string]string{"pred": q.pPred, "question": q.pQuest},
	}
}

func (q *Qwest) handle(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	q.mu.Lock()
	defer q.mu.Unlock()
	var body struct {
		Text   string `json:"text"`
		Answer bool   `json:"answer"`
		Name   string `json:"name"`
	}
	if req.Method != http.MethodGet {
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	fail := func(code int, msg string) { http.Error(w, msg, code) }
	if q.pending != nil && req.URL.Path != "/api/answer" && req.URL.Path != "/api/state" {
		fail(http.StatusConflict, "ждёт ответа на вопрос")
		return
	}
	switch req.URL.Path {
	case "/api/state":
		json.NewEncoder(w).Encode(q.state())
	case "/api/new": // открыть новый сеанс: seans-<сегодня>.uic
		name := "seans-" + time.Now().Format("2006-01-02") + ".uic"
		// новый сеанс — на следующий день; в тот же день сеанс продолжается
		if st, err := os.Stat(q.seans(name)); err == nil {
			if st.Mode().Perm()&0200 == 0 {
				fail(http.StatusConflict, "сеанс "+name+" сегодня уже закрыт; его можно открыть для чтения")
				return
			}
		} else if err := os.WriteFile(q.seans(name), nil, 0644); err != nil {
			fail(http.StatusInternalServerError, err.Error())
			return
		}
		if err := q.switchTo(name); err != nil {
			fail(http.StatusInternalServerError, err.Error())
			return
		}
		json.NewEncoder(w).Encode(q.state())
	case "/api/open": // открыть сеанс: продолжить (не закрыт) или читать (закрыт)
		if filepath.Base(body.Name) != body.Name || !strings.HasPrefix(body.Name, "seans-") {
			fail(http.StatusBadRequest, "нет такого сеанса")
			return
		}
		if err := q.switchTo(body.Name); err != nil {
			fail(http.StatusNotFound, err.Error())
			return
		}
		json.NewEncoder(w).Encode(q.state())
	case "/api/close": // закрыть текущий сеанс: файл только для чтения
		if q.cur == "" || q.closed {
			fail(http.StatusConflict, "открытого сеанса нет")
			return
		}
		if err := os.Chmod(q.seans(q.cur), 0444); err != nil {
			fail(http.StatusInternalServerError, err.Error())
			return
		}
		q.closed = true
		json.NewEncoder(w).Encode(q.state())
	case "/api/eval":
		text := strings.TrimSpace(body.Text)
		if q.cur == "" || q.closed {
			fail(http.StatusConflict, "открытого сеанса нет")
			return
		}
		if text == "" {
			fail(http.StatusBadRequest, "пусто")
			return
		}
		q.pPred = text
		json.NewEncoder(w).Encode(q.wait(q.start(text)))
	case "/api/answer":
		if q.pending == nil {
			fail(http.StatusConflict, "вопроса нет")
			return
		}
		r := q.pending
		r.answers <- body.Answer
		json.NewEncoder(w).Encode(q.wait(r))
	default:
		http.NotFound(w, req)
	}
}

func main() {
	addr := flag.String("addr", "localhost:1944", "адрес")
	flag.Parse()
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err != nil {
		log.Fatal("запуск — из папки script: go run ./infra")
	}
	q, err := open(dir)
	if err != nil {
		log.Fatal(err)
	}
	http.HandleFunc("/api/", q.handle)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(page)
	})
	log.Printf("Qwest: http://%s (данные: %s)", *addr, dir)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func hasArrow(vs []format.Value) bool {
	for _, v := range vs {
		if v.Kind == format.Op && v.Text == "->" {
			return true
		}
	}
	return false
}

// migrateSeans: сеанс прежнего вида (только предикации автора) один раз
// интерпретируется заново без вопросов и переписывается моделями
// «[ предикация ] -> интерпретация»; права файла (закрыт / открыт) сохраняются.
func (q *Qwest) migrateSeans(name string, vs []format.Value, perm os.FileMode) ([]format.Value, error) {
	ask := q.in.Ask
	q.in.Ask = func(string) bool { return false }
	defer func() { q.in.Ask = ask }()
	known := len(q.in.Unknown)
	var sb strings.Builder
	for _, v := range vs {
		text := v.Text
		if v.Kind == format.Array {
			text = format.Format(v.Items)
		}
		val, err := q.in.Run(text, q.env)
		t := Turn{Pred: text, Answer: val.String()}
		if err != nil {
			t.Answer, t.Error = err.Error(), true
		}
		sb.WriteString(record(t))
	}
	q.in.Unknown = q.in.Unknown[:known]
	path := q.seans(name)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(sb.String()), perm); err != nil {
		return nil, err
	}
	if err := os.Rename(tmp, path); err != nil {
		return nil, err
	}
	out, err := format.Parse(sb.String())
	if err != nil {
		return nil, fmt.Errorf("%s после переноса: %w", name, err)
	}
	return out, nil
}
