// epi/script/disp/serie.go
// serie: интерпретатор рядов — строка (ряд литер) и список (ряд элементов): dstnz, front, post, elmnt, cnctn, cnvrg
package disp
import ("strconv"; "strings"; "epi/script/format")
// serieOf: ряд значения — литеры строки или элементы списка; str — строка
func serieOf(v *format.Value) (items []format.Value, str bool, ok bool) {
if v == nil {return nil, false, false}
switch v.Kind {
case format.String, format.LongString:
	for _, r := range v.Text {items = append(items, format.Value{Kind: format.String, Text: string(r)})}; return items, true, true
case format.Array: return v.Items, false, true}; return nil, false, false }
// serieVal: ряд как значение — строка или список
func serieVal(items []format.Value, str bool) format.Value {
if !str {return format.Value{Kind: format.Array, Items: append([]format.Value{}, items...)}}
var b strings.Builder
for _, it := range items {b.WriteString(it.Text)}; return format.Value{Kind: format.String, Text: b.String()} }
// serieFrame: фрейм serie; N — целое не меньше 0 (у elmnt — номер с 1), N больше длины — front весь ряд, post пусто, elmnt @none
func serieFrame() *Frame {
count := func(b *format.Value) (int, bool) {f, integer, ok := num(b); return int(f), ok && integer && f >= 0}
cut := func(head bool) Prep {return func(a, b *format.Value) (format.Value, bool) {
		items, str, ok := serieOf(a)
		n, ok2 := count(b)
		if !ok || !ok2 {return none, false}
		n = min(n, len(items))
		if head {return serieVal(items[:n], str), true}; return serieVal(items[n:], str), true}}
k := map[string]Prep{
	"dstnz": func(a, b *format.Value) (format.Value, bool) {
		items, _, ok := serieOf(b)
		if a != nil || !ok {return none, false}; return format.Value{Kind: format.Integer, Text: strconv.Itoa(len(items))}, true},
	"front": cut(true),
	"post":  cut(false),
	"elmnt": func(a, b *format.Value) (format.Value, bool) {
		items, _, ok := serieOf(a)
		n, ok2 := count(b)
		if !ok || !ok2 || n < 1 {return none, false}
		if n > len(items) {return none, true}; return items[n-1], true},
	"cnctn": func(a, b *format.Value) (format.Value, bool) {
		x, xs, ok1 := serieOf(a)
		if !ok1 {return none, false}
		if y, ys, ok2 := serieOf(b); ok2 && xs == ys {return serieVal(append(append([]format.Value{}, x...), y...), xs), true}
		if xs || b == nil {return none, false}; return serieVal(append(append([]format.Value{}, x...), *b), false), true},
	"cnvrg": func(a, b *format.Value) (format.Value, bool) {
		if a != nil || b == nil || (b.Kind != format.String && b.Kind != format.LongString) {return none, false}; return format.Value{Kind: format.String, Text: strings.ToLower(b.Text)}, true},}
return &Frame{Name: "serie", Kernel: k} }
