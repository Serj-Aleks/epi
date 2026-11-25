// epi/script/disp/assoc.go
// assoc: интерпретатор ассоциативных массивов — объектный блок { Ключ : значение }: valor, clavs, parad; равенство без учёта порядка пар
package disp
import ("strings"; "epi/script/format")
// assocFrame: фрейм assoc; ключ — строка или терм-значение ('B), ключа нет — @none; parad — новый объект, исходный не меняется
func assocFrame() *Frame {
key := func(b *format.Value) (string, bool) {
	if b == nil || (b.Kind != format.String && b.Kind != format.Term) {return "", false}; return b.Text, true}
k := map[string]Prep{
	"valor": func(a, b *format.Value) (format.Value, bool) {
		kk, ok := key(b)
		if a == nil || a.Kind != format.Object || !ok {return none, false}
		for _, e := range a.Entries() {if format.SameTerm(e.Key.Text, kk) {
			if len(e.Value) == 1 {return e.Value[0], true}; return format.Value{Kind: format.Expr, Items: e.Value}, true}}; return none, true},
	"clavs": func(a, b *format.Value) (format.Value, bool) {
		if a != nil || b == nil || b.Kind != format.Object {return none, false}
		out := format.Value{Kind: format.Array, Items: []format.Value{}}
		for _, e := range b.Entries() {out.Items = append(out.Items, format.Value{Kind: format.String, Text: e.Key.Text})}; return out, true},
	"parad": func(a, b *format.Value) (format.Value, bool) {
		if a == nil || b == nil || a.Kind != format.Object || b.Kind != format.Object {return none, false}
		es := a.Entries()
		for _, n := range b.Entries() {
			found := false
			for i := range es {if format.SameTerm(es[i].Key.Text, n.Key.Text) {es[i].Value, found = n.Value, true}}
			if !found {es = append(es, n)}}
		out := format.Value{Kind: format.Object, Items: []format.Value{}}
		for _, e := range es {out.Items = append(append(out.Items, e.Key, format.Value{Kind: format.Op, Text: ":"}), e.Value...)}; return out, true},}
return &Frame{Name: "assoc", Kernel: k} }
// congr: структурное равенство — объекты без учёта порядка пар, списки и группы по порядку, остальное — текстом без учёта регистра
func congr(a, b format.Value) bool {
if a.Kind != b.Kind {return false}
switch a.Kind {
case format.Object:
	x, y := a.Entries(), b.Entries()
	if len(x) != len(y) {return false}
	for _, e := range x {
		ok := false
		for _, f := range y {if format.SameTerm(e.Key.Text, f.Key.Text) {ok = congr(format.Value{Kind: format.Expr, Items: e.Value}, format.Value{Kind: format.Expr, Items: f.Value}); break}}
		if !ok {return false}}; return true
case format.Array, format.Expr:
	if len(a.Items) != len(b.Items) {return false}
	for i := range a.Items {if !congr(a.Items[i], b.Items[i]) {return false}}; return true}
return strings.EqualFold(a.String(), b.String()) }
