// epi/script/proba/proba_test.go
// proba: общие помощники тестов ядра и тест заскриптованных функций — файлы *.uic, строка «[ предикация ] -> ответ»
package proba
import ("os"; "path/filepath"; "strings"; "testing"; "epi/script/disp")
// runText: интерпретировать текст; ответы на вопросы системы — по очереди из answers
func runText(t *testing.T, text string, limit int, answers ...bool) (string, *disp.Interp, []string, error) {
t.Helper()
var asked []string
in := disp.New(limit, func(q string) bool {
	asked = append(asked, q)
	if len(answers) == 0 {t.Fatalf("неожиданный вопрос: %s", q)}
	a := answers[0]
	answers = answers[1:]; return a})
v, err := in.Run(text, disp.NewEnv(nil)); return v.String(), in, asked, err }
func expect(t *testing.T, cases map[string]string) {
t.Helper()
for src, want := range cases {
	got, _, _, err := runText(t, src, 100)
	if err != nil || got != want {t.Errorf("%q: %q %v, ожидалось %q", src, got, err, want)}} }
// TestScripta: заскриптованные функции — файлы *.uic папки proba, каждый в своём контексте и со своей пустой папкой данных (scrin); строка «[ предикация ] -> ответ»,
// ответ — канонический текст интерпретации, «! "текст"» — ошибка, в сообщении которой есть этот текст,
// «? "текст"» — не распознано: вопрос «записать?», в котором есть этот текст
func TestScripta(t *testing.T) {
files, _ := filepath.Glob("*.uic")
if len(files) == 0 {t.Fatal("нет файлов *.uic")}
for _, f := range files {
	b, err := os.ReadFile(f)
	if err != nil {t.Fatal(err)}
	in, env := disp.New(10000, nil), disp.NewEnv(nil)
	in.Root = t.TempDir()
	var asked []string
	in.Ask = func(q string) bool {asked = append(asked, q); return false}
	for n, line := range strings.Split(string(b), "\n") {
		if strings.TrimSpace(line) == "" {continue}
		pred, want, ok := casus(line)
		if !ok {t.Errorf("%s:%d: строка не вида «[ предикация ] -> ответ»", f, n+1); continue}
		asked = nil
		v, err := in.Run(pred, env)
		switch {
		case strings.HasPrefix(want, "?"):
			w := strings.Trim(strings.TrimSpace(strings.TrimPrefix(want, "?")), `"`)
			if len(asked) == 0 || !strings.Contains(strings.Join(asked, " "), w) {t.Errorf("%s:%d: %s = %v %v, ждём вопрос «%s», вопросы: %v", f, n+1, pred, v, err, w, asked)}
		case strings.HasPrefix(want, "!"):
			w := strings.Trim(strings.TrimSpace(strings.TrimPrefix(want, "!")), `"`)
			if err == nil || !strings.Contains(err.Error(), w) {t.Errorf("%s:%d: %s = %v %v, ждём ошибку «%s»", f, n+1, pred, v, err, w)}
		case err != nil || v.String() != want: t.Errorf("%s:%d: %s = %v %v, ждём %s", f, n+1, pred, v, err, want)}}}}
// casus: строка теста — предикация внутри внешних [ ] и ответ после « -> »; скобки внутри "…" и ''…'' не считаются
func casus(line string) (string, string, bool) {
line = strings.TrimSpace(line)
if !strings.HasPrefix(line, "[") {return "", "", false}
depth, str, long := 0, false, false
for i, c := range line {
	switch {
	case !str && strings.HasPrefix(line[i:], "''"): long = !long
	case long:
	case c == '"': str = !str
	case str:
	case c == '[' || c == '(' || c == '{': depth++
	case c == ']' || c == ')' || c == '}': depth--
		if depth == 0 {rest := strings.TrimSpace(line[i+1:]); if !strings.HasPrefix(rest, "->") {return "", "", false}; return strings.TrimSpace(line[1:i]), strings.TrimSpace(rest[2:]), true}}}; return "", "", false }
