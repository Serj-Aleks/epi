// epi/script/disp/intrp.go
// intrp: встроенные интерпретаторы arifm, logic, trgnm, lnrfm и системные препозиции; числа и флаги
package disp
import ("math"; "strconv"; "strings"; "epi/script/format")
// num: значение, целое ли, число ли
func num(v *format.Value) (float64, bool, bool) {
	if v == nil {return 0, false, false}
	switch v.Kind {
	case format.Integer:
		n, err := strconv.ParseInt(v.Text, 10, 64); return float64(n), true, err == nil
	case format.Decimal:
		f, err := strconv.ParseFloat(v.Text, 64); return f, false, err == nil}; return 0, false, false}
// number: число как значение — целое или десятичное
func number(f float64, integer bool) format.Value {
if integer && f == math.Trunc(f) && math.Abs(f) < 1<<53 {return format.Value{Kind: format.Integer, Text: strconv.FormatInt(int64(f), 10)}}
s := strconv.FormatFloat(f, 'f', -1, 64)
if !strings.Contains(s, ".") {s += ".0"}; return format.Value{Kind: format.Decimal, Text: s} }
// flag: флаг @true или @false
func flag(b bool) format.Value {
if b {return vtrue}; return vfals }
// baseFrames: фреймы-интерпретаторы системы (serie — serie.go, assoc — assoc.go); минус в начале предикации — унарный, функция одного аргумента — в начале; сравнение строк — по порядку литер (как в Go)
// recur, trans, nexus, qwest, qwizt, clone, rename, import, union — ядро задаёт среда (Interp.recur, Interp.Raw, словари)
func baseFrames() []*Frame {
arith := func(op func(x, y float64) (float64, bool)) Prep {return func(a, b *format.Value) (format.Value, bool) {
		x, xi, ok1 := num(a)
		y, yi, ok2 := num(b)
		if a == nil {x, xi, ok1 = 0, true, true}
		if !ok1 || !ok2 {return none, false}
		r, ok := op(x, y)
		if !ok {return none, false}; return number(r, xi && yi), true}}
cmp := func(op func(x, y float64) bool) Prep {return func(a, b *format.Value) (format.Value, bool) {
		if s, ok1 := text(a); ok1 {if t, ok2 := text(b); ok2 {return flag(op(float64(strings.Compare(s, t)), 0)), true}}
		x, _, ok1 := num(a)
		y, _, ok2 := num(b)
		if !ok1 || !ok2 {return none, false}; return flag(op(x, y)), true}}
eq := func(neg bool) Prep {return func(a, b *format.Value) (format.Value, bool) {
		if a == nil {return none, false}
		x, _, ok1 := num(a)
		y, _, ok2 := num(b)
		e := ok1 && ok2 && x == y || !(ok1 && ok2) && congr(*a, *b); return flag(e != neg), true}}
plusOnlyInfix := func(p Prep) Prep {return func(a, b *format.Value) (format.Value, bool) {if a == nil {return none, false}; return p(a, b)}}
arifm := map[string]Prep{
	"+": plusOnlyInfix(arith(func(x, y float64) (float64, bool) { return x + y, true })),
	"-": arith(func(x, y float64) (float64, bool) { return x - y, true }),
	"*": plusOnlyInfix(arith(func(x, y float64) (float64, bool) { return x * y, true })),
	"/": plusOnlyInfix(func(a, b *format.Value) (format.Value, bool) {
		x, _, ok1 := num(a)
		y, _, ok2 := num(b)
		if !ok1 || !ok2 || y == 0 {return none, false}; return number(x/y, false), true}),}
logic := map[string]Prep{
	"<":  plusOnlyInfix(cmp(func(x, y float64) bool { return x < y })),
	">":  plusOnlyInfix(cmp(func(x, y float64) bool { return x > y })),
	"<=": plusOnlyInfix(cmp(func(x, y float64) bool { return x <= y })),
	">=": plusOnlyInfix(cmp(func(x, y float64) bool { return x >= y })),
	"=":  eq(false),
	"<>": eq(true),}
boolean := func(op func(x, y bool) bool) Prep {return func(a, b *format.Value) (format.Value, bool) {
		x, ok1 := flagOf(a)
		y, ok2 := flagOf(b)
		if !ok1 || !ok2 {return none, false}; return flag(op(x, y)), true}}
logic["&"] = boolean(func(x, y bool) bool { return x && y })
logic["|"] = boolean(func(x, y bool) bool { return x || y })
unary := func(fn func(x float64) (float64, bool)) Prep {return func(a, b *format.Value) (format.Value, bool) {
		x, _, ok := num(b)
		if a != nil || !ok {return none, false}
		r, ok := fn(x)
		if !ok || math.IsNaN(r) || math.IsInf(r, 0) {return none, false}; return number(r, false), true}}
all := func(fn func(float64) float64) func(float64) (float64, bool) {return func(x float64) (float64, bool) { return fn(x), true }}
unit := func(fn func(float64) float64) func(float64) (float64, bool) {return func(x float64) (float64, bool) { return fn(x), x >= -1 && x <= 1 }}
trgnm := map[string]Prep{
	"sin": unary(all(math.Sin)), "cos": unary(all(math.Cos)), "tan": unary(all(math.Tan)),
	"asin": unary(unit(math.Asin)), "acos": unary(unit(math.Acos)), "atan": unary(all(math.Atan)),}
lnrfm := map[string]Prep{
	"ln":  unary(func(x float64) (float64, bool) { return math.Log(x), x > 0 }),
	"exp": unary(all(math.Exp)),}; return []*Frame{{Name: "arifm", Kernel: arifm}, {Name: "logic", Kernel: logic},
	{Name: "trgnm", Kernel: trgnm}, {Name: "lnrfm", Kernel: lnrfm}, serieFrame(), assocFrame(),
	{Name: "zycle", Kernel: map[string]Prep{"recur": func(a, b *format.Value) (format.Value, bool) {return none, false}}},
	{Name: "trans", Kernel: map[string]Prep{"trans": func(a, b *format.Value) (format.Value, bool) {return none, false}}},
	{Name: "nexus", Kernel: map[string]Prep{"nexus": func(a, b *format.Value) (format.Value, bool) {return none, false}}},
	{Name: "qwest", Kernel: map[string]Prep{"qwest": func(a, b *format.Value) (format.Value, bool) {return none, false}}},
	{Name: "clone", Kernel: map[string]Prep{"clone": func(a, b *format.Value) (format.Value, bool) {return none, false}}},
	{Name: "rename", Kernel: map[string]Prep{"rename": func(a, b *format.Value) (format.Value, bool) {return none, false}}},
	{Name: "qwizt", Kernel: map[string]Prep{"qwizt": func(a, b *format.Value) (format.Value, bool) {return none, false}}},
	{Name: "import", Kernel: map[string]Prep{"import": func(a, b *format.Value) (format.Value, bool) {return none, false}}},
	{Name: "union", Kernel: map[string]Prep{"union": func(a, b *format.Value) (format.Value, bool) {return none, false}}},
	{Name: "radix", Kernel: map[string]Prep{"radix": func(a, b *format.Value) (format.Value, bool) {return none, false}}},
	{Name: "rescr", Kernel: map[string]Prep{"rescr": func(a, b *format.Value) (format.Value, bool) {return none, false}}}} }
// flagOf: значение — флаг @true / @false
func flagOf(v *format.Value) (bool, bool) {
if v == nil || v.Kind != format.Literal {return false, false}
switch v.Text {
case "@true": return true, true
case "@false": return false, true}; return false, false }
