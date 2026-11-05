// epi/script/format/ftype.go
package format
import ("regexp")
// Kind — вид значения.
type Kind int
const (
	Array      Kind = iota // [ ]
	Object                 // { }
	Expr                   // ( )
	Integer                // 123
	Decimal                // 1.23
	Money                  // $1.23
	Tuple                  // 1.2.3
	Range                  // 1..20
	Coord                  // 12x340
	Hex                    // #1A2B
	Date                   // 2021-4-5, 2021-4-5/10:23:45, 2021-4-5/10:23:45+2:00
	Time                   // 10:23:45
	String                 // "abc"
	LongString             // ''abc''
	Tag                    // <abc>
	File                   // %abc
	URL                    // abc://
	Email                  // ab@cde
	Binary16               // #"a1b2"
	Binary64               // ##"atbf"
	Term                   // abc
	GetTerm                // :abc
	LitTerm                // 'abc
	Selector               // .abc
	Refinement             // /abc
	Literal                // @abc
	Field                  // abc.def
	GetField               // :abc.def
	Path                   // abc/def
	Op) // + - * / < > ... ->
var kindNames = [...]string{"массив", "объект", "выражение", "целое", "десятичное",
	"деньги", "кортеж", "диапазон", "координаты", "шестнадцатеричное", "дата", "время",
	"строка", "длинная строка", "тег", "файл", "URL", "email", "двоичное-16",
	"двоичное-64", "терм", "терм-получение", "ссылка на терм",
	"селектор", "уточнение", "литерал", "поле", "получение поля",
	"путь", "оператор"}
// Value — узел дерева. У блоков заполнено Items, у остальных — Text:
// написание из исходного текста (у строк — содержимое без кавычек).
type Value struct {
	Kind  Kind
	Text  string
	Items []Value
	Line  int
	Col   int
	End   int} // строка, на которой значение кончается
// Error — ошибка разбора с местом в тексте.
type Error struct {
	Line, Col int
	Msg       string}
// Operators — операторы: всегда отдельный токен.
var Operators = map[string]bool{"+": true, "-": true, "*": true, "/": true, "//": true,
	"%": true, ":": true, ".": true, "<": true, ">": true, "<=": true, ">=": true, "=": true, "<>": true, "->": true,
	"&": true, "|": true}
// Терм начинается с буквы любого алфавита (решение 29.10.2026); системный
// лексикон — общеупотребительные символы и термы латино-эллинской этимологии.
const termRe = `\pL[\pL\pN_?!*+~&|-]*`
var atoms = []struct {
	kind Kind
	re   *regexp.Regexp}{
	{Money, regexp.MustCompile(`^-?\$[0-9]+(\.[0-9]+)?$`)},
	{Hex, regexp.MustCompile(`^#[0-9A-Fa-f]+$`)},
	{Date, regexp.MustCompile(`^[0-9]{1,4}-[0-9]{1,2}-[0-9]{1,2}(/[0-9]{1,2}:[0-9]{2}(:[0-9]{2}(\.[0-9]+)?)?([+-][0-9]{1,2}:[0-9]{2})?)?$`)},
	{Time, regexp.MustCompile(`^-?[0-9]+:[0-9]{2}(:[0-9]{2}(\.[0-9]+)?)?$`)},
	{Range, regexp.MustCompile(`^-?[0-9]+\.\.-?[0-9]+$`)},
	{Coord, regexp.MustCompile(`^-?[0-9]+x-?[0-9]+$`)},
	{Tuple, regexp.MustCompile(`^[0-9]+(\.[0-9]+){2,}$`)},
	{Decimal, regexp.MustCompile(`^-?[0-9]+\.[0-9]+([eE][+-]?[0-9]+)?$`)},
	{Integer, regexp.MustCompile(`^-?[0-9]+$`)},
	{URL, regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]*://[^\s]*$`)},
	{Email, regexp.MustCompile(`^[^@\s"<>]+@[^@\s"<>]+$`)},
	{File, regexp.MustCompile(`^%[^\s]+$`)},
	{Literal, regexp.MustCompile(`^@` + termRe + `$`)},
	{LitTerm, regexp.MustCompile(`^'` + termRe + `$`)},
	{Selector, regexp.MustCompile(`^\.` + termRe + `$`)},
	{Refinement, regexp.MustCompile(`^/` + termRe + `$`)},
	{GetField, regexp.MustCompile(`^:` + termRe + `(\.` + termRe + `)+$`)},
	{Field, regexp.MustCompile(`^` + termRe + `(\.` + termRe + `)+$`)},
	{Path, regexp.MustCompile(`^` + termRe + `(/` + termRe + `)+$`)},
	{GetTerm, regexp.MustCompile(`^:` + termRe + `$`)},
	{Term, regexp.MustCompile(`^` + termRe + `$`)},}
type parser struct {
	src       []rune
	pos       int
	line, col int}
// Entry — определение объектного блока: ключ и значение (одно или несколько).
type Entry struct {
	Key   Value
	Value []Value}
