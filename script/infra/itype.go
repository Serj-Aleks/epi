// epi/script/infra/itype.go
package main
import ("embed"; "regexp"; "epi/script/disp"; "sync")
// страница панели: разметка, стили, код
//go:embed index.html style.css *.js
var page embed.FS
// Turn — шаг диалога: предикация, интерпретация, фреймы-интерпретаторы
type Turn struct {
	Pred   string   `json:"pred"`
	Answer string   `json:"answer"`
	Frames []string `json:"frames"`
	Error  bool     `json:"error,omitempty"`}
// Reply — ответ сервера: шаг завершён (Turn) или диспетчер спрашивает
type Reply struct {
	Turns    []Turn `json:"turns,omitempty"`
	Question string   `json:"question,omitempty"`
	Options  []string `json:"options,omitempty"`} // варианты ответа на вопрос с выбором
// Seans — сеанс протокола
type Seans struct {
	Name   string `json:"name"`
	Closed bool   `json:"closed"`}
// ask — вопрос диспетчера или транслятора; Options пусто — вопрос «да / нет»
type ask struct {
	Text    string
	Options []string}
type run struct {
	questions chan ask
	answers   chan string // «y», «n» или выбранный вариант
	done      chan []Turn}
// Script — состояние: фреймы-интерпретаторы общие, контекст — у сеанса
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
	cidx    map[string]string // концепты темы: имя (строчными) → файл
	tidx    map[string][]string} // трансклюзии темы: имя (строчными) → включаемые концепты
const limit = 10000
const gaussABC = "oiztyvslanxedbhfmg"
// aSort: алфавит темы — порядок концептов и тем (ASORT ПИ)
const aSort = "nmlrhgkcsztdbpfvwuoaeiyjqx нмлркхгжщчшсзцтдбпфвюуоёэеиыйяаьъ"
// gaussDigits: цифры 18-ричной системы Гаусса (o — 0 … g — 17)
const gaussDigits = "oiztyvslanxedbhfmg"
// qwestTypes: спутники, в которых ищет qwest; reTags, reMarks, reSpace — разметка, которую убирает qtext
var qwestTypes = []string{"md", "html", "json", "xml", "txt"}
var reTags, reMarks, reSpace = regexp.MustCompile(`<[^>]*>`), regexp.MustCompile("[#*_`\\[\\]{}|>]+"), regexp.MustCompile(`\s+`)
// satTypes: типы спутников концепта
var satTypes = []string{"md", "json", "svg", "css", "xml", "html", "txt"}
// reLink: ссылка [[Имя]] в контенте (вне блоков кода) — как SyncContentLinks в ПИ
var reLink, reFence = regexp.MustCompile(`\[\[([^\]]+)\]\]`), regexp.MustCompile("(?s)```.*?```")
// reIncl: трансклюзия {{Имя}} в контенте (вне блоков кода)
var reIncl = regexp.MustCompile(`\{\{([^{}]+)\}\}`)
// refKeys: параметры концепта, в значениях которых — имена концептов (refSlots ПИ)
var refKeys = []string{"Alias", "Attrib", "Preposit", "Prfix", "Sufix", "Radix", "Taxon", "Links", "Defs", "I-process", "I-sort"}
// reToken, reRefMark: слово имени и ссылка или трансклюзия с именем
var reToken, reRefMark = regexp.MustCompile(`[\p{L}\p{N}_-]+`), regexp.MustCompile(`(\[\[|\{\{)([^\]\}]+)(\]\]|\}\})`)
// aOrder: место литеры в алфавите темы; пробел, «-», «,» — раньше всех (BuildCharOrder ПИ)
var aOrder = func() map[rune]int {
m := map[rune]int{' ': -1, '-': -1, ',': -1}
pos := 0
for _, c := range aSort {if c != ' ' {m[c] = pos; pos++}}; return m}()
// reGauss: гаусс-индекс — число в 18-ричной системе Гаусса, цифры gaussDigits (o — 0 … g — 17)
var reGauss = regexp.MustCompile(`^[` + gaussDigits + `]{2,}$`)
// linkOrder: связи локального контекста — оси как в ПИ (Prfix, Attrib, Sufix, Preposit), затем Radix, Taxon
var linkOrder = []string{"Prfix", "Attrib", "Sufix", "Preposit", "Radix", "Taxon", "Links"}
// graphTypes: связи графа в порядке ПИ, затем Radix и вычисляемая Transcl (в списке всегда, на оси — по настройке)
// graphCodes: коды связей в списке под графом; настройки N-west, West, S-west, Nord, List — именами связей
var graphTypes = []string{"Taxon", "Links", "Attrib", "Preposit", "Prfix", "Sufix", "Radix", "Transcl"}
var graphCodes = map[string]string{"Themes": "TM", "Alias": "AL", "Attrib": "AT", "Preposit": "PP", "Prfix": "PF", "Sufix": "SF", "Taxon": "TX", "Links": "LK", "Radix": "RX", "Transcl": "TC"}
// gnode: узел графа (как в ПИ) — имя, картинка, цвет, соседи (в подсказке); Type theme — узел-тема, Theme — тема соседа
type gnode struct {
	Name      string  `json:"name"`
	Icon      string  `json:"icon,omitempty"`
	Color     string  `json:"color,omitempty"`
	Type      string  `json:"type,omitempty"`
	Theme     string  `json:"theme,omitempty"`
	Neighbors []gnode `json:"neighbors,omitempty"`}
// dicts: указатели словарей с формами слов — объекты темы и атрибуты (lexic/atrbt)
type dicts struct{ obj, atr map[string]string }
