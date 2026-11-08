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
	Question string   `json:"question,omitempty"`
	Options  []string `json:"options,omitempty"`} // вопрос с выбором (trans): варианты ответа
// Seans — сеанс протокола.
type Seans struct {
	Name   string `json:"name"`
	Closed bool   `json:"closed"`}
// ask — вопрос диспетчера или транслятора; Options пусто — вопрос «да / нет».
type ask struct {
	Text    string
	Options []string}
type run struct {
	questions chan ask
	answers   chan string // «y», «n» или выбранный вариант
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
	pQuest  string
	pOpts   []string
	choose  func(text string, opts []string) string // вопрос с выбором в текущей интерпретации
	cidx    map[string]string // концепты темы: имя (строчными) → файл; клоны лежат не по пути своего имени (9.18)
	tidx    map[string][]string} // трансклюзии темы: имя (строчными) → включаемые концепты (из спутников md, html, json; 9.19)
const limit = 10000
const gaussABC = "oiztyvslanxedbhfmg"
