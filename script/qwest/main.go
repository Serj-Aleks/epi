// Qwest — строка взаимодействия с диспетчером предикаций в браузере
// (claude/ason-spec.md, раздел 9.11). Предикация вводится в строке, ответ
// диспетчера — следующей строкой диалога.
//
// Данные (папка qwest): фреймы-интерпретаторы (arifm, logic) с моделями
// интерпретаций, ignor.uic — нераспознанные предикации, prtcl — протокол:
// сеансы seans-<дата открытия>.uic. Сеанс открывается и закрывается автором
// (кнопками); открытый можно продолжать, любой — читать. Закрытый сеанс —
// файл только для чтения.
//
// Запуск: go run ./qwest [-addr 127.0.0.1:1944] [-data папка]
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

	"epi/script/ason"
	"epi/script/disp"
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
	Turn     *Turn  `json:"turn,omitempty"`
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
	done      chan Turn
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
	// модели фреймов-интерпретаторов: «[ предикация ] -> интерпретация» (9.11)
	for _, f := range q.in.Frames {
		b, err := os.ReadFile(q.file(f.Name))
		if err != nil {
			continue
		}
		vs, err := ason.Parse(string(b))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", f.Name, err)
		}
		for i := 0; i+2 < len(vs); i += 3 {
			if vs[i].Kind != ason.Array || vs[i+1].Kind != ason.Op || vs[i+1].Text != "->" {
				return nil, fmt.Errorf("%s: строка %d: ожидалась модель «[ … ] -> …»", f.Name, vs[i].Line)
			}
			f.Add(disp.Model{Pred: ason.Format(vs[i].Items), Value: vs[i+2]})
		}
	}
	if b, err := os.ReadFile(q.file("ignor.uic")); err == nil {
		vs, err := ason.Parse(string(b))
		if err != nil {
			return nil, fmt.Errorf("ignor.uic: %w", err)
		}
		for _, v := range vs {
			if v.Kind == ason.Array {
				q.in.Unknown = append(q.in.Unknown, ason.Format(v.Items))
			}
		}
	}
	q.in.OnModel = func(f *disp.Frame, m disp.Model) {
		if err := appendFile(q.file(f.Name), "[ "+m.Pred+" ] -> "+m.Value.String()+"\n"); err != nil {
			log.Print(err)
		}
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

func (q *Qwest) file(name string) string  { return filepath.Join(q.dir, name) }
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
	env, err := q.in.Global()
	if err != nil {
		return err
	}
	q.cur, q.closed, q.env, q.turns = name, st.Mode().Perm()&0200 == 0, env, nil
	b, err := os.ReadFile(q.seans(name))
	if err != nil {
		return err
	}
	vs, err := ason.Parse(string(b))
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	ask := q.in.Ask
	q.in.Ask = func(string) bool { return false }
	defer func() { q.in.Ask = ask }()
	known := len(q.in.Unknown)
	for _, v := range vs {
		text := v.Text
		if v.Kind == ason.Array {
			text = ason.Format(v.Items)
		}
		val, err := q.in.Run(text, q.env)
		t := Turn{Pred: text, Answer: val.String(), Frames: append([]string(nil), q.in.Used...)}
		if err != nil {
			t.Answer, t.Error = err.Error(), true
		}
		q.turns = append(q.turns, t)
	}
	q.in.Unknown = q.in.Unknown[:known]
	return nil
}

// input: предикация автора как строка сеанса — блок данных `[ … ]` (9.3);
// нечитаемый ввод — строкой.
func input(text string) string {
	if _, err := ason.Parse(text); err != nil {
		return ason.Value{Kind: ason.String, Text: text}.String() + "\n"
	}
	return "[ " + text + " ]\n"
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
	r := &run{questions: make(chan string), answers: make(chan bool), done: make(chan Turn, 1)}
	known := len(q.in.Unknown)
	q.in.Ask = func(question string) bool {
		r.questions <- question
		return <-r.answers
	}
	go func() {
		v, err := q.in.Run(text, q.env)
		t := Turn{Pred: text, Answer: v.String(), Frames: append([]string(nil), q.in.Used...)}
		if err != nil {
			t.Answer, t.Error = err.Error(), true
		}
		for _, u := range q.in.Unknown[known:] {
			if e := appendFile(q.file("ignor.uic"), "[ "+u+" ]\n"); e != nil {
				log.Print(e)
			}
		}
		r.done <- t
	}()
	return r
}

// wait: следующий вопрос или результат.
func (q *Qwest) wait(r *run) Reply {
	select {
	case question := <-r.questions:
		q.pending, q.pQuest = r, question
		return Reply{Question: question}
	case t := <-r.done:
		q.pending, q.pPred, q.pQuest = nil, "", ""
		q.turns = append(q.turns, t)
		if err := appendFile(q.seans(q.cur), input(t.Pred)); err != nil {
			log.Print(err)
		}
		return Reply{Turn: &t}
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
	addr := flag.String("addr", "127.0.0.1:1944", "адрес")
	dir := flag.String("data", "qwest", "папка данных Qwest")
	flag.Parse()
	q, err := open(*dir)
	if err != nil {
		log.Fatal(err)
	}
	http.HandleFunc("/api/", q.handle)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(page)
	})
	log.Printf("Qwest: http://%s (данные: %s)", *addr, *dir)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
