// epi/script/disp/zycle.go
// zycle: интерпретатор цикла второго уровня — препозиция recur
package disp
import ("strconv"; "epi/script/format")
// recur: цикл recur { In : [ … ] As : Фрейм.Параметр Corpus : [ … ] } — элементы In по очереди в As, Corpus на каждом шаге; In — блок, число N (1…N) или имя, дающее то или другое
// ответ — последнего шага
func (in *Interp) recur(s *slot, env *Env) (format.Value, error) {
fail := func(why string) (format.Value, error) {in.unknown(s.toks(), "— recur "+why); return none, nil}
if s.prefix == nil || s.op != nil || len(s.a) != 1 || s.a[0].Kind != format.Object {return fail("интерпретирует только «recur { In : … As : … Corpus : [ … ] }»")}
var src, as, corpus []format.Value
for _, e := range s.a[0].Entries() {
	switch format.Key(e.Key.Text) {
	case "In": src = e.Value
	case "As": as = e.Value
	case "Corpus": corpus = e.Value}}
if src == nil || len(as) != 1 || as[0].Kind != format.Field || len(corpus) != 1 || corpus[0].Kind != format.Array {return fail("— нужны In, As (Фрейм.Параметр) и Corpus ([ … ])")}
items, err := in.evalOperand(src, env, false)
if err != nil {return none, err}
if f, integer, isNum := num(&items); isNum {
	if !integer || f < 0 || f > float64(in.Limit) {return fail("— In-число: целое от 0 до лимита шагов")}
	items = format.Value{Kind: format.Array}
	for i := 1; i <= int(f); i++ {items.Items = append(items.Items, format.Value{Kind: format.Integer, Text: strconv.Itoa(i)})}}
if items.Kind != format.Array {return fail("— In должен быть блоком [ … ], числом или их именем")}
r := none
for _, it := range items.Items {
	if _, err := in.defParam(&slot{def: &as[0]}, []format.Value{it}, env); err != nil {return none, err}
	if err := in.step(); err != nil {return none, err}
	if r, _, err = in.block(corpus[0].Items, env); err != nil {return none, err}}
in.use(in.FrameOf("zycle")); return r, nil }
