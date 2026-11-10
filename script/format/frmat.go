// epi/script/format/frmat.go
// Package format: текст ASON в дерево значений и дерево обратно в канонический текст
package format
import ("fmt"; "regexp"; "strings"; "unicode")
// String: имя вида значения
func (k Kind) String() string { return kindNames[k] }
// Error: ошибка разбора с местом
func (e *Error) Error() string {return fmt.Sprintf("строка %d, позиция %d: %s", e.Line, e.Col, e.Msg) }
// errAt: ошибка в строке line, позиции col
func (p *parser) errAt(line, col int, f string, a ...any) error {return &Error{line, col, fmt.Sprintf(f, a...)} }
// peek: следующая литера без сдвига
func (p *parser) peek() rune {
if p.pos < len(p.src) {return p.src[p.pos]}; return 0 }
// next: следующая литера со сдвигом, счёт строк и позиций
func (p *parser) next() rune {
r := p.src[p.pos]
p.pos++
if r == '\n' {p.line, p.col = p.line+1, 1} else {p.col++}; return r }
// isSpace, isDelim, closing: пробел, разделитель, закрывающая скобка блока
func isSpace(r rune) bool { return r == ' ' || r == '\t' || r == '\n' || r == '\r' }
func isDelim(r rune) bool { return isSpace(r) || strings.ContainsRune("[](){}", r) }
func closing(k Kind) rune { return map[Kind]rune{Array: ']', Object: '}', Expr: ')'}[k] }
// Parse: весь текст — значения верхнего уровня; при ошибке ввод отвергается целиком
func Parse(text string) ([]Value, error) {
p := &parser{src: []rune(text), line: 1, col: 1}
items, err := p.seq(0, 0, 0)
if err != nil {return nil, err}; return items, nil }
// seq: значения до закрывающей скобки end (0 — до конца текста)
func (p *parser) seq(end rune, line, col int) ([]Value, error) {
var items []Value
for {
	for p.pos < len(p.src) && isSpace(p.peek()) {p.next()}
	if p.pos >= len(p.src) {if end != 0 {return nil, p.errAt(line, col, "блок не закрыт: нет «%c»", end)}; return items, nil}
	r := p.peek()
	if r == end && end != 0 {p.next(); return items, nil}
	if strings.ContainsRune("])}", r) {return nil, p.errAt(p.line, p.col, "лишняя закрывающая скобка «%c»", r)}
	v, err := p.value()
	if err != nil {return nil, err}
	v.End = p.line
	items = append(items, v)} }
// value: одно значение — блок, строка, тег или атом; определение пишется через пробел «Ключ :»
func (p *parser) value() (Value, error) {
line, col := p.line, p.col
r := p.peek()
switch r {
case '[', '(', '{':
	k := map[rune]Kind{'[': Array, '(': Expr, '{': Object}[r]
	p.next()
	items, err := p.seq(closing(k), line, col)
	if err != nil {return Value{}, err}
	if k == Object {if err := checkObject(items, line, col); err != nil {return Value{}, err}}
	if items == nil {items = []Value{}}; return Value{Kind: k, Items: items, Line: line, Col: col}, nil
case '"':
	p.next()
	s, err := p.quoted(line, col); return Value{Kind: String, Text: s, Line: line, Col: col}, err}
rest := string(p.src[p.pos:min(p.pos+3, len(p.src))])
switch {
case strings.HasPrefix(rest, "''"):
	p.next()
	p.next()
	start := p.pos
	for {
		if p.pos+1 >= len(p.src) {return Value{}, p.errAt(line, col, "длинная строка не закрыта: нет «''»")}
		if p.src[p.pos] == '\'' && p.src[p.pos+1] == '\'' {
			s := string(p.src[start:p.pos])
			p.next()
			p.next(); return p.after(Value{Kind: LongString, Text: s, Line: line, Col: col})}
		p.next()}
case strings.HasPrefix(rest, `##"`), strings.HasPrefix(rest, `#"`):
	k := Binary16
	if rest[1] == '#' {
		k = Binary64
		p.next()}
	p.next()
	p.next()
	s, err := p.quoted(line, col)
	if err != nil {return Value{}, err}
	ok := regexp.MustCompile(`^[0-9A-Fa-f]*$`).MatchString(s)
	if k == Binary64 {ok = regexp.MustCompile(`^[A-Za-z0-9+/=]*$`).MatchString(s)}
	if !ok {return Value{}, p.errAt(line, col, "недопустимые символы в двоичном значении %q", s)}; return Value{Kind: k, Text: s, Line: line, Col: col}, nil
case r == '<' && p.pos+1 < len(p.src) && (unicode.IsLetter(p.src[p.pos+1]) || p.src[p.pos+1] == '/'):
	start := p.pos
	for p.pos < len(p.src) && p.peek() != '>' {
		if p.peek() == '\n' {return Value{}, p.errAt(line, col, "тег не закрыт: нет «>»")}
		p.next()}
	if p.pos >= len(p.src) {return Value{}, p.errAt(line, col, "тег не закрыт: нет «>»")}
	p.next(); return p.after(Value{Kind: Tag, Text: string(p.src[start+1 : p.pos-1]), Line: line, Col: col})}
start := p.pos
for p.pos < len(p.src) && !isDelim(p.peek()) {
	if p.peek() == '"' {return Value{}, p.errAt(p.line, p.col, "кавычка внутри токена")}
	p.next()}
tok := string(p.src[start:p.pos])
if Operators[tok] {return Value{Kind: Op, Text: tok, Line: line, Col: col}, nil}
if len(tok) > 1 && strings.HasSuffix(tok, ":") {return Value{}, p.errAt(line, col, "определение пишется через пробел: «%s :»", strings.TrimSuffix(tok, ":"))}
for _, a := range atoms {if a.re.MatchString(tok) {return Value{Kind: a.kind, Text: tok, Line: line, Col: col}, nil}}; return Value{}, p.errAt(line, col, "непонятный токен %q", tok) }
// after: за строкой или тегом нет слитного продолжения
func (p *parser) after(v Value) (Value, error) {
if p.pos < len(p.src) && !isDelim(p.peek()) {return Value{}, p.errAt(p.line, p.col, "после значения нужен пробел или скобка")}; return v, nil }
// quoted: содержимое строки в кавычках
func (p *parser) quoted(line, col int) (string, error) {
var b strings.Builder
for {
	if p.pos >= len(p.src) {return "", p.errAt(line, col, "строка не закрыта: нет «\"»")}
	r := p.next()
	switch r {
	case '"':
		_, err := p.after(Value{}); return b.String(), err
	case '\n': return "", p.errAt(line, col, "перевод строки внутри «\"…\"»: для многострочного текста — ''…''")
	case '\\':
		if p.pos >= len(p.src) {return "", p.errAt(line, col, "строка не закрыта: нет «\"»")}
		l, c := p.line, p.col
		e := p.next()
		m := map[rune]rune{'"': '"', '\\': '\\', 'n': '\n', 't': '\t'}
		x, ok := m[e]
		if !ok {return "", p.errAt(l, c-1, "неизвестное экранирование «\\%c»", e)}
		b.WriteRune(x)
	default:
		b.WriteRune(r)}} }
// IsDef: на позиции i начинается определение «Ключ : …»
func IsDef(items []Value, i int) bool {return i+1 < len(items) && (items[i].Kind == Term || items[i].Kind == Field) &&
	items[i+1].Kind == Op && items[i+1].Text == ":" }
// checkObject: объектный блок — определения «Ключ : значение», значение — до следующего определения
func checkObject(items []Value, line, col int) error {
if len(items) > 0 && !IsDef(items, 0) {return &Error{items[0].Line, items[0].Col, "объектный блок должен начинаться с определения «Ключ : значение»"}}
for i := 0; i < len(items); i++ {
	if IsDef(items, i) {
		if i+2 == len(items) || IsDef(items, i+2) {return &Error{items[i].Line, items[i].Col, fmt.Sprintf("у «%s» нет значения", items[i].Text)}}
		i++}}; return nil }
// Entries: объектный блок на пары «ключ — значение»
func (v Value) Entries() []Entry {
var es []Entry
for i := 0; i < len(v.Items); i++ {
	if IsDef(v.Items, i) {
		es = append(es, Entry{Key: v.Items[i]})
		i++} else if len(es) > 0 {es[len(es)-1].Value = append(es[len(es)-1].Value, v.Items[i])}}; return es }
// Format: значения в канонический текст; перевод строки между значениями сохраняется (отделяет предикации)
func Format(vs []Value) string {
var b strings.Builder
for i, v := range vs {
	if i > 0 {
		if v.Line > vs[i-1].End && vs[i-1].End > 0 {b.WriteString("\n")} else {b.WriteString(" ")}}
	b.WriteString(v.String())}; return b.String() }
// String: значение в канонический текст
func (v Value) String() string {
switch v.Kind {
case Array, Object, Expr:
	open := map[Kind]string{Array: "[", Object: "{", Expr: "("}[v.Kind]
	if len(v.Items) == 0 {return open + string(closing(v.Kind))}; return open + " " + Format(v.Items) + " " + string(closing(v.Kind))
case String:
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`, "\t", `\t`); return `"` + r.Replace(v.Text) + `"`
case LongString: return "''" + v.Text + "''"
case Tag: return "<" + v.Text + ">"
case Binary16: return `#"` + v.Text + `"`
case Binary64: return `##"` + v.Text + `"`}; return v.Text }
// Key: ключ терма — первая буква как написана (строчная — препозиция, прописная — объект), остальные без учёта регистра
func Key(s string) string {
r := []rune(s)
if len(r) == 0 {return s}; return string(r[0]) + strings.ToLower(string(r[1:])) }
// SameTerm: написания термов равны по Key
func SameTerm(a, b string) bool { return Key(a) == Key(b) }
// IsPrepTerm: терм — препозиция (со строчной буквы)
func IsPrepTerm(s string) bool {
for _, r := range s {return unicode.IsLower(r)}; return false }
