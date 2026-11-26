// epi/script/disp/errat.go
// errat: интерпретатор ошибок — errat T: поднять ошибку с текстом T (вычисление прекращается вверх по вызовам, как return err в Go);
// cavet [ предикация ]: выполнить с перехватом — пара [ значение @none ] или [ @none "текст ошибки" ]; нераспознанное внутри — тоже ошибка, без вопроса
package disp
import ("errors"; "epi/script/format")
// erratFrame: фрейм errat; ядро — в errat и cavet (нужен контекст)
func erratFrame() *Frame {
stub := func(a, b *format.Value) (format.Value, bool) {return none, false}
return &Frame{Name: "errat", Kernel: map[string]Prep{"errat": stub, "cavet": stub}} }
// raise: «errat T» — ошибка с текстом T (строка); имя не errat — занято errAt (format, правило 25)
func (in *Interp) raise(s *slot, env *Env) (format.Value, error) {
if s.prefix == nil || s.op != nil {in.unknown(s.toks(), "— errat пишется в начале: errat \"что не так и что делать\""); return none, nil}
v, err := in.evalOperand(s.a, env, false)
if err != nil {return none, err}
t, ok := text(&v)
if !ok {in.unknown(s.toks(), "— errat: текст ошибки — строка"); return none, nil}
if _, fr := in.prepOf("errat"); fr != nil {in.use(fr)}; return none, errors.New(t) }
// cavet: «cavet [ предикация ]» — пара [ значение @none ] или [ @none "текст" ]; ошибка Stopped (лимит) не перехватывается
func (in *Interp) cavet(s *slot, env *Env) (v format.Value, err error) {
if s.prefix == nil || s.op != nil || len(s.a) != 1 || s.a[0].Kind != format.Array {in.unknown(s.toks(), "— cavet пишется в начале с блоком: cavet [ предикация ]"); return none, nil}
if _, fr := in.prepOf("cavet"); fr != nil {in.use(fr)}
pair := func(x, e format.Value) format.Value {return format.Value{Kind: format.Array, Items: []format.Value{x, e}}}
fail := func(msg string) format.Value {return pair(none, format.Value{Kind: format.String, Text: msg})}
in.caveo++
defer func() {
	in.caveo--
	if r := recover(); r != nil {
		c, ok := r.(caught)
		if !ok {panic(r)}
		v, err = fail(c.msg), nil}}()
x, _, e := in.block(s.a[0].Items, env)
var stop Stopped
if errors.As(e, &stop) {return none, e}
if e != nil {return fail(e.Error()), nil}; return pair(x, none), nil }
