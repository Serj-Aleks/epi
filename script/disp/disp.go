// Package disp — диспетчер предикаций (claude/ason-spec.md, разделы 9.7–9.9).
//
// Предикация читается из дерева ASON (пакет format). Форму предикации задаёт её
// препозиция; препозиция — имя интерпретатора. Препозиции — символы
// (`+`, `>`) и термы со строчной буквы; объекты — термы с прописной.
// «Ключ : значение» — определение. Нераспознанная предикация — не ошибка:
// она предлагается к записи в базу.
package disp

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"epi/script/format"
)

// Prep — интерпретатор препозиции. ok=false: для таких аргументов
// интерпретации нет (предикация уходит в базу нераспознанных).
type Prep func(a, b *format.Value) (format.Value, bool)

// Interp — среда интерпретации.
type Interp struct {
	Frames  []*Frame                // фреймы-интерпретаторы препозиций (9.11)
	Used    []string                // фреймы, интерпретировавшие последнюю Run
	OnModel func(f *Frame, m Model) // новая модель во фрейме-интерпретаторе
	Limit   int                     // лимит шагов (0.16)
	Ask     func(q string) bool
	Unknown []string // база нераспознанных предикаций (канонический текст)
	steps   int
	allowed int
}

// Stopped — пользователь не разрешил продолжать после исчерпания лимита.
type Stopped struct{ Steps int }

func (s Stopped) Error() string {
	return fmt.Sprintf("остановлено пользователем после %d шагов", s.Steps)
}

// Frame — фрейм-интерпретатор (9.11). Kernel — неизменяемая часть:
// препозиции, которые он интерпретирует. Динамическая часть растёт в
// контексте (алиасы, определения) и в базе моделей сессии.
type Frame struct {
	Name   string
	Kernel map[string]Prep
	Models []Model // динамическая часть: модели интерпретаций (9.11)
}

// Model — модель интерпретации «[ предикация ] -> интерпретация» в контексте
// фрейма-интерпретатора.
type Model struct {
	Pred  string
	Value format.Value
}

// Add добавляет модель, если такой предикации во фрейме ещё нет; true — добавлена.
func (f *Frame) Add(m Model) bool {
	for _, x := range f.Models {
		if x.Pred == m.Pred {
			return false
		}
	}
	f.Models = append(f.Models, m)
	return true
}

// Frame находит фрейм-интерпретатор по имени.
func (in *Interp) Frame(name string) *Frame {
	for _, f := range in.Frames {
		if f.Name == name {
			return f
		}
	}
	return nil
}

// pure: в предикации только значения, без имён — модель порождается только
// для таких предикаций (явные числа, 9.11); параметризованные — следующий шаг.
func pure(toks []format.Value) bool {
	for _, t := range toks {
		switch t.Kind {
		case format.Term, format.GetTerm, format.Field, format.GetField, format.Path:
			if !isPrepTok(t) {
				return false
			}
		case format.Array, format.Expr, format.Object:
			if !pure(t.Items) {
				return false
			}
		}
	}
	return true
}

// prep находит интерпретатор символа препозиции и его фрейм.
func (in *Interp) prep(sym string) (Prep, *Frame) {
	for _, f := range in.Frames {
		if p, ok := f.Kernel[sym]; ok {
			return p, f
		}
	}
	return nil, nil
}

func (in *Interp) use(f *Frame) {
	for _, n := range in.Used {
		if n == f.Name {
			return
		}
	}
	in.Used = append(in.Used, f.Name)
}

// New — среда с фреймами-интерпретаторами arifm и logic (имена — пять литер, 0.18).
func New(limit int, ask func(string) bool) *Interp {
	return &Interp{Frames: baseFrames(), Limit: limit, Ask: ask, allowed: limit}
}

// Env — контекст: модели «имя -> предикация». Имена сравниваются по format.Key.
type Env struct {
	vars   map[string]*binding
	parent *Env
}

type binding struct {
	pred []format.Value // правая часть модели «имя -> …»
	env  *Env           // контекст, в котором она записана
	// параметры фрейма: «Фрейм.Параметр : …» (9.12); параметров вне фрейма нет
	params map[string]*binding
}

func NewEnv(parent *Env) *Env { return &Env{vars: map[string]*binding{}, parent: parent} }

func (e *Env) find(name string) (*binding, *Env) {
	for c := e; c != nil; c = c.parent {
		if b, ok := c.vars[format.Key(name)]; ok {
			return b, c
		}
	}
	return nil, nil
}

// set: имя, уже связанное в иерархии контекстов, перевязывается там; новое —
// связывается в текущем контексте.
func (e *Env) set(name string, b *binding) {
	if _, c := e.find(name); c != nil {
		c.vars[format.Key(name)] = b
		return
	}
	e.vars[format.Key(name)] = b
}

var (
	none  = format.Value{Kind: format.Literal, Text: "@none"}
	vtrue = format.Value{Kind: format.Literal, Text: "@true"}
	vfals = format.Value{Kind: format.Literal, Text: "@false"}
)

// ---- разбор последовательности на предикации ----

type slot struct {
	def    *format.Value  // ключ определения «Ключ : …»
	alias  bool           // «имя : препозиция» — алиас препозиции (9.8)
	prefix *format.Value  // препозиция в начале
	a      []format.Value // операнд: значение или «Терм { аргументы }»
	op     *format.Value
	b      []format.Value
	ok     bool
}

type pred struct {
	left, right *slot
	toks        []format.Value
	ok          bool
}

// isPrepTok: токен — препозиция: символ (кроме `->` и `:`) или терм со
// строчной буквы (9.9).
func isPrepTok(v format.Value) bool {
	switch v.Kind {
	case format.Op:
		return v.Text != "->" && v.Text != ":" && v.Text != "."
	case format.Term:
		return format.IsPrepTerm(v.Text)
	}
	return false
}

func isModel(v format.Value) bool { return v.Kind == format.Op && v.Text == "->" }

type reader struct{ vs []format.Value }

func (r reader) isPrep(i int) bool { return i < len(r.vs) && isPrepTok(r.vs[i]) }

func (r reader) operand(i int) ([]format.Value, int, bool) {
	vs := r.vs
	if i >= len(vs) || vs[i].Kind == format.Op || r.isPrep(i) || format.IsDef(vs, i) {
		return nil, i, false
	}
	if vs[i].Kind == format.Term && i+1 < len(vs) && vs[i+1].Kind == format.Object {
		return vs[i : i+2], i + 2, true
	}
	return vs[i : i+1], i + 1, true
}

func (r reader) slot(i int) (*slot, int) {
	vs := r.vs
	if format.IsDef(vs, i) {
		key := vs[i]
		// значение, начинающееся с препозиции, — алиас препозиции (9.8)
		if r.isPrep(i + 2) {
			return &slot{def: &key, a: vs[i+2 : i+3], alias: true, ok: true}, i + 3
		}
		inner, j := r.slot(i + 2)
		inner.def = &key
		return inner, j
	}
	s := &slot{}
	if r.isPrep(i) {
		p := vs[i]
		s.prefix = &p
		i++
	}
	a, j, ok := r.operand(i)
	if !ok {
		return s, j
	}
	s.a, i = a, j
	if r.isPrep(i) {
		if s.prefix != nil {
			return s, i + 1 // вторая препозиция
		}
		o := vs[i]
		s.op = &o
		b, j, ok := r.operand(i + 1)
		if !ok {
			return s, j // препозиция в конце
		}
		s.b, i = b, j
		if r.isPrep(i) {
			return s, i + 1 // вторая препозиция: в одном слоте одна
		}
	}
	s.ok = true
	return s, i
}

// segments делит последовательность на предикации (9.10): граница — точка
// ` . ` (отдельный токен) или перевод строки (значение начинается на строке
// позже, чем кончилось предыдущее). Пустые части пропускаются.
func segments(vs []format.Value) [][]format.Value {
	var segs [][]format.Value
	start := 0
	cut := func(end int) {
		if end > start {
			segs = append(segs, vs[start:end])
		}
	}
	for i, v := range vs {
		switch {
		case v.Kind == format.Op && v.Text == ".":
			cut(i)
			start = i + 1
		case i > start && v.Line > vs[i-1].End && vs[i-1].End > 0:
			cut(i)
			start = i
		}
	}
	cut(len(vs))
	return segs
}

// Split делит текст на предикации верхнего уровня (9.10) в каноническом
// тексте: на каждую — свой ответ (9.11.8).
func Split(text string) ([]string, error) {
	vs, err := format.Parse(text)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, seg := range segments(vs) {
		out = append(out, format.Format(seg))
	}
	return out, nil
}

// readPred читает предикацию из части seg; часть должна быть ровно одной
// предикацией, иначе она не распознана целиком.
func readPred(seg []format.Value) pred {
	r := reader{vs: seg}
	p := pred{toks: seg}
	var i int
	p.left, i = r.slot(0)
	p.ok = p.left.ok
	if p.ok && i < len(seg) && isModel(seg[i]) {
		p.right, i = r.slot(i + 1)
		p.ok = p.right.ok
	}
	if i != len(seg) {
		p.ok = false
	}
	return p
}

// ---- интерпретация ----

// Run читает текст и интерпретирует его в контексте env.
func (in *Interp) Run(text string, env *Env) (format.Value, error) {
	in.Used = nil
	in.steps, in.allowed = 0, in.Limit // лимит — на одну интерпретацию
	vs, err := format.Parse(text)
	if err != nil {
		return none, err
	}
	v, _, err := in.block(vs, env)
	return v, err
}

// block: предикации по порядку. Модель с истинным левым слотом завершает блок
// интерпретацией правого (9.1); с ложным — дальше. Иначе результат — последняя.
func (in *Interp) block(vs []format.Value, env *Env) (format.Value, bool, error) {
	res := none

	for _, seg := range segments(vs) {
		p := readPred(seg)
		if err := in.step(); err != nil {
			return none, false, err
		}
		if !p.ok {
			in.unknown(p.toks, "не распознана"+caseHint(p.toks, env))
			res = none
			continue
		}
		if p.right == nil {
			v, err := in.slot(p.left, env, false)
			if err != nil {
				return none, false, err
			}
			res = v
			continue
		}
		f, err := in.slot(p.left, env, false)
		if err != nil {
			return none, false, err
		}
		switch f.Text {
		case "@true":
			v, err := in.slot(p.right, env, true)
			return v, true, err
		case "@false":
			res = none
		default:
			return none, false, fmt.Errorf("строка %d: левый слот модели дал не флаг: %s", p.toks[0].Line, f)
		}
	}
	return res, false, nil
}

func (in *Interp) step() error {
	in.steps++
	if in.Limit > 0 && in.steps > in.allowed {
		if in.Ask != nil && in.Ask(fmt.Sprintf("исчерпан лимит %d шагов. Продолжать?", in.Limit)) {
			in.allowed += in.Limit
			return nil
		}
		return Stopped{in.steps - 1}
	}
	return nil
}

func (in *Interp) unknown(toks []format.Value, why string) {
	text := format.Format(toks)
	if in.Ask != nil && in.Ask(fmt.Sprintf("предикация «%s» %s. Записать её?", text, why)) {
		in.Unknown = append(in.Unknown, text)
	}
}

// resolve: чем интерпретируется препозиция-токен: символом (сим != "")
// или фреймом-интерпретатором (frame != nil). Алиасы раскрываются.
func resolve(v format.Value, env *Env) (sym string, frame *binding) {
	for depth := 0; depth < 16; depth++ {
		if v.Kind == format.Op {
			return v.Text, nil
		}
		b, _ := env.find(v.Text)
		if b == nil || len(b.pred) != 1 {
			return "", nil
		}
		switch w := b.pred[0]; {
		case w.Kind == format.Array:
			return "", b
		case isPrepTok(w):
			v = w
		default:
			return "", nil
		}
	}
	return "", nil
}

// slot: right=true — правый слот модели: блок в нём интерпретируется.
func (in *Interp) slot(s *slot, env *Env, right bool) (format.Value, error) {
	if s.def != nil {
		d := *s
		d.def = nil
		toks := d.toks()
		if format.IsPrepTerm(s.def.Text) && !s.alias && !(len(toks) == 1 && toks[0].Kind == format.Array) {
			in.unknown(s.toks(), fmt.Sprintf("— «%s» со строчной буквы — препозиция, она определяется алиасом или фреймом; объект пишется с прописной: «%s : …»", s.def.Text, upperFirst(s.def.Text)))
			return none, nil
		}
		if s.def.Kind == format.Field { // «Фрейм.Параметр : значение» (9.12)
			return in.defParam(s, toks, env)
		}
		if !format.IsPrepTerm(s.def.Text) && !(len(toks) == 1 && toks[0].Kind == format.Array) {
			in.unknown(s.toks(), fmt.Sprintf("— «%s» с прописной буквы — фрейм; значение получает параметр фрейма: «%s.A : …»", s.def.Text, s.def.Text))
			return none, nil
		}
		if len(toks) == 1 && toks[0].Kind == format.Expr { // ( ) — вычислить сейчас (9.3)
			v, err := in.operand(toks, env, false)
			if err != nil {
				return none, err
			}
			toks = []format.Value{v}
		}
		env.set(s.def.Text, &binding{pred: toks, env: env})
		return none, nil
	}
	if s.op == nil && s.prefix == nil {
		return in.operand(s.a, env, right)
	}
	ptok := s.op
	if s.prefix != nil {
		ptok = s.prefix
	}
	sym, frame := resolve(*ptok, env)
	if frame != nil { // препозиция-фрейм: только в начале, аргументы — объектный блок
		if s.prefix == nil || s.a[0].Kind != format.Object {
			in.unknown(s.toks(), fmt.Sprintf("— «%s» интерпретирует только «%s { … }»", ptok.Text, ptok.Text))
			return none, nil
		}
		return in.call(frame, s.a[0], env)
	}
	a, err := in.operand(s.a, env, false)
	if err != nil {
		return none, err
	}
	pa, pb := &a, (*format.Value)(nil)
	if s.prefix != nil {
		pa, pb = nil, &a
	} else {
		b, err := in.operand(s.b, env, false)
		if err != nil {
			return none, err
		}
		pb = &b
	}
	f, fr := in.prep(sym)
	if f == nil {
		in.unknown(s.toks(), fmt.Sprintf("— нет интерпретатора препозиции «%s»", ptok.Text))
		return none, nil
	}
	v, ok := f(pa, pb)
	if !ok {
		in.unknown(s.toks(), fmt.Sprintf("— «%s» (%s) не интерпретирует такие аргументы", ptok.Text, fr.Name))
		return none, nil
	}
	in.use(fr)
	if toks := s.toks(); pure(toks) {
		m := Model{Pred: format.Format(toks), Value: v}
		if fr.Add(m) && in.OnModel != nil {
			in.OnModel(fr, m)
		}
	}
	return v, nil
}

func (s *slot) toks() []format.Value {
	var t []format.Value
	if s.def != nil {
		t = append(t, *s.def, format.Value{Kind: format.Op, Text: ":"})
	}
	if s.prefix != nil {
		t = append(t, *s.prefix)
	}
	t = append(t, s.a...)
	if s.op != nil {
		t = append(append(t, *s.op), s.b...)
	}
	return t
}

func (in *Interp) operand(a []format.Value, env *Env, right bool) (format.Value, error) {
	v := a[0]
	switch v.Kind {
	case format.Expr:
		r, _, err := in.block(v.Items, env)
		return r, err
	case format.Array:
		if right {
			r, _, err := in.block(v.Items, NewEnv(env))
			return r, err
		}
		return v, nil
	case format.LitTerm:
		return format.Value{Kind: format.Term, Text: v.Text[1:]}, nil
	case format.GetTerm:
		b, _ := env.find(v.Text[1:])
		if b == nil || len(b.pred) != 1 {
			in.unknown(a, "— имя не определено")
			return none, nil
		}
		return b.pred[0], nil
	case format.Term:
		return in.apply(a, env)
	case format.Field:
		return in.param(a, env)
	}
	return v, nil
}

// apply: аппликация объекта по имени, с аргументами или без.
func (in *Interp) apply(a []format.Value, env *Env) (format.Value, error) {
	b, _ := env.find(a[0].Text)
	if b == nil {
		in.unknown(a, "— имя не определено")
		return none, nil
	}
	if len(b.pred) == 1 && b.pred[0].Kind == format.Array { // фрейм
		if len(a) == 2 {
			return in.call(b, a[1], env)
		}
		return in.call(b, format.Value{Kind: format.Object}, env)
	}
	if len(b.pred) == 0 { // фрейм только с параметрами
		return frameValue(b), nil
	}
	if len(a) == 2 {
		in.unknown(a, "— «"+a[0].Text+"» не фрейм")
		return none, nil
	}
	// модель «имя -> предикация»: живая формула, интерпретируется при обращении
	r, _, err := in.block(b.pred, b.env)
	return r, err
}

// call: экземпляр фрейма — контекст, дочерний к контексту описания фрейма;
// аргументы «Ключ : предикация» — модели экземпляра, интерпретируются в
// контексте вызова при обращении.
func (in *Interp) call(frame *binding, args format.Value, env *Env) (format.Value, error) {
	inst := NewEnv(frame.env)
	for _, e := range args.Entries() {
		inst.vars[format.Key(e.Key.Text)] = &binding{pred: e.Value, env: env}
	}
	if err := in.step(); err != nil {
		return none, err
	}
	r, _, err := in.block(frame.pred[0].Items, inst)
	return r, err
}

// ---- интерпретаторы препозиций ----

func num(v *format.Value) (float64, bool, bool) { // значение, целое ли, число ли
	if v == nil {
		return 0, false, false
	}
	switch v.Kind {
	case format.Integer:
		n, err := strconv.ParseInt(v.Text, 10, 64)
		return float64(n), true, err == nil
	case format.Decimal:
		f, err := strconv.ParseFloat(v.Text, 64)
		return f, false, err == nil
	}
	return 0, false, false
}

func number(f float64, integer bool) format.Value {
	if integer && f == math.Trunc(f) && math.Abs(f) < 1<<53 {
		return format.Value{Kind: format.Integer, Text: strconv.FormatInt(int64(f), 10)}
	}
	s := strconv.FormatFloat(f, 'f', -1, 64)
	if !strings.Contains(s, ".") {
		s += ".0"
	}
	return format.Value{Kind: format.Decimal, Text: s}
}

func flag(b bool) format.Value {
	if b {
		return vtrue
	}
	return vfals
}

func baseFrames() []*Frame {
	arith := func(op func(x, y float64) (float64, bool)) Prep {
		return func(a, b *format.Value) (format.Value, bool) {
			x, xi, ok1 := num(a)
			y, yi, ok2 := num(b)
			if a == nil { // в начале предикации: только минус
				x, xi, ok1 = 0, true, true
			}
			if !ok1 || !ok2 {
				return none, false
			}
			r, ok := op(x, y)
			if !ok {
				return none, false
			}
			return number(r, xi && yi), true
		}
	}
	cmp := func(op func(x, y float64) bool) Prep {
		return func(a, b *format.Value) (format.Value, bool) {
			x, _, ok1 := num(a)
			y, _, ok2 := num(b)
			if !ok1 || !ok2 {
				return none, false
			}
			return flag(op(x, y)), true
		}
	}
	eq := func(neg bool) Prep {
		return func(a, b *format.Value) (format.Value, bool) {
			if a == nil {
				return none, false
			}
			x, _, ok1 := num(a)
			y, _, ok2 := num(b)
			e := ok1 && ok2 && x == y || !(ok1 && ok2) && a.Kind == b.Kind && strings.EqualFold(a.String(), b.String())
			return flag(e != neg), true
		}
	}
	plusOnlyInfix := func(p Prep) Prep {
		return func(a, b *format.Value) (format.Value, bool) {
			if a == nil {
				return none, false
			}
			return p(a, b)
		}
	}
	arifm := map[string]Prep{
		"+": plusOnlyInfix(arith(func(x, y float64) (float64, bool) { return x + y, true })),
		"-": arith(func(x, y float64) (float64, bool) { return x - y, true }),
		"*": plusOnlyInfix(arith(func(x, y float64) (float64, bool) { return x * y, true })),
		"/": plusOnlyInfix(func(a, b *format.Value) (format.Value, bool) {
			x, _, ok1 := num(a)
			y, _, ok2 := num(b)
			if !ok1 || !ok2 || y == 0 {
				return none, false
			}
			return number(x/y, false), true
		}),
	}
	logic := map[string]Prep{
		"<":  plusOnlyInfix(cmp(func(x, y float64) bool { return x < y })),
		">":  plusOnlyInfix(cmp(func(x, y float64) bool { return x > y })),
		"<=": plusOnlyInfix(cmp(func(x, y float64) bool { return x <= y })),
		">=": plusOnlyInfix(cmp(func(x, y float64) bool { return x >= y })),
		"=":  eq(false),
		"<>": eq(true),
	}
	return []*Frame{{Name: "arifm", Kernel: arifm}, {Name: "logic", Kernel: logic}}
}

func upperFirst(s string) string {
	r := []rune(s)
	if len(r) > 0 {
		r[0] = unicode.ToUpper(r[0])
	}
	return string(r)
}

// caseHint: подсказка, если в предикации неопределённый терм со строчной
// буквы — он читается как препозиция (9.9), а имелся в виду, вероятно, объект.
func caseHint(toks []format.Value, env *Env) string {
	for _, t := range toks {
		if t.Kind == format.Term && format.IsPrepTerm(t.Text) {
			if b, _ := env.find(t.Text); b == nil {
				return fmt.Sprintf(" («%s» со строчной буквы — препозиция; объект пишется с прописной: «%s»)", t.Text, upperFirst(t.Text))
			}
		}
	}
	return ""
}

// splitField: «Фрейм.Параметр» → фрейм и параметр; оба — с прописной (9.9, 9.12).
func splitField(t string) (frame, param string, ok bool) {
	parts := strings.Split(t, ".")
	if len(parts) != 2 || format.IsPrepTerm(parts[0]) || format.IsPrepTerm(parts[1]) {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// defParam: «Фрейм.Параметр : значение». Значение связывается без вычисления
// (9.3), `( )` — вычислить сейчас. Фрейма нет — он появляется с этим параметром.
func (in *Interp) defParam(s *slot, toks []format.Value, env *Env) (format.Value, error) {
	frame, param, ok := splitField(s.def.Text)
	if !ok {
		in.unknown(s.toks(), "— параметр пишется «Фрейм.Параметр», оба с прописной буквы")
		return none, nil
	}
	if len(toks) == 1 && toks[0].Kind == format.Expr {
		v, err := in.operand(toks, env, false)
		if err != nil {
			return none, err
		}
		toks = []format.Value{v}
	}
	fb, _ := env.find(frame)
	if fb == nil {
		fb = &binding{env: env}
		env.set(frame, fb)
	}
	if fb.params == nil {
		fb.params = map[string]*binding{}
	}
	fb.params[format.Key(param)] = &binding{pred: toks, env: env}
	return none, nil
}

// param: значение «Фрейм.Параметр» — модель параметра интерпретируется при обращении.
func (in *Interp) param(a []format.Value, env *Env) (format.Value, error) {
	frame, param, ok := splitField(a[0].Text)
	if !ok {
		in.unknown(a, "— параметр пишется «Фрейм.Параметр», оба с прописной буквы")
		return none, nil
	}
	fb, _ := env.find(frame)
	if fb == nil {
		in.unknown(a, fmt.Sprintf("— фрейм «%s» не определён", frame))
		return none, nil
	}
	pb := fb.params[format.Key(param)]
	if pb == nil {
		in.unknown(a, fmt.Sprintf("— у фрейма «%s» нет параметра «%s»", frame, param))
		return none, nil
	}
	r, _, err := in.block(pb.pred, pb.env)
	return r, err
}

// frameValue: фрейм как значение — объектный блок его параметров.
func frameValue(b *binding) format.Value {
	keys := make([]string, 0, len(b.params))
	for k := range b.params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	v := format.Value{Kind: format.Object, Items: []format.Value{}}
	for _, k := range keys {
		v.Items = append(v.Items, format.Value{Kind: format.Term, Text: k}, format.Value{Kind: format.Op, Text: ":"})
		v.Items = append(v.Items, b.params[k].pred...)
	}
	return v
}
