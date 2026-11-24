// epi/script/disp/clstr.go
// clstr: классификация строки — текст на предикации, предикация на слоты, операнды и препозиции
package disp
import ("epi/script/format")
// isPrepTok: токен — препозиция: символ (кроме ->, :, .) или терм со строчной
func isPrepTok(v format.Value) bool {
switch v.Kind {
case format.Op: return v.Text != "->" && v.Text != ":" && v.Text != "."
case format.Term: return format.IsPrepTerm(v.Text)}; return false }
// isModel: токен модели ->
func isModel(v format.Value) bool { return v.Kind == format.Op && v.Text == "->" }
// isPrep: токен i — препозиция
func (r reader) isPrep(i int) bool { return i < len(r.vs) && isPrepTok(r.vs[i]) }
// readOperand: операнд с позиции i — значение или «Терм { аргументы }»
func (r reader) readOperand(i int) ([]format.Value, int, bool) {
vs := r.vs
if i >= len(vs) || vs[i].Kind == format.Op || r.isPrep(i) || format.IsDef(vs, i) {return nil, i, false}
if vs[i].Kind == format.Term && i+1 < len(vs) && (vs[i+1].Kind == format.Object || vs[i+1].Kind == format.Array && upperFirst(vs[i].Text) == vs[i].Text) {return vs[i : i+2], i + 2, true}; return vs[i : i+1], i + 1, true }
// readSlot: слот с позиции i — определение (значение с препозиции — алиас препозиции), препозиция в начале или между операндами
// у Raw-препозиции слово со строчной — операнд (qwest numer); в слоте одна препозиция; «Ключ :» в конце — пустое значение;
// «п : Фрейм» — алиас препозиции на объект-фрейм
func (r reader) readSlot(i int) (*slot, int) {
vs := r.vs
if format.IsDef(vs, i) {
	key := vs[i]
	if i+2 >= len(vs) {return &slot{def: &key, ok: true}, i + 2}
	if r.isPrep(i + 2) {return &slot{def: &key, a: vs[i+2 : i+3], alias: true, ok: true}, i + 3}
	if format.IsPrepTerm(key.Text) && i+3 == len(vs) && vs[i+2].Kind == format.Term && !format.IsPrepTerm(vs[i+2].Text) {return &slot{def: &key, a: vs[i+2 : i+3], alias: true, ok: true}, i + 3}
	inner, j := r.readSlot(i + 2)
	inner.def = &key; return inner, j}
s := &slot{}
if r.isPrep(i) {
	p := vs[i]
	s.prefix = &p
	i++
	if p.Kind == format.Term && r.raw[p.Text] && i+1 == len(vs) && vs[i].Kind == format.Term {s.a, s.ok = vs[i:i+1], true; return s, i + 1}}
a, j, ok := r.readOperand(i)
if !ok {return s, j}
s.a, i = a, j
if r.isPrep(i) {
	if s.prefix != nil {return s, i + 1}
	o := vs[i]
	s.op = &o
	b, j, ok := r.readOperand(i + 1)
	if !ok {return s, j}
	s.b, i = b, j
	if r.isPrep(i) {return s, i + 1}}
s.ok = true; return s, i }
// segments: последовательность на предикации — граница « . » или перевод строки, пустые части пропускаются
func segments(vs []format.Value) [][]format.Value {
var segs [][]format.Value
start := 0
cut := func(end int) {if end > start {segs = append(segs, vs[start:end])}}
for i, v := range vs {
	switch {
	case v.Kind == format.Op && v.Text == ".":
		cut(i)
		start = i + 1
	case i > start && v.Line > vs[i-1].End && vs[i-1].End > 0:
		cut(i)
		start = i}}
cut(len(vs)); return segs }
// Split: текст на предикации верхнего уровня в каноническом виде (на каждую — свой ответ)
func Split(text string) ([]string, error) {
vs, err := format.Parse(text)
if err != nil {return nil, err}
var out []string
for _, seg := range segments(vs) {out = append(out, format.Canon(seg))}; return out, nil }
// readPred: предикация из части seg — ровно одна, иначе не распознана
func readPred(seg []format.Value, raw map[string]bool) pred {
r := reader{vs: seg, raw: raw}
p := pred{toks: seg}
var i int
p.left, i = r.readSlot(0)
p.ok = p.left.ok
if p.ok && i < len(seg) && isModel(seg[i]) {
	p.right, i = r.readSlot(i + 1)
	p.ok = p.right.ok}
if i != len(seg) {p.ok = false}; return p }
// toks: токены слота
func (s *slot) toks() []format.Value {
var t []format.Value
if s.def != nil {t = append(t, *s.def, format.Value{Kind: format.Op, Text: ":"})}
if s.prefix != nil {t = append(t, *s.prefix)}
t = append(t, s.a...)
if s.op != nil {t = append(append(t, *s.op), s.b...)}; return t }
