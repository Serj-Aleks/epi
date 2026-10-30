// Qwest — строка взаимодействия с диспетчером предикаций в браузере
// (claude/ason-spec.md, раздел 9.11). Предикация вводится в строке, ответ
// диспетчера — следующей строкой диалога. Сессия сохраняется: каждая
// предикация порождает модель «Predicatio — Interpretatio» в файле сессии.
//
// Запуск: go run ./qwest [-addr 127.0.0.1:8790] [-data папка]
package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

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

type run struct {
	questions chan string
	answers   chan bool
	done      chan Turn
}

// Session — сессия Qwest: контекст интерпретации и модели диалога.
type Session struct {
	mu      sync.Mutex
	in      *disp.Interp
	env     *disp.Env
	turns   []Turn
	pending *run
	pPred   string // предикация, по которой ждёт ответа вопрос
	pQuest  string
	dir     string
}

const limit = 10000

func openSession(dir string) (*Session, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	s := &Session{dir: dir}
	s.in = disp.New(limit, func(string) bool { return false })
	env, err := s.in.Global()
	if err != nil {
		return nil, err
	}
	s.env = env
	// восстановление: модели сессии читаются, предикации интерпретируются
	// заново без вопросов — так восстанавливается контекст (определения)
	if b, err := os.ReadFile(s.file("sessio.ason")); err == nil {
		vs, err := ason.Parse(string(b))
		if err != nil {
			return nil, fmt.Errorf("sessio.ason: %w", err)
		}
		for _, v := range vs {
			t := turnOf(v)
			if t.Pred == "" {
				continue
			}
			if !t.Error {
				s.in.Run(t.Pred, s.env)
			}
			s.turns = append(s.turns, t)
		}
	}
	if b, err := os.ReadFile(s.file("ignota.ason")); err == nil {
		vs, err := ason.Parse(string(b))
		if err != nil {
			return nil, fmt.Errorf("ignota.ason: %w", err)
		}
		for _, v := range vs {
			if v.Kind == ason.Array {
				s.in.Unknown = append(s.in.Unknown, ason.Format(v.Items))
			}
		}
	}
	return s, nil
}

func (s *Session) file(name string) string { return filepath.Join(s.dir, name) }

// model: запись шага как модели с ключами слотов (одна строка файла сессии).
func model(t Turn) string {
	interp := "@none"
	if len(t.Frames) > 0 {
		interp = "[ " + strings.Join(t.Frames, " ") + " ]"
	}
	ans := ason.Value{Kind: ason.String, Text: t.Answer}.String()
	if t.Error {
		interp = "'erratum"
	}
	pred := "[ " + t.Pred + " ]"
	if _, err := ason.Parse(t.Pred); err != nil { // нечитаемый ввод — строкой
		pred = ason.Value{Kind: ason.String, Text: t.Pred}.String()
	}
	return fmt.Sprintf("{ Predicatio : %s  Interpretatio : %s  Interpres : %s }\n", pred, ans, interp)
}

func turnOf(v ason.Value) Turn {
	var t Turn
	if v.Kind != ason.Object {
		return t
	}
	for _, e := range v.Entries() {
		if len(e.Value) != 1 {
			continue
		}
		x := e.Value[0]
		switch e.Key.Text {
		case "Predicatio":
			t.Pred = ason.Format(x.Items)
			if x.Kind == ason.String {
				t.Pred = x.Text
			}
		case "Interpretatio":
			t.Answer = x.Text
		case "Interpres":
			if x.Kind == ason.LitTerm && x.Text == "'erratum" {
				t.Error = true
			}
			for _, f := range x.Items {
				t.Frames = append(t.Frames, f.Text)
			}
		}
	}
	return t
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
func (s *Session) start(text string) *run {
	r := &run{questions: make(chan string), answers: make(chan bool), done: make(chan Turn, 1)}
	known := len(s.in.Unknown)
	s.in.Ask = func(q string) bool {
		r.questions <- q
		return <-r.answers
	}
	go func() {
		v, err := s.in.Run(text, s.env)
		t := Turn{Pred: text, Answer: v.String(), Frames: append([]string(nil), s.in.Used...)}
		var st disp.Stopped
		if errors.As(err, &st) {
			t.Answer, t.Error = err.Error(), true
		} else if err != nil {
			t.Answer, t.Error = err.Error(), true
		}
		for _, u := range s.in.Unknown[known:] {
			if e := appendFile(s.file("ignota.ason"), "[ "+u+" ]\n"); e != nil {
				log.Print(e)
			}
		}
		r.done <- t
	}()
	return r
}

// wait: следующий вопрос или результат.
func (s *Session) wait(r *run) Reply {
	select {
	case q := <-r.questions:
		s.pending, s.pQuest = r, q
		return Reply{Question: q}
	case t := <-r.done:
		s.pending, s.pPred, s.pQuest = nil, "", ""
		s.turns = append(s.turns, t)
		if err := appendFile(s.file("sessio.ason"), model(t)); err != nil {
			log.Print(err)
		}
		return Reply{Turn: &t}
	}
}

func (s *Session) handle(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	s.mu.Lock()
	defer s.mu.Unlock()
	var body struct {
		Text   string `json:"text"`
		Answer bool   `json:"answer"`
	}
	if req.Method != http.MethodGet {
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	switch req.URL.Path {
	case "/api/history":
		// ожидающий вопрос отдаётся тоже: страницу можно перезагрузить, не потеряв его
		json.NewEncoder(w).Encode(map[string]any{"turns": s.turns, "pending": map[string]string{"pred": s.pPred, "question": s.pQuest}})
	case "/api/eval":
		text := strings.TrimSpace(body.Text)
		if s.pending != nil || text == "" {
			http.Error(w, "ждёт ответа на вопрос", http.StatusConflict)
			return
		}
		s.pPred = text
		json.NewEncoder(w).Encode(s.wait(s.start(text)))
	case "/api/answer":
		if s.pending == nil {
			http.Error(w, "вопроса нет", http.StatusConflict)
			return
		}
		r := s.pending
		r.answers <- body.Answer
		json.NewEncoder(w).Encode(s.wait(r))
	default:
		http.NotFound(w, req)
	}
}

func main() {
	addr := flag.String("addr", "127.0.0.1:8790", "адрес")
	dir := flag.String("data", "qwest-data", "папка сессии")
	flag.Parse()
	s, err := openSession(*dir)
	if err != nil {
		log.Fatal(err)
	}
	http.HandleFunc("/api/", s.handle)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(page)
	})
	log.Printf("Qwest: http://%s (сессия: %s)", *addr, *dir)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
