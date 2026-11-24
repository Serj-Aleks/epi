// epi/script/disp/frams.go
// frams: фреймы — аппликация и вызов (роли In, результат Ex), параметры «Фрейм.Параметр», синонимы
package disp
import ("fmt"; "sort"; "strings"; "unicode"; "epi/script/format")
// apply: аппликация объекта по имени, с аргументами или без; модель «имя -> предикация» интерпретируется при обращении; фрейм с телом отмечается в Used, как интерпретатор
func (in *Interp) apply(a []format.Value, env *Env) (format.Value, error) {
b, _ := env.findFrame(a[0].Text)
if b == nil {in.unknown(a, "— имя не определено"); return none, nil}
if len(b.pred) == 1 && b.pred[0].Kind == format.Array {
	in.use(&Frame{Name: a[0].Text})
	if len(a) == 2 {return in.call(a[0].Text, b, a[1], env)}; return in.call(a[0].Text, b, format.Value{Kind: format.Object}, env)}
if len(b.pred) == 0 {return frameValue(b), nil}
if len(a) == 2 {in.unknown(a, "— «"+a[0].Text+"» не фрейм"); return none, nil}
r, _, err := in.block(b.pred, b.env); return r, err }
// call: экземпляр фрейма — контекст, дочерний к описанию; аргументы «Ключ : предикация» интерпретируются в контексте вызова, ( ) — сейчас (9.3, рекурсия);
// аргумент — и имя в теле, и параметр фрейма на время вызова; у фрейма с In или Ex все его параметры живут на время вызова и в файл не пишутся; In — роли входа:
// позиционный вызов «Фрейм [ 2 10 ]» по их порядку, неподанная роль — параметр фрейма по умолчанию; ошибка с подсказкой:
// в In не имя роли, значений больше, чем ролей, у роли нет ни значения, ни умолчания;
// результат — Ex: термин — значение этого параметра (функция), блок — блок параметров выхода (процедура); без Ex — последняя импликация
func (in *Interp) call(name string, frame *binding, args format.Value, env *Env) (format.Value, error) {
inst := NewEnv(frame.env)
if frame.params == nil {frame.params = map[string]*binding{}}
roles := roleNames(frame, "In")
es := args.Entries()
if args.Kind == format.Array {for i, v := range args.Items {if i < len(roles) {es = append(es, format.Entry{Key: format.Value{Kind: format.Term, Text: roles[i]}, Value: []format.Value{v}})}}}
if pb := frame.params[format.Key("In")]; pb != nil {for _, v := range pb.pred {its := []format.Value{v}; if v.Kind == format.Array {its = v.Items}
	for _, t := range its {if r := []rune(t.Text); t.Kind != format.Term || len(r) == 0 || !unicode.IsUpper(r[0]) {return none, fmt.Errorf("%s.In : %s — в In имена ролей с прописной, а не значения; значение подаётся при вызове: %s [ … ]", name, format.Canon(pb.pred), name)}}}}
if args.Kind == format.Array && len(args.Items) > len(roles) {return none, fmt.Errorf("%s: подано значений %d, ролей %d (%s.In : [ %s ])", name, len(args.Items), len(roles), name, strings.Join(roles, " "))}
for _, r := range roles {
	given := frame.params[format.Key(r)] != nil
	for _, e := range es {given = given || format.SameTerm(e.Key.Text, r)}
	if !given {return none, fmt.Errorf("%s: нет значения роли %s — подайте при вызове: %s [ %s ], или задайте по умолчанию: %s.%s : …", name, r, name, strings.Join(roles, " "), name, r)}}
for _, r := range roles {if pb := frame.params[format.Key(r)]; pb != nil {inst.vars[format.Key(r)] = pb}}
saved, local := map[string]*binding{}, frame.params[format.Key("In")] != nil || frame.params[format.Key("Ex")] != nil
if local {
	for k, b := range frame.params {saved[k] = b}
	if in.calls == nil {in.calls = map[*binding]int{}}
	in.calls[frame]++
	defer func() {frame.params = saved; if in.calls[frame]--; in.calls[frame] == 0 {delete(in.calls, frame)}}()
} else {defer func() {for k, old := range saved {if old == nil {delete(frame.params, k)} else {frame.params[k] = old}}}()}
for _, e := range es {
	if len(e.Value) == 1 && e.Value[0].Kind == format.Expr {v, err := in.evalOperand(e.Value, env, false); if err != nil {return none, err}; e.Value = []format.Value{v}}
	k, b := format.Key(e.Key.Text), &binding{pred: e.Value, env: env}
	if _, ok := saved[k]; !ok && !local {saved[k] = frame.params[k]}
	inst.vars[k], frame.params[k] = b, b}
if err := in.step(); err != nil {return none, err}
r, _, err := in.block(frame.pred[0].Items, inst)
if err != nil {return r, err}
if pb := frame.params[format.Key("Ex")]; pb != nil && len(pb.pred) == 1 && pb.pred[0].Kind == format.Term {
	if xb := frame.params[format.Key(pb.pred[0].Text)]; xb != nil {r, _, err = in.block(xb.pred, xb.env); return r, err}; return none, nil}
if ex := roleNames(frame, "Ex"); len(ex) > 0 {
	v := format.Value{Kind: format.Object, Items: []format.Value{}}
	for _, n := range ex {
		x := none
		if pb := frame.params[format.Key(n)]; pb != nil {if x, _, err = in.block(pb.pred, pb.env); err != nil {return none, err}}
		v.Items = append(v.Items, format.Value{Kind: format.Term, Text: n}, format.Value{Kind: format.Op, Text: ":"}, x)}; return v, nil}
return r, nil }
// roleNames: имена из параметра key фрейма — блок терминов или один термин (In, Ex)
func roleNames(b *binding, key string) []string {
pb := b.params[format.Key(key)]
if pb == nil {return nil}
var out []string
for _, v := range pb.pred {
	if v.Kind == format.Term {out = append(out, v.Text)}
	if v.Kind == format.Array {for _, t := range v.Items {if t.Kind == format.Term {out = append(out, t.Text)}}}}; return out }
// upperFirst: первая буква прописная
func upperFirst(s string) string {
r := []rune(s)
if len(r) > 0 {r[0] = unicode.ToUpper(r[0])}; return string(r) }
// caseHint: подсказка — неопределённый терм со строчной читается как препозиция, а имелся в виду, вероятно, объект
func caseHint(toks []format.Value, env *Env) string {
for _, t := range toks {if t.Kind == format.Term && format.IsPrepTerm(t.Text) {if b, _ := env.find(t.Text); b == nil {return fmt.Sprintf(" («%s» со строчной буквы — препозиция; объект пишется с прописной: «%s»)", t.Text, upperFirst(t.Text))}}}; return "" }
// splitField: «Фрейм.Параметр» → фрейм и параметр, оба с прописной
func splitField(t string) (frame, param string, ok bool) {
parts := strings.Split(t, ".")
if len(parts) != 2 || format.IsPrepTerm(parts[0]) || format.IsPrepTerm(parts[1]) {return "", "", false}; return parts[0], parts[1], true }
// defParam: «Фрейм.Параметр : значение» — связывается без вычисления, ( ) — сейчас; фрейма нет — появляется; пишется в свой фрейм, не в синоним
func (in *Interp) defParam(s *slot, toks []format.Value, env *Env) (format.Value, error) {
frame, param, ok := splitField(s.def.Text)
if !ok {in.unknown(s.toks(), "— параметр пишется «Фрейм.Параметр», оба с прописной буквы"); return none, nil}
if len(toks) == 1 && toks[0].Kind == format.Expr {
	v, err := in.evalOperand(toks, env, false)
	if err != nil {return none, err}
	toks = []format.Value{v}}
fb, _ := env.find(frame)
if fb == nil {
	fb = &binding{env: env}
	env.set(frame, fb)}
if fb.params == nil {fb.params = map[string]*binding{}}
fb.params[format.Key(param)] = &binding{pred: toks, env: env}
if in.OnParam != nil && in.calls[fb] == 0 {if err := in.OnParam(frame, param, toks); err != nil {return none, err}}; return none, nil }
// param: значение «Фрейм.Параметр» — модель параметра интерпретируется при обращении
func (in *Interp) param(a []format.Value, env *Env) (format.Value, error) {
frame, param, ok := splitField(a[0].Text)
if !ok {in.unknown(a, "— параметр пишется «Фрейм.Параметр», оба с прописной буквы"); return none, nil}
fb, _ := env.findFrame(frame)
if fb == nil {in.unknown(a, fmt.Sprintf("— фрейм «%s» не определён", frame)); return none, nil}
pb := fb.params[format.Key(param)]
if pb == nil {in.unknown(a, fmt.Sprintf("— у фрейма «%s» нет параметра «%s»", frame, param)); return none, nil}
r, _, err := in.block(pb.pred, pb.env); return r, err }
// frameValue: фрейм как значение — объектный блок его параметров
func frameValue(b *binding) format.Value {
keys := make([]string, 0, len(b.params))
for k := range b.params {keys = append(keys, k)}
sort.Strings(keys)
v := format.Value{Kind: format.Object, Items: []format.Value{}}
for _, k := range keys {
	v.Items = append(v.Items, format.Value{Kind: format.Term, Text: k}, format.Value{Kind: format.Op, Text: ":"})
	v.Items = append(v.Items, b.params[k].pred...)}; return v }
// findFrame: фрейм по имени или синониму (Alias) и имя самого фрейма; собственное имя важнее синонима
func (e *Env) findFrame(name string) (*binding, string) {
if b, _ := e.find(name); b != nil {return b, name}
for c := e; c != nil; c = c.parent {
	keys := make([]string, 0, len(c.vars))
	for k := range c.vars {keys = append(keys, k)}
	sort.Strings(keys)
	for _, k := range keys {if p := c.vars[k].params[aliasParam]; p != nil && hasTerm(p.pred, name) {return c.vars[k], k}}}; return nil, name }
// hasTerm: предикация — терм name или блок, где он есть
func hasTerm(pred []format.Value, name string) bool {
for _, v := range pred {
	switch v.Kind {
	case format.Term:
		if format.SameTerm(v.Text, name) {return true}
	case format.Array:
		if hasTerm(v.Items, name) {return true}}}; return false }
