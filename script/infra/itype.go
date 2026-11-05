// epi/script/infra/itype.go
package main
import (_ "embed"; "epi/script/disp"; "sync")
//go:embed index.html
var page []byte
// Turn — шаг диалога: предикация, интерпретация, фреймы-интерпретаторы.
type Turn struct {
	Pred   string   `json:"pred"`
	Answer string   `json:"answer"`
	Frames []string `json:"frames"`
	Error  bool     `json:"error,omitempty"`}
// Reply — ответ сервера: шаг завершён (Turn) или диспетчер спрашивает.
type Reply struct {
	Turns    []Turn `json:"turns,omitempty"`
	Question string `json:"question,omitempty"`}
// Seans — сеанс протокола.
type Seans struct {
	Name   string `json:"name"`
	Closed bool   `json:"closed"`}
type run struct {
	questions chan string
	answers   chan bool
	done      chan []Turn}
// Script — состояние: фреймы-интерпретаторы общие, контекст — у сеанса.
type Script struct {
	mu      sync.Mutex
	in      *disp.Interp
	dir     string
	theme   string // активная тема (папка thema/<имя>) или ""
	cur     string // текущий сеанс (имя файла) или ""
	closed  bool
	env     *disp.Env
	turns   []Turn
	pending *run
	pPred   string // предикация, по которой ждёт ответа вопрос
	pQuest  string}
const limit = 10000
const gaussABC = "oiztyvslanxedbhfmg"
