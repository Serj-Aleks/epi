// epi/script/disp/dtype.go
package disp
import ("epi/script/format")
// Prep: интерпретатор препозиции; ok=false — для таких аргументов интерпретации нет (в базу нераспознанных)
type Prep func(a, b *format.Value) (format.Value, bool)
// Interp: среда интерпретации
type Interp struct {
	Frames []*Frame // фреймы-интерпретаторы препозиций
	Used   []string // фреймы, интерпретировавшие последнюю Run
	// OnParam: «Фрейм.Параметр : …» записано (фреймы автора сохраняются в теме); ошибка — ответ предикации
	OnParam func(frame, param string, pred []format.Value) error
	// OnAlias: «имя : препозиция» записано (алиасы — системный лексикон)
	OnAlias func(name, target string)
	// OnFrame: «Фрейм : [ … ]» — тело фрейма записано (импликации концепта)
	OnFrame func(name string, body []format.Value)
	// Raw: препозиции ядра, получающие операнд без вычисления (nexus Цвет — имя концепта)
	Raw map[string]func(arg format.Value) (format.Value, bool)
	RawErr error // причина отказа Raw: ответ — ошибка, а не «не распознана — записать?»
	Limit   int // лимит шагов
	Ask     func(q string) bool
	Unknown []string // база нераспознанных предикаций (канонический текст)
	steps   int
	allowed int}
// Stopped: пользователь не разрешил продолжать после исчерпания лимита
type Stopped struct{ Steps int }
// Frame: фрейм-интерпретатор, Kernel — препозиции, которые он интерпретирует (история — в протоколе)
type Frame struct {
	Name   string
	Kernel map[string]Prep}
// Env: контекст — модели «имя -> предикация», имена сравниваются по format.Key
type Env struct {
	vars   map[string]*binding
	parent *Env}
type binding struct {
	pred []format.Value // правая часть модели «имя -> …»
	env  *Env           // контекст, в котором она записана
	// параметры фрейма: «Фрейм.Параметр : …»; параметров вне фрейма нет
	params map[string]*binding}
var (
	none  = format.Value{Kind: format.Literal, Text: "@none"}
	vtrue = format.Value{Kind: format.Literal, Text: "@true"}
	vfals = format.Value{Kind: format.Literal, Text: "@false"})
type slot struct {
	def    *format.Value  // ключ определения «Ключ : …»
	alias  bool           // «имя : препозиция» — алиас препозиции
	prefix *format.Value  // препозиция в начале
	a      []format.Value // операнд: значение или «Терм { аргументы }»
	op     *format.Value
	b      []format.Value
	ok     bool}
type pred struct {
	left, right *slot
	toks        []format.Value
	ok          bool}
type reader struct{ vs []format.Value; raw map[string]bool } // raw — препозиции с невычисляемым операндом (Interp.Raw)
// aliasParam: параметр-синоним — X.Alias : Y или X.Alias : [ Y Z ], тогда Y — тот же фрейм, что X
const aliasParam = "Alias"
