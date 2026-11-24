// epi/script/disp/rescr.go
// rescr: интерпретатор переписывания — «rescr Фрейм [ предикация ]»: предикация как данные сопоставляется с образцами тела фрейма (9.4, 9.5)
package disp
import ("fmt"; "strings"; "epi/script/format")
// rescr: правило тела «образец -> замена» по порядку, срабатывает первое совпавшее; место образца — роль фрейма «Фрейм.Роль»:
// роль In — одно значение, роль Conseq — последовательность (в том числе пустая), роль дважды — одно и то же;
// терм без пометки — буквально; результат — замена блоком [ … ]; не совпало — вопрос, у фрейма с Idem : @true — предикация как есть
func (in *Interp) rescr(s *slot, env *Env) (format.Value, error) {
fail := func(why string) (format.Value, error) {in.unknown(s.toks(), "— rescr "+why); return none, nil}
if s.prefix == nil || s.op != nil || len(s.a) != 2 || s.a[0].Kind != format.Term || s.a[1].Kind != format.Array {return fail("пишется «rescr Фрейм [ предикация ]»")}
name := s.a[0].Text
b, _ := env.findFrame(name)
if b == nil || len(b.pred) != 1 || b.pred[0].Kind != format.Array {return fail("— у «" + name + "» нет тела с правилами")}
vars := map[string]bool{}
for _, r := range roleNames(b, "In") {vars[format.Key(r)] = false}
for _, r := range roleNames(b, "Conseq") {vars[format.Key(r)] = true}
role := func(v format.Value) (string, bool, bool) {
	if v.Kind != format.Field {return "", false, false}
	f, r, ok := strings.Cut(v.Text, ".")
	if !ok || !format.SameTerm(f, name) {return "", false, false}
	seq, ok := vars[format.Key(r)]; return format.Key(r), seq, ok}
_, fr := in.prepOf("rescr")
for _, seg := range segments(b.pred[0].Items) {
	if err := in.step(); err != nil {return none, err}
	i := 0
	for i < len(seg) && !isModel(seg[i]) {i++}
	if i == len(seg) {continue}
	bind := map[string][]format.Value{}
	if !match(unwrap(seg[:i]), s.a[1].Items, bind, role) {continue}
	if fr != nil {in.use(fr)}; return format.Value{Kind: format.Array, Items: subst(unwrap(seg[i+1:]), bind, role)}, nil}
if pb := b.params[format.Key("Idem")]; pb != nil && len(pb.pred) == 1 && pb.pred[0].Kind == format.Literal && pb.pred[0].Text == "@true" {
	if fr != nil {in.use(fr)}; return s.a[1], nil}
return fail(fmt.Sprintf("— ни одно правило «%s» не подошло к %s", name, format.Canon([]format.Value{s.a[1]}))) }
// unwrap: слот из одной группы ( … ) — её содержимое: скобки только группируют слот
func unwrap(vs []format.Value) []format.Value {
if len(vs) == 1 && vs[0].Kind == format.Expr {return vs[0].Items}; return vs }
// match: образец pat против значений vs с перебором длины последовательностей; bind — связанные роли
func match(pat, vs []format.Value, bind map[string][]format.Value, role func(format.Value) (string, bool, bool)) bool {
if len(pat) == 0 {return len(vs) == 0}
p := pat[0]
if k, seq, ok := role(p); ok {
	if got, done := bind[k]; done {
		if len(got) > len(vs) || !aequa(got, vs[:len(got)]) {return false}; return match(pat[1:], vs[len(got):], bind, role)}
	lo, hi := 1, 1
	if seq {lo, hi = 0, len(vs)}
	for n := lo; n <= hi && n <= len(vs); n++ {
		bind[k] = vs[:n]
		if match(pat[1:], vs[n:], bind, role) {return true}}
	delete(bind, k); return false}
if len(vs) == 0 {return false}
v := vs[0]
switch p.Kind {
case format.Array, format.Object, format.Expr:
	if v.Kind != p.Kind {return false}
	saved := map[string][]format.Value{}
	for k, x := range bind {saved[k] = x}
	if !match(p.Items, v.Items, bind, role) {for k := range bind {if _, ok := saved[k]; !ok {delete(bind, k)}}; return false}
default:
	if !aequa([]format.Value{p}, []format.Value{v}) {return false}}
return match(pat[1:], vs[1:], bind, role) }
// aequa: значения равны — термы без учёта регистра (кроме первой литеры), остальное — каноническим текстом
func aequa(a, b []format.Value) bool {
if len(a) != len(b) {return false}
for i := range a {
	if (a[i].Kind == format.Term || a[i].Kind == format.Field) && a[i].Kind == b[i].Kind {if !format.SameTerm(a[i].Text, b[i].Text) {return false}; continue}
	if format.Canon(a[i:i+1]) != format.Canon(b[i:i+1]) {return false}}; return true }
// subst: замена — роли заменяются связанными значениями, последовательность вставляется на место роли
func subst(vs []format.Value, bind map[string][]format.Value, role func(format.Value) (string, bool, bool)) []format.Value {
var out []format.Value
for _, v := range vs {
	if k, _, ok := role(v); ok {if got, done := bind[k]; done {out = append(out, got...); continue}}
	if v.Kind == format.Array || v.Kind == format.Object || v.Kind == format.Expr {v.Items = subst(v.Items, bind, role)}
	out = append(out, v)}; return out }
