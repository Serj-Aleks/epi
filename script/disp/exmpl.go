// epi/script/disp/exmpl.go
// exmpl: интерпретатор шаблонов в тексте — шаблон строкой в записи Go (RE2), удобнее длинной строкой ''…'' (без экранирования): analg, total, vecsl
package disp
import ("fmt"; "regexp"; "strings"; "epi/script/format")
// pattern: шаблон из строки; ошибка записи — не распознано
func pattern(v *format.Value) (*regexp.Regexp, bool) {
s, ok := text(v)
if !ok {return nil, false}
re, err := regexp.Compile(s)
if err != nil {return nil, false}; return re, true }
// exmplFrame: фрейм exmpl; S analg P — вся строка подходит под шаблон (флаг); S total P — все вхождения, каждое — список [ вхождение группа … ], нет — []; vecsl — в vecsl
func exmplFrame() *Frame {
k := map[string]Prep{
	"analg": func(a, b *format.Value) (format.Value, bool) {
		s, ok := text(a)
		re, ok2 := pattern(b)
		if !ok || !ok2 {return none, false}
		whole, err := regexp.Compile(`^(?:` + re.String() + `)$`)
		if err != nil {return none, false}; return flag(whole.MatchString(s)), true},
	"total": func(a, b *format.Value) (format.Value, bool) {
		s, ok := text(a)
		re, ok2 := pattern(b)
		if !ok || !ok2 {return none, false}
		out := format.Value{Kind: format.Array, Items: []format.Value{}}
		for _, m := range re.FindAllStringSubmatch(s, -1) {out.Items = append(out.Items, strs(m))}; return out, true},
	"vecsl": func(a, b *format.Value) (format.Value, bool) {return none, false},}
return &Frame{Name: "exmpl", Kernel: k} }
// strs: список строк
func strs(ss []string) format.Value {
out := format.Value{Kind: format.Array, Items: []format.Value{}}
for _, s := range ss {out.Items = append(out.Items, format.Value{Kind: format.String, Text: s})}; return out }
// vecsl: «S vecsl [ P R ]» — все вхождения P заменить на R ($1 — группа); «S vecsl [ P Фрейм ]» — каждое вхождение заменить тем, что вернёт фрейм (роли In по порядку — вхождение и группы)
func (in *Interp) vecsl(s *slot, env *Env) (format.Value, error) {
fail := func(why string) (format.Value, error) {in.unknown(s.toks(), "— vecsl: "+why); return none, nil}
if s.prefix != nil || s.op == nil {return fail("пишется «Строка vecsl [ Шаблон Замена ]»")}
a, err := in.evalOperand(s.a, env, false)
if err != nil {return none, err}
var b format.Value
if len(s.b) == 1 && s.b[0].Kind == format.Array {b = s.b[0]} else {if b, err = in.evalOperand(s.b, env, true); err != nil {return none, err}}
str, ok := text(&a)
if !ok {return fail("слева нужна строка")}
if b.Kind != format.Array || len(b.Items) != 2 {return fail("справа нужна пара [ Шаблон Замена ]")}
re, ok := pattern(&b.Items[0])
if !ok {return fail("шаблон не читается — запись Go (RE2), удобнее ''…''")}
use := func() {if _, fr := in.prepOf("vecsl"); fr != nil {in.use(fr)}}
if r, ok := text(&b.Items[1]); ok {use(); return format.Value{Kind: format.String, Text: re.ReplaceAllString(str, r)}, nil}
if b.Items[1].Kind != format.Term {return fail("замена — строка или имя фрейма")}
name := b.Items[1].Text
fb, _ := env.findFrame(name)
if fb == nil || len(roleNames(fb, "In")) == 0 {return fail("«" + name + "» — нужен фрейм с ролями In: вхождение и группы")}
n := len(roleNames(fb, "In"))
var out strings.Builder
last := 0
for _, ix := range re.FindAllStringSubmatchIndex(str, -1) {
	var g []string
	for i := 0; i < len(ix); i += 2 {if ix[i] < 0 {g = append(g, "")} else {g = append(g, str[ix[i]:ix[i+1]])}}
	if len(g) > n {g = g[:n]}
	v, err := in.call(name, fb, strs(g), env)
	if err != nil {return none, err}
	t, ok := text(&v)
	if !ok {return none, fmt.Errorf("vecsl: «%s» вернул не строку: %s — замена должна быть строкой", name, v.String())}
	out.WriteString(str[last:ix[0]] + t)
	last = ix[1]}
out.WriteString(str[last:])
in.use(&Frame{Name: name})
use(); return format.Value{Kind: format.String, Text: out.String()}, nil }
