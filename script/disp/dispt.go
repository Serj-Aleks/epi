// epi/script/disp/dispt.go
// Package disp — диспетчер предикаций: предикация читается из дерева ASON, форму задаёт препозиция (имя интерпретатора)
// препозиции — символы (+, >) и термы со строчной, объекты — с прописной, «Ключ : значение» — определение
// нераспознанная предикация — не ошибка, она предлагается к записи в базу
package disp
import ("fmt"; "epi/script/format")
// Error: текст остановки по лимиту
func (s Stopped) Error() string {return fmt.Sprintf("остановлено пользователем после %d шагов", s.Steps) }
// FrameOf: фрейм-интерпретатор по имени
func (in *Interp) FrameOf(name string) *Frame {
for _, f := range in.Frames {if f.Name == name {return f}}; return nil }
// prepOf: интерпретатор символа препозиции и его фрейм
func (in *Interp) prepOf(sym string) (Prep, *Frame) {
for _, f := range in.Frames {if p, ok := f.Kernel[sym]; ok {return p, f}}; return nil, nil }
// use: фрейм в списке интерпретировавших
func (in *Interp) use(f *Frame) {
for _, n := range in.Used {if n == f.Name {return}}
in.Used = append(in.Used, f.Name) }
// New: среда с фреймами-интерпретаторами (имена — пять литер)
func New(limit int, ask func(string) bool) *Interp {return &Interp{Frames: baseFrames(), Limit: limit, Ask: ask, allowed: limit} }
// NewEnv: контекст, дочерний к parent
func NewEnv(parent *Env) *Env { return &Env{vars: map[string]*binding{}, parent: parent} }
// find: связь имени в иерархии контекстов и её контекст
func (e *Env) find(name string) (*binding, *Env) {
for c := e; c != nil; c = c.parent {if b, ok := c.vars[format.Key(name)]; ok {return b, c}}; return nil, nil }
// set: имя, связанное в иерархии контекстов, перевязывается там, новое — в текущем контексте
func (e *Env) set(name string, b *binding) {
if _, c := e.find(name); c != nil {c.vars[format.Key(name)] = b; return}
e.vars[format.Key(name)] = b }
// Run: прочитать текст и интерпретировать в контексте env (лимит шагов — на одну интерпретацию)
func (in *Interp) Run(text string, env *Env) (format.Value, error) {
in.Used = nil
in.steps, in.allowed = 0, in.Limit
vs, err := format.Parse(text)
if err != nil {return none, err}
v, _, err := in.block(vs, env); return v, err }
// block: предикации по порядку; модель с истинным левым слотом завершает блок интерпретацией правого, с ложным — дальше
// иначе результат — последняя
func (in *Interp) block(vs []format.Value, env *Env) (format.Value, bool, error) {
res := none
for _, seg := range segments(vs) {
	p := readPred(seg, in.rawNames())
	if err := in.step(); err != nil {return none, false, err}
	if !p.ok {
		in.unknown(p.toks, "не распознана"+caseHint(p.toks, env))
		res = none; continue}
	if p.right == nil {
		v, err := in.evalSlot(p.left, env, false)
		if err != nil {return none, false, err}
		res = v; continue}
	f, err := in.evalSlot(p.left, env, false)
	if err != nil {return none, false, err}
	switch f.Text {
	case "@true":
		v, err := in.evalSlot(p.right, env, true); return v, true, err
	case "@false":
		res = none
	default: return none, false, fmt.Errorf("строка %d: левый слот модели дал не флаг: %s", p.toks[0].Line, f)}}; return res, false, nil }
// step: шаг интерпретации; лимит исчерпан — вопрос, продолжать ли
func (in *Interp) step() error {
in.steps++
if in.Limit > 0 && in.steps > in.allowed {if in.Ask != nil && in.Ask(fmt.Sprintf("исчерпан лимит %d шагов. Продолжать?", in.Limit)) {in.allowed += in.Limit; return nil}; return Stopped{in.steps - 1}}; return nil }
// unknown: нераспознанная предикация — вопрос «записать?» и запись в базу
func (in *Interp) unknown(toks []format.Value, why string) {
text := format.Canon(toks)
if in.Ask != nil && in.Ask(fmt.Sprintf("предикация «%s» %s. Записать её?", text, why)) {in.Unknown = append(in.Unknown, text)} }
// Unreadable: текст не прочитан — тоже предикация без обработчика: вопрос «записать?» и запись в базу
func (in *Interp) Unreadable(text string, err error) {
if in.Ask != nil && in.Ask(fmt.Sprintf("предикация «%s» не прочитана: %v. Записать её?", text, err)) {in.Unknown = append(in.Unknown, text)} }
// resolve: препозиция-токен — символ (sym) или фрейм-интерпретатор (frame); алиасы раскрываются, неопределённый терм — ядра (sin, ln)
func resolve(v format.Value, env *Env) (sym string, frame *binding) {
for depth := 0; depth < 16; depth++ {
	if v.Kind == format.Op {return v.Text, nil}
	b, _ := env.find(v.Text)
	if b == nil && v.Kind == format.Term {return v.Text, nil}
	if b == nil || len(b.pred) != 1 {return "", nil}
	switch w := b.pred[0]; {
	case w.Kind == format.Array: return "", b
	case isPrepTok(w):
		v = w
	default: return "", nil}}; return "", nil }
// evalSlot: интерпретация слота; right — правый слот модели, блок в нём интерпретируется
// определение: Фрейм.Параметр : значение, ( ) — вычислить сейчас, параметры фрейма сохраняются
// recur — цикл, rescr — переписывание, Raw — операнд без вычисления, препозиция-фрейм — только в начале, аргументы — объектный блок
func (in *Interp) evalSlot(s *slot, env *Env, right bool) (format.Value, error) {
if s.def != nil {
	d := *s
	d.def = nil
	toks := d.toks()
	if format.IsPrepTerm(s.def.Text) && !s.alias && !(len(toks) == 1 && toks[0].Kind == format.Array) {in.unknown(s.toks(), fmt.Sprintf("— «%s» со строчной буквы — препозиция, она определяется алиасом или фреймом; объект пишется с прописной: «%s : …»", s.def.Text, upperFirst(s.def.Text))); return none, nil}
	if s.def.Kind == format.Field {return in.defParam(s, toks, env)}
	if !format.IsPrepTerm(s.def.Text) && !(len(toks) == 1 && toks[0].Kind == format.Array) {in.unknown(s.toks(), fmt.Sprintf("— «%s» с прописной буквы — фрейм; значение получает параметр фрейма: «%s.A : …»", s.def.Text, s.def.Text)); return none, nil}
	if len(toks) == 1 && toks[0].Kind == format.Expr {
		v, err := in.evalOperand(toks, env, false)
		if err != nil {return none, err}
		toks = []format.Value{v}}
	nb := &binding{pred: toks, env: env}
	if old, _ := env.find(s.def.Text); old != nil && !format.IsPrepTerm(s.def.Text) {nb.params = old.params}
	env.set(s.def.Text, nb)
	if in.OnFrame != nil && !format.IsPrepTerm(s.def.Text) && len(toks) == 1 && toks[0].Kind == format.Array {in.OnFrame(s.def.Text, toks[0].Items)}
	if s.alias && in.OnAlias != nil {in.OnAlias(s.def.Text, format.Canon(toks))}; return none, nil}
if s.op == nil && s.prefix == nil {return in.evalOperand(s.a, env, right)}
ptok := s.op
if s.prefix != nil {ptok = s.prefix}
sym, frame := resolve(*ptok, env)
if frame == nil && sym == "recur" {return in.recur(s, env)}
if frame == nil && sym == "rescr" {return in.rescr(s, env)}
if r := in.Raw[sym]; frame == nil && r != nil {
	_, fr := in.prepOf(sym)
	if s.prefix == nil || s.op != nil || len(s.a) != 1 {in.unknown(s.toks(), "— «"+sym+"» пишется в начале с одним операндом"); return none, nil}
	v, ok := r(s.a[0])
	if err := in.RawErr; !ok && err != nil {in.RawErr = nil; return none, err}
	if !ok {in.unknown(s.toks(), "— «"+sym+"» не интерпретирует такой операнд"); return none, nil}
	if fr != nil {in.use(fr)}; return v, nil}
if frame != nil {
	if s.prefix == nil || s.a[0].Kind != format.Object {in.unknown(s.toks(), fmt.Sprintf("— «%s» интерпретирует только «%s { … }»", ptok.Text, ptok.Text)); return none, nil}; return in.call(ptok.Text, frame, s.a[0], env)}
a, err := in.evalOperand(s.a, env, false)
if err != nil {return none, err}
pa, pb := &a, (*format.Value)(nil)
if s.prefix != nil {pa, pb = nil, &a} else {
	b, err := in.evalOperand(s.b, env, false)
	if err != nil {return none, err}
	pb = &b}
f, fr := in.prepOf(sym)
if f == nil {in.unknown(s.toks(), fmt.Sprintf("— нет интерпретатора препозиции «%s»", ptok.Text)); return none, nil}
v, ok := f(pa, pb)
if !ok {in.unknown(s.toks(), fmt.Sprintf("— «%s» (%s) не интерпретирует такие аргументы", ptok.Text, fr.Name)); return none, nil}
in.use(fr); return v, nil }
// evalOperand: значение операнда — ( ) вычисляется, [ ] справа — блок, 'терм, :терм, терм — аппликация, поле — параметр
func (in *Interp) evalOperand(a []format.Value, env *Env, right bool) (format.Value, error) {
v := a[0]
switch v.Kind {
case format.Expr:
	r, _, err := in.block(v.Items, env); return r, err
case format.Array:
	if right {r, _, err := in.block(v.Items, NewEnv(env)); return r, err}; return v, nil
case format.LitTerm: return format.Value{Kind: format.Term, Text: v.Text[1:]}, nil
case format.GetTerm:
	b, _ := env.find(v.Text[1:])
	if b == nil || len(b.pred) != 1 {in.unknown(a, "— имя не определено"); return none, nil}; return b.pred[0], nil
case format.Term: return in.apply(a, env)
case format.Field: return in.param(a, env)}; return v, nil }
// rawNames: имена препозиций Raw для чтения предикации
func (in *Interp) rawNames() map[string]bool {
m := map[string]bool{}
for k := range in.Raw {m[k] = true}; return m }
