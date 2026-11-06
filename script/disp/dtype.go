// epi/script/disp/dtype.go
package disp
import ("epi/script/format")
// Prep — интерпретатор препозиции. ok=false: для таких аргументов
// интерпретации нет (предикация уходит в базу нераспознанных).
type Prep func(a, b *format.Value) (format.Value, bool)
// Interp — среда интерпретации.
type Interp struct {
	Frames []*Frame // фреймы-интерпретаторы препозиций (9.11)
	Used   []string // фреймы, интерпретировавшие последнюю Run
	// OnParam: «Фрейм.Параметр : …» записано (фреймы автора сохраняются в теме)
	OnParam func(frame, param string, pred []format.Value)
	// OnAlias: «имя : препозиция» записано (алиасы — системный лексикон, 9.15)
	OnAlias func(name, target string)
	// OnFrame: «Фрейм : [ … ]» — тело фрейма записано (импликации концепта)
	OnFrame func(name string, body []format.Value)
	Limit   int // лимит шагов (0.16)
	Ask     func(q string) bool
	Unknown []string // база нераспознанных предикаций (канонический текст)
	steps   int
	allowed int}
// Stopped — пользователь не разрешил продолжать после исчерпания лимита.
type Stopped struct{ Steps int }
// Frame — фрейм-интерпретатор (9.11). Kernel — неизменяемая часть:
// препозиции, которые он интерпретирует. Прецеденты интерпретаций во фрейм не
// пишутся (решено 03.11.2026): история — в протоколе.
type Frame struct {
	Name   string
	Kernel map[string]Prep}
// Env — контекст: модели «имя -> предикация». Имена сравниваются по format.Key.
type Env struct {
	vars   map[string]*binding
	parent *Env}
type binding struct {
	pred []format.Value // правая часть модели «имя -> …»
	env  *Env           // контекст, в котором она записана
	// параметры фрейма: «Фрейм.Параметр : …» (9.12); параметров вне фрейма нет
	params map[string]*binding}
var (
	none  = format.Value{Kind: format.Literal, Text: "@none"}
	vtrue = format.Value{Kind: format.Literal, Text: "@true"}
	vfals = format.Value{Kind: format.Literal, Text: "@false"})
type slot struct {
	def    *format.Value  // ключ определения «Ключ : …»
	alias  bool           // «имя : препозиция» — алиас препозиции (9.8)
	prefix *format.Value  // препозиция в начале
	a      []format.Value // операнд: значение или «Терм { аргументы }»
	op     *format.Value
	b      []format.Value
	ok     bool}
type pred struct {
	left, right *slot
	toks        []format.Value
	ok          bool}
type reader struct{ vs []format.Value }
// aliasParam — параметр-синоним концепта (решено 03.11.2026): `X.Alias : Y`
// или `X.Alias : [ Y Z ]`; тогда `Y` обозначает тот же фрейм, что `X`.
const aliasParam = "Alias"
