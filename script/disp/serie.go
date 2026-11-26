// epi/script/disp/serie.go
// serie: интерпретатор рядов — строка (ряд литер) и список (ряд элементов): dstnz, front, post, elmnt, cnctn, cnvrg, reduc, order;
// строки: index (и в списке), quant, combi, clean, mutnt, divrg; число ↔ текст: nitxt, txnin
package disp
import ("sort"; "strconv"; "strings"; "unicode/utf8"; "epi/script/format")
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
	"index": func(a, b *format.Value) (format.Value, bool) {
		if a == nil || b == nil {return none, false}
		if a.Kind == format.Array {for i, it := range a.Items {if congr(it, *b) {return format.Value{Kind: format.Integer, Text: strconv.Itoa(i + 1)}, true}}; return format.Value{Kind: format.Integer, Text: "0"}, true}
		s, ok1 := text(a); t, ok2 := text(b)
		if !ok1 || !ok2 {return none, false}
		i := strings.Index(s, t)
		if i >= 0 {i = utf8.RuneCountInString(s[:i]) + 1} else {i = 0}; return format.Value{Kind: format.Integer, Text: strconv.Itoa(i)}, true},
	"quant": func(a, b *format.Value) (format.Value, bool) {
		s, ok1 := text(a); t, ok2 := text(b)
		if !ok1 || !ok2 {return none, false}
		out := format.Value{Kind: format.Array, Items: []format.Value{}}
		for _, p := range strings.Split(s, t) {out.Items = append(out.Items, format.Value{Kind: format.String, Text: p})}; return out, true},
	"combi": func(a, b *format.Value) (format.Value, bool) {
		t, ok := text(b)
		if a == nil || a.Kind != format.Array || !ok {return none, false}
		var ps []string
		for i := range a.Items {p, ok := text(&a.Items[i]); if !ok {return none, false}; ps = append(ps, p)}; return format.Value{Kind: format.String, Text: strings.Join(ps, t)}, true},
	"clean": func(a, b *format.Value) (format.Value, bool) {
		s, ok := text(b)
		if a != nil || !ok {return none, false}; return format.Value{Kind: format.String, Text: strings.TrimSpace(s)}, true},
	"mutnt": func(a, b *format.Value) (format.Value, bool) {
		s, ok := text(a)
		if !ok || b == nil || b.Kind != format.Array || len(b.Items) != 2 {return none, false}
		o, ok1 := text(&b.Items[0]); n, ok2 := text(&b.Items[1])
		if !ok1 || !ok2 || o == "" {return none, false}; return format.Value{Kind: format.String, Text: strings.ReplaceAll(s, o, n)}, true},
	"divrg": func(a, b *format.Value) (format.Value, bool) {
		s, ok := text(b)
		if a != nil || !ok {return none, false}; return format.Value{Kind: format.String, Text: strings.ToUpper(s)}, true},
	"nitxt": func(a, b *format.Value) (format.Value, bool) {
		if a != nil || b == nil || (b.Kind != format.Integer && b.Kind != format.Decimal) {return none, false}; return format.Value{Kind: format.String, Text: b.Text}, true},
	"txnin": func(a, b *format.Value) (format.Value, bool) {
		s, ok := text(b)
		if a != nil || !ok {return none, false}
		s = strings.TrimSpace(s)
		if _, err := strconv.Atoi(s); err == nil {return format.Value{Kind: format.Integer, Text: s}, true}
		if f, err := strconv.ParseFloat(s, 64); err == nil {return format.Value{Kind: format.Decimal, Text: strconv.FormatFloat(f, 'f', -1, 64)}, true}; return none, false},
	"reduc": func(a, b *format.Value) (format.Value, bool) {return none, false},
	"order": func(a, b *format.Value) (format.Value, bool) {return none, false},
	"cnvrg": func(a, b *format.Value) (format.Value, bool) {
		if a != nil || b == nil || (b.Kind != format.String && b.Kind != format.LongString) {return none, false}; return format.Value{Kind: format.String, Text: strings.ToLower(b.Text)}, true},}
return &Frame{Name: "serie", Kernel: k} }
// reduc: «reduc [ … ]» — список из значений элементов блока, каждый вычислен в контексте вызова (как reduce в Ребол)
func (in *Interp) reduc(s *slot, env *Env) (format.Value, error) {
if s.prefix == nil || s.op != nil || len(s.a) != 1 || s.a[0].Kind != format.Array {in.unknown(s.toks(), "— reduc пишется «reduc [ … ]»"); return none, nil}
out := format.Value{Kind: format.Array, Items: []format.Value{}}
for _, it := range s.a[0].Items {
	v, err := in.evalOperand([]format.Value{it}, env, false)
	if err != nil {return none, err}
	out.Items = append(out.Items, v)}
if _, fr := in.prepOf("reduc"); fr != nil {in.use(fr)}; return out, nil }
// order: «L order Фрейм» — список L, упорядоченный фреймом сравнения с двумя ролями In: @true — первый раньше второго;
// порядок равных сохраняется; фрейм дал не флаг — не распознано
func (in *Interp) order(s *slot, env *Env) (format.Value, error) {
fail := func(why string) (format.Value, error) {in.unknown(s.toks(), "— order: "+why); return none, nil}
if s.prefix != nil || s.op == nil || len(s.b) != 1 || s.b[0].Kind != format.Term {return fail("пишется «Список order Фрейм»")}
name := s.b[0].Text
fb, _ := env.findFrame(name)
if fb == nil || len(roleNames(fb, "In")) != 2 {return fail("«" + name + "» — нужен фрейм сравнения с двумя ролями In")}
l, err := in.evalOperand(s.a, env, false)
if err != nil {return none, err}
if l.Kind != format.Array {return fail("слева нужен список")}
items := append([]format.Value{}, l.Items...)
var bad error
notFlag := false
sort.SliceStable(items, func(i, j int) bool {
	if bad != nil || notFlag {return false}
	v, err := in.call(name, fb, format.Value{Kind: format.Array, Items: []format.Value{items[i], items[j]}}, env)
	if err != nil {bad = err; return false}
	if v.Kind != format.Literal || (v.Text != "@true" && v.Text != "@false") {notFlag = true; return false}
	return v.Text == "@true"})
if bad != nil {return none, bad}
if notFlag {return fail("«" + name + "» дал не флаг")}
in.use(&Frame{Name: name})
if _, fr := in.prepOf("order"); fr != nil {in.use(fr)}; return format.Value{Kind: format.Array, Items: items}, nil }
// text: текст строки (обычной или длинной)
func text(v *format.Value) (string, bool) {
if v == nil || (v.Kind != format.String && v.Kind != format.LongString) {return "", false}; return v.Text, true }
