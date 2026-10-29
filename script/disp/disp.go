// Package disp — диспетчер предикаций (claude/ason-spec.md, раздел 9.7).
//
// Предикация читается из дерева ASON (пакет ason). Форму предикации задаёт её
// препозиция; препозиция — имя интерпретатора. Предикация без препозиции —
// аппликация фрейма по имени; «имя:» — описание фрейма (определение).
// Нераспознанная предикация — не ошибка: она предлагается к записи в базу.
package disp

import (
	_ "embed"
	"fmt"
	"math"
	"strconv"
	"strings"

	"epi/script/ason"
)

// Prep — интерпретатор препозиции. ok=false: для таких аргументов
// интерпретации нет (предикация уходит в базу нераспознанных).
type Prep func(a, b *ason.Value) (ason.Value, bool)

// Interp — среда интерпретации.
type Interp struct {
	Preps   map[string]Prep // препозиция → интерпретатор
	Limit   int             // лимит шагов (0.16)
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

// New — среда с интерпретаторами арифметики и сравнения.
func New(limit int, ask func(string) bool) *Interp {
	return &Interp{Preps: basePreps(), Limit: limit, Ask: ask, allowed: limit}
}

// Env — контекст: модели «имя → предикация». Имена регистронезависимы.
type Env struct {
	vars   map[string]*binding
	parent *Env
}

type binding struct {
	pred []ason.Value // правая часть модели «имя -> …»
	env  *Env         // контекст, в котором она записана
}

func NewEnv(parent *Env) *Env { return &Env{vars: map[string]*binding{}, parent: parent} }

func (e *Env) find(name string) (*binding, *Env) {
	for c := e; c != nil; c = c.parent {
		if b, ok := c.vars[strings.ToLower(name)]; ok {
			return b, c
		}
	}
	return nil, nil
}

// set: имя, уже связанное в иерархии контекстов, перевязывается там; новое —
// связывается в текущем контексте.
func (e *Env) set(name string, b *binding) {
	if _, c := e.find(name); c != nil {
		c.vars[strings.ToLower(name)] = b
		return
	}
	e.vars[strings.ToLower(name)] = b
}

var (
	none  = ason.Value{Kind: ason.Literal, Text: "@none"}
	vtrue = ason.Value{Kind: ason.Literal, Text: "@true"}
	vfals = ason.Value{Kind: ason.Literal, Text: "@false"}
)

// ---- разбор последовательности на предикации (жёсткий синтаксис 9.7) ----

type slot struct {
	def    *ason.Value  // «имя:» — описание (определение)
	prefix *ason.Value  // препозиция в начале
	a      []ason.Value // операнд: значение или «терм { аргументы }»
	op     *ason.Value
	b      []ason.Value
	alias  bool // «имя: препозиция» — алиас препозиции (9.8)
	ok     bool
}

type pred struct {
	left, right *slot
	toks        []ason.Value
	ok          bool
}

// prepName: имя препозиции, если токен — препозиция: символ препозиции или
// терм-алиас, связанный с препозицией (`plus: +`, решение 9.8).
func prepName(v ason.Value, env *Env) (string, bool) {
	for depth := 0; depth < 16; depth++ {
		if v.Kind == ason.Op {
			return v.Text, v.Text != "->"
		}
		if v.Kind != ason.Term || env == nil {
			return "", false
		}
		b, _ := env.find(v.Text)
		if b == nil || len(b.pred) != 1 {
			return "", false
		}
		v = b.pred[0]
	}
	return "", false
}

func isModel(v ason.Value) bool { return v.Kind == ason.Op && v.Text == "->" }

type reader struct {
	vs  []ason.Value
	env *Env
}

func (r reader) isPrep(i int) bool {
	if i >= len(r.vs) {
		return false
	}
	_, ok := prepName(r.vs[i], r.env)
	return ok
}

func (r reader) operand(i int) ([]ason.Value, int, bool) {
	vs := r.vs
	if i >= len(vs) || vs[i].Kind == ason.Op || vs[i].Kind == ason.SetTerm || r.isPrep(i) {
		return nil, i, false
	}
	if vs[i].Kind == ason.Term && i+1 < len(vs) && vs[i+1].Kind == ason.Object {
		return vs[i : i+2], i + 2, true
	}
	return vs[i : i+1], i + 1, true
}

func (r reader) slot(i int) (*slot, int) {
	vs := r.vs
	s := &slot{}
	if i < len(vs) && vs[i].Kind == ason.SetTerm {
		d := vs[i]
		// значение, начинающееся с препозиции, — алиас препозиции (9.8)
		if r.isPrep(i + 1) {
			return &slot{def: &d, a: vs[i+1 : i+2], alias: true, ok: true}, i + 2
		}
		inner, j := r.slot(i + 1)
		inner.def = &d
		return inner, j
	}
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
			return s, i + 1 // вторая препозиция: в одной предикации одна
		}
	}
	s.ok = true
	return s, i
}

// pred читает одну предикацию с позиции i в текущем контексте (алиасы,
// определённые предыдущими предикациями, уже действуют). Нераспознанная часть
// забирает токены до следующей границы.
func (r reader) pred(i int) (pred, int) {
	vs := r.vs
	start := i
	p := pred{}
	p.left, i = r.slot(i)
	p.ok = p.left.ok
	if p.ok && i < len(vs) && isModel(vs[i]) {
		p.right, i = r.slot(i + 1)
		p.ok = p.right.ok
	}
	if !p.ok {
		for i < len(vs) && (vs[i].Kind == ason.Op || r.isPrep(i) || (i > start && r.isPrep(i-1))) {
			i++
		}
		if i == start {
			i++
		}
	}
	p.toks = vs[start:i]
	return p, i
}

// ---- интерпретация ----

// Run читает текст и интерпретирует его в контексте env.
func (in *Interp) Run(text string, env *Env) (ason.Value, error) {
	vs, err := ason.Parse(text)
	if err != nil {
		return none, err
	}
	v, _, err := in.block(vs, env)
	return v, err
}

// block: предикации по порядку. Модель с истинным левым слотом завершает блок
// интерпретацией правого (9.1); с ложным — дальше. Иначе результат — последняя.
func (in *Interp) block(vs []ason.Value, env *Env) (ason.Value, bool, error) {
	res := none
	r := reader{vs: vs, env: env}
	for i := 0; i < len(vs); {
		var p pred
		p, i = r.pred(i)
		if err := in.step(); err != nil {
			return none, false, err
		}
		if !p.ok {
			in.unknown(p.toks, "не распознана")
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

func (in *Interp) unknown(toks []ason.Value, why string) {
	text := ason.Format(toks)
	if in.Ask != nil && in.Ask(fmt.Sprintf("предикация «%s» %s. Записать её?", text, why)) {
		in.Unknown = append(in.Unknown, text)
	}
}

// slot: right=true — правый слот модели: блок в нём интерпретируется.
func (in *Interp) slot(s *slot, env *Env, right bool) (ason.Value, error) {
	if s.def != nil {
		d := *s
		d.def = nil
		toks := d.toks()
		if len(toks) == 1 && toks[0].Kind == ason.Expr { // ( ) — вычислить сейчас (9.3)
			v, err := in.operand(toks, env, false)
			if err != nil {
				return none, err
			}
			toks = []ason.Value{v}
		}
		env.set(strings.TrimSuffix(s.def.Text, ":"), &binding{pred: toks, env: env})
		return none, nil
	}
	if s.op == nil && s.prefix == nil {
		return in.operand(s.a, env, right)
	}
	a, err := in.operand(s.a, env, false)
	if err != nil {
		return none, err
	}
	name, pa, pb := "", &a, (*ason.Value)(nil)
	if s.prefix != nil {
		name, _ = prepName(*s.prefix, env)
		pa, pb = nil, &a
	} else {
		b, err := in.operand(s.b, env, false)
		if err != nil {
			return none, err
		}
		name, _ = prepName(*s.op, env)
		pb = &b
	}
	f := in.Preps[name]
	if f == nil {
		in.unknown(s.toks(), fmt.Sprintf("— нет интерпретатора препозиции «%s»", name))
		return none, nil
	}
	v, ok := f(pa, pb)
	if !ok {
		in.unknown(s.toks(), fmt.Sprintf("— «%s» не интерпретирует такие аргументы", name))
		return none, nil
	}
	return v, nil
}

func (s *slot) toks() []ason.Value {
	var t []ason.Value
	if s.def != nil {
		t = append(t, *s.def)
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

func (in *Interp) operand(a []ason.Value, env *Env, right bool) (ason.Value, error) {
	v := a[0]
	switch v.Kind {
	case ason.Expr:
		r, _, err := in.block(v.Items, env)
		return r, err
	case ason.Array:
		if right {
			r, _, err := in.block(v.Items, NewEnv(env))
			return r, err
		}
		return v, nil
	case ason.LitTerm:
		return ason.Value{Kind: ason.Term, Text: v.Text[1:]}, nil
	case ason.GetTerm:
		b, _ := env.find(v.Text[1:])
		if b == nil || len(b.pred) != 1 {
			in.unknown(a, "— имя не определено")
			return none, nil
		}
		return b.pred[0], nil
	case ason.Term:
		return in.apply(a, env)
	}
	return v, nil
}

// apply: аппликация фрейма по имени, с аргументами-моделями или без.
func (in *Interp) apply(a []ason.Value, env *Env) (ason.Value, error) {
	b, _ := env.find(a[0].Text)
	if b == nil {
		in.unknown(a, "— имя не определено")
		return none, nil
	}
	if len(b.pred) == 1 && b.pred[0].Kind == ason.Array { // фрейм
		inst := NewEnv(b.env)
		if len(a) == 2 {
			for _, e := range a[1].Entries() {
				inst.vars[strings.ToLower(strings.TrimSuffix(e.Key.Text, ":"))] = &binding{pred: e.Value, env: env}
			}
		}
		if err := in.step(); err != nil {
			return none, err
		}
		r, _, err := in.block(b.pred[0].Items, inst)
		return r, err
	}
	if len(a) == 2 {
		in.unknown(a, "— «"+a[0].Text+"» не фрейм")
		return none, nil
	}
	// модель «имя -> предикация»: живая формула, интерпретируется при обращении
	r, _, err := in.block(b.pred, b.env)
	return r, err
}

// ---- интерпретаторы препозиций ----

func num(v *ason.Value) (float64, bool, bool) { // значение, целое ли, число ли
	if v == nil {
		return 0, false, false
	}
	switch v.Kind {
	case ason.Integer:
		n, err := strconv.ParseInt(v.Text, 10, 64)
		return float64(n), true, err == nil
	case ason.Decimal:
		f, err := strconv.ParseFloat(v.Text, 64)
		return f, false, err == nil
	}
	return 0, false, false
}

func number(f float64, integer bool) ason.Value {
	if integer && f == math.Trunc(f) && math.Abs(f) < 1<<53 {
		return ason.Value{Kind: ason.Integer, Text: strconv.FormatInt(int64(f), 10)}
	}
	s := strconv.FormatFloat(f, 'f', -1, 64)
	if !strings.Contains(s, ".") {
		s += ".0"
	}
	return ason.Value{Kind: ason.Decimal, Text: s}
}

func flag(b bool) ason.Value {
	if b {
		return vtrue
	}
	return vfals
}

func basePreps() map[string]Prep {
	arith := func(op func(x, y float64) (float64, bool)) Prep {
		return func(a, b *ason.Value) (ason.Value, bool) {
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
		return func(a, b *ason.Value) (ason.Value, bool) {
			x, _, ok1 := num(a)
			y, _, ok2 := num(b)
			if !ok1 || !ok2 {
				return none, false
			}
			return flag(op(x, y)), true
		}
	}
	eq := func(neg bool) Prep {
		return func(a, b *ason.Value) (ason.Value, bool) {
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
		return func(a, b *ason.Value) (ason.Value, bool) {
			if a == nil {
				return none, false
			}
			return p(a, b)
		}
	}
	return map[string]Prep{
		"+": plusOnlyInfix(arith(func(x, y float64) (float64, bool) { return x + y, true })),
		"-": arith(func(x, y float64) (float64, bool) { return x - y, true }),
		"*": plusOnlyInfix(arith(func(x, y float64) (float64, bool) { return x * y, true })),
		"/": plusOnlyInfix(func(a, b *ason.Value) (ason.Value, bool) {
			x, _, ok1 := num(a)
			y, _, ok2 := num(b)
			if !ok1 || !ok2 || y == 0 {
				return none, false
			}
			return number(x/y, false), true
		}),
		"<":  plusOnlyInfix(cmp(func(x, y float64) bool { return x < y })),
		">":  plusOnlyInfix(cmp(func(x, y float64) bool { return x > y })),
		"<=": plusOnlyInfix(cmp(func(x, y float64) bool { return x <= y })),
		">=": plusOnlyInfix(cmp(func(x, y float64) bool { return x >= y })),
		"=":  eq(false),
		"<>": eq(true),
	}
}

//go:embed lexicon.ason
var lexicon string

// Global — контекст верхнего уровня с системным лексиконом (lexicon.ason).
func (in *Interp) Global() (*Env, error) {
	env := NewEnv(nil)
	if _, err := in.Run(lexicon, env); err != nil {
		return nil, fmt.Errorf("лексикон: %w", err)
	}
	return env, nil
}
