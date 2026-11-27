// epi/script/disp/assoc.go
// assoc: интерпретатор ассоциативных массивов — объектный блок { Ключ : значение }: valut, lkeis, parad, annul; равенство без учёта порядка пар;
// json: jsnin — текст json в значение, injsn — значение в текст json
package disp
import ("encoding/json"; "strings"; "epi/script/format")
// assocFrame: фрейм assoc; ключ — строка ("B"), ключа нет — @none; parad — новый объект, исходный не меняется; parad — объект или пара [ K V ]; annul — новый объект без пары, ключа нет — объект как есть
func assocFrame() *Frame {
key := func(b *format.Value) (string, bool) {
	if b == nil || (b.Kind != format.String && b.Kind != format.Term) {return "", false}; return b.Text, true}
k := map[string]Prep{
	"valut": func(a, b *format.Value) (format.Value, bool) {
		kk, ok := key(b)
		if a == nil || a.Kind != format.Object || !ok {return none, false}
		for _, e := range a.Entries() {if format.SameTerm(e.Key.Text, kk) {
			if len(e.Value) == 1 {return e.Value[0], true}; return format.Value{Kind: format.Expr, Items: e.Value}, true}}; return none, true},
	"lkeis": func(a, b *format.Value) (format.Value, bool) {
		if a != nil || b == nil || b.Kind != format.Object {return none, false}
		out := format.Value{Kind: format.Array, Items: []format.Value{}}
		for _, e := range b.Entries() {out.Items = append(out.Items, format.Value{Kind: format.String, Text: e.Key.Text})}; return out, true},
	"annul": func(a, b *format.Value) (format.Value, bool) {
		kk, ok := key(b)
		if a == nil || a.Kind != format.Object || !ok {return none, false}
		out := format.Value{Kind: format.Object, Items: []format.Value{}}
		for _, e := range a.Entries() {if !format.SameTerm(e.Key.Text, kk) {out.Items = append(append(out.Items, e.Key, format.Value{Kind: format.Op, Text: ":"}), e.Value...)}}; return out, true},
	"jsnin": func(a, b *format.Value) (format.Value, bool) {
		s, ok := text(b)
		if a != nil || !ok {return none, false}
		d := json.NewDecoder(strings.NewReader(s))
		d.UseNumber()
		v, err := jsonValue(d)
		if err != nil {return none, false}; return v, true},
	"injsn": func(a, b *format.Value) (format.Value, bool) {
		if a != nil || b == nil {return none, false}
		x, ok := jsonAny(*b)
		if !ok {return none, false}
		out, err := json.MarshalIndent(x, "", "  ")
		if err != nil {return none, false}; return format.Value{Kind: format.String, Text: string(out)}, true},
	"parad": func(a, b *format.Value) (format.Value, bool) {
		if a == nil || b == nil || a.Kind != format.Object {return none, false}
		if b.Kind == format.Array {
			kk, ok := key(func() *format.Value {if len(b.Items) == 2 {return &b.Items[0]}; return nil}())
			if !ok || kk == "" {return none, false}
			p := format.Value{Kind: format.Object, Items: []format.Value{clavis(kk), {Kind: format.Op, Text: ":"}, b.Items[1]}}; b = &p}
		if b.Kind != format.Object {return none, false}
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
// clavis: ключ пары — терм, если текст читается как терм, иначе строка («"7" : …»)
func clavis(s string) format.Value {
if vs, err := format.Parse(s); err == nil && len(vs) == 1 && vs[0].Kind == format.Term {return vs[0]}; return format.Value{Kind: format.String, Text: s} }
// jsonValue: текст json в значение — объект с порядком пар, массив, строка, число, true и false — @true и @false, null — @none
func jsonValue(d *json.Decoder) (format.Value, error) {
t, err := d.Token()
if err != nil {return none, err}
switch x := t.(type) {
case json.Delim:
	if x == '[' {
		out := format.Value{Kind: format.Array, Items: []format.Value{}}
		for d.More() {v, err := jsonValue(d); if err != nil {return none, err}; out.Items = append(out.Items, v)}
		_, err := d.Token(); return out, err}
	out := format.Value{Kind: format.Object, Items: []format.Value{}}
	for d.More() {
		kt, err := d.Token()
		if err != nil {return none, err}
		k := kt.(string)
		v, err := jsonValue(d)
		if err != nil {return none, err}
		out.Items = append(out.Items, clavis(k), format.Value{Kind: format.Op, Text: ":"}, v)}
	_, err := d.Token(); return out, err
case string: return format.Value{Kind: format.String, Text: x}, nil
case json.Number:
	if _, err := x.Int64(); err == nil {return format.Value{Kind: format.Integer, Text: x.String()}, nil}; return format.Value{Kind: format.Decimal, Text: x.String()}, nil
case bool: return flag(x), nil}; return none, nil }
// jsonAny: значение для записи json — объект словарём (ключи по порядку, как пишет Go), остальное — как есть; ok — значение записывается в json
func jsonAny(v format.Value) (any, bool) {
switch v.Kind {
case format.Object:
	m := map[string]any{}
	for _, e := range v.Entries() {
		if len(e.Value) != 1 {return nil, false}
		x, ok := jsonAny(e.Value[0])
		if !ok {return nil, false}
		m[e.Key.Text] = x}; return m, true
case format.Array:
	l := []any{}
	for _, it := range v.Items {x, ok := jsonAny(it); if !ok {return nil, false}; l = append(l, x)}; return l, true
case format.String, format.LongString, format.Term: return v.Text, true
case format.Integer, format.Decimal: return json.Number(v.Text), true
case format.Literal:
	switch v.Text {case "@true": return true, true; case "@false": return false, true; case "@none": return nil, true}}; return nil, false }
