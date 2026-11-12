// epi/script/infra/itype.go
package main
import ("embed"; "regexp"; "sync"; "epi/script/disp")
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
// SeansRow — сеанс в списке сеансов: имя, закрыт ли
type SeansRow struct {
	Name   string `json:"name"`
	Closed bool   `json:"closed"`}
// ask — вопрос диспетчера или транслятора; Options пусто — вопрос «да / нет»
type ask struct {
	Text    string
	Options []string}
// job: интерпретация в отдельной горутине — вопросы в браузер, ответы, итог
type job struct {
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
	pending *job
	pPred   string // предикация, по которой ждёт ответа вопрос
	pQuest  string
	pOpts   []string
	choose  func(text string, opts []string) string // вопрос с выбором в текущей интерпретации
	cidx    map[string]string // концепты темы: имя (строчными) → файл
	tidx    map[string][]string} // трансклюзии темы: имя (строчными) → включаемые концепты
const limit = 10000
// aSort: алфавит темы — порядок концептов и тем
const aSort = "nmlrhgkcsztdbpfvwuoaeiyjqx нмлркхгжщчшсзцтдбпфвюуоёэеиыйяаьъ"
// qwestTypes: спутники, в которых ищет qwest; reTags, reMarks, reSpace — разметка, которую убирает qtext
var qwestTypes = []string{"md", "html", "json", "xml", "txt"}
var reTags, reMarks, reSpace = regexp.MustCompile(`<[^>]*>`), regexp.MustCompile("[#*_`\\[\\]{}|>]+"), regexp.MustCompile(`\s+`)
// satTypes: типы спутников концепта
var satTypes = []string{"md", "json", "svg", "css", "xml", "html", "txt"}
// reRefs: ссылки [[Имя]] в контенте (вне блоков кода)
var reRefs, reFence = regexp.MustCompile(`\[\[([^\]]+)\]\]`), regexp.MustCompile("(?s)```.*?```")
// reIncl: трансклюзия {{Имя}} в контенте (вне блоков кода)
var reIncl = regexp.MustCompile(`\{\{([^{}]+)\}\}`)
// refKeys: параметры концепта, в значениях которых — имена концептов
var refKeys = []string{"Alias", "Attrib", "Preposit", "Prfix", "Sufix", "Radix", "Taxon", "Links", "Defs", "I-process", "I-sort"}
// reToken, reRefMark: слово имени и ссылка или трансклюзия с именем
var reToken, reRefMark = regexp.MustCompile(`[\p{L}\p{N}_-]+`), regexp.MustCompile(`(\[\[|\{\{)([^\]\}]+)(\]\]|\}\})`)
// aOrder: место литеры в алфавите темы; пробел, «-», «,» — раньше всех
var aOrder = func() map[rune]int {
m := map[rune]int{' ': -1, '-': -1, ',': -1}
pos := 0
for _, c := range aSort {if c != ' ' {m[c] = pos; pos++}}; return m}()
// linkOrder: связи локального контекста — оси (Prfix, Attrib, Sufix, Preposit), затем Radix, Taxon
var linkOrder = []string{"Prfix", "Attrib", "Sufix", "Preposit", "Radix", "Taxon", "Links"}
// graphTypes: связи графа, затем Radix и вычисляемая Transcl (в списке всегда, на оси — по настройке)
// graphCodes: коды связей в списке под графом; настройки N-west, West, S-west, Nord, List — именами связей
var graphTypes = []string{"Taxon", "Links", "Attrib", "Preposit", "Prfix", "Sufix", "Radix", "Transcl"}
var graphCodes = map[string]string{"Themes": "TM", "Alias": "AL", "Attrib": "AT", "Preposit": "PP", "Prfix": "PF", "Sufix": "SF", "Taxon": "TX", "Links": "LK", "Radix": "RX", "Transcl": "TC"}
// gnode: узел графа — имя, картинка, цвет, соседи (в подсказке); Type theme — узел-тема, Theme — тема соседа
type gnode struct {
	Name      string  `json:"name"`
	Icon      string  `json:"icon,omitempty"`
	Color     string  `json:"color,omitempty"`
	Type      string  `json:"type,omitempty"`
	Theme     string  `json:"theme,omitempty"`
	Neighbors []gnode `json:"neighbors,omitempty"`}
// dicts: указатели словарей с формами слов — объекты темы и атрибуты (lexic/atrbt)
type dicts struct{ obj, atr map[string]string }
