// epi/script/disp/disp_test.go
package disp
import ("errors"; "strings"; "testing"; "epi/script/format")
// run: интерпретировать текст; ответы на вопросы системы — по очереди из answers
func run(t *testing.T, text string, limit int, answers ...bool) (string, *Interp, []string, error) {
t.Helper()
var asked []string
in := New(limit, func(q string) bool {
	asked = append(asked, q)
	if len(answers) == 0 {t.Fatalf("неожиданный вопрос: %s", q)}
	a := answers[0]
	answers = answers[1:]; return a})
v, err := in.Run(text, NewEnv(nil)); return v.String(), in, asked, err }
func expect(t *testing.T, cases map[string]string) {
t.Helper()
for src, want := range cases {
	got, _, _, err := run(t, src, 100)
	if err != nil || got != want {t.Errorf("%q: %q %v, ожидалось %q", src, got, err, want)}} }
func TestArithmetic(t *testing.T) {
expect(t, map[string]string{
	`( 2 + 3 )`:           "5",
	`2 + 3`:               "5",
	`7 - 10`:              "-3",
	`- 4`:                 "-4",
	`6 * 7`:               "42",
	`7 / 2`:               "3.5",
	`1.5 + 1`:             "2.5",
	`( 2 + 3 ) * 4`:       "20",
	`10 > 3`:              "@true",
	`3 >= 4`:              "@false",
	`"a" = "A"`:           "@true",
	`2 <> 2.0`:            "@false",
	`X.A : 5 . X.A * X.A`: "25",}) }
// Модель: истинный левый слот — правый слот и конец блока; ложный — дальше
func TestModel(t *testing.T) {
expect(t, map[string]string{
	`10 > 3 -> 'ok`:                                 "ok",
	`1 > 3 -> 'a . 2 > 3 -> 'b . 'c`:                "c",
	`1 > 3 -> 'a . 5 > 3 -> 'b . 'c`:                "b",
	`X.A : 2 . X.A > 1 -> [ X.B : 10 . X.B + X.A ]`: "12",
	`X.A : 2 . X.A > 1 -> X.B : 7 . X.B`:            "@none",})
_, _, _, err := run(t, `5 -> 'a`, 100)
if err == nil || !strings.Contains(err.Error(), "не флаг") {t.Errorf("левый слот не флаг: %v", err)} }
// Цикл — фрейм, применяющий сам себя; « : » связывает, ( ) вычисляет сейчас
func TestLoop(t *testing.T) {
src := "C.I : 0\nC.Sum : 0\nCount : [\n    C.I < 10 -> [ C.I : (C.I + 1) . C.Sum : (C.Sum + C.I) . Count ]\n]\nCount\nC.Sum"
got, _, _, err := run(t, src, 1000)
if err != nil || got != "55" {t.Fatalf("цикл: %q %v", got, err)} }
// Связывание без вычисления: живая формула и алиас
func TestBinding(t *testing.T) {
expect(t, map[string]string{
	`F.A : 1 . F.B : F.A + 10 . F.A : 5 . F.B`:   "15",
	`F.A : 1 . F.B : (F.A + 10) . F.A : 5 . F.B`: "11",}) }
// Фрейм-интерпретатор со строчной буквы — препозиция в начале предикации;
// фрейм-объект с прописной — аппликация по имени
func TestFrames(t *testing.T) {
expect(t, map[string]string{
	"dupla : [ X * 2 ]\nF.N : 4\ndupla { X : F.N + 1 }": "10",
	`Twice : [ X * 2 ] . Twice { X : 21 }`:              "42",}) }
// Нераспознанная предикация — не ошибка: вопрос и запись в базу
func TestUnknown(t *testing.T) {
cases := map[string]string{
	`7 // 2`:                    "7 // 2",     // нет интерпретатора символа
	`7 foo 2`:                   "7 foo 2",    // нет интерпретатора препозиции-терма
	`"a" + 1`:                   `"a" + 1`,    // нет интерпретации для таких аргументов
	`Frobnicate`:                "Frobnicate", // имя не определено
	`1 +`:                       "1 +",        // препозиция в конце
	`1 + 2 + 3`:                 "1 + 2 + 3",  // две препозиции в слоте
	`1 / 0`:                     "1 / 0",
	`print`:                     "print",     // препозиция без операнда
	`x : 5`:                     "x : 5",     // препозиция определяется алиасом или фреймом
	`dupla : [ X ] . 1 dupla 2`: "1 dupla 2",} // фрейм-препозиция — только в начале
for src, rec := range cases {
	_, in, asked, err := run(t, src, 100, true)
	if err != nil || len(asked) != 1 || len(in.Unknown) != 1 || in.Unknown[0] != rec {t.Errorf("%q: записано %q, вопросы %q, ошибка %v", src, in.Unknown, asked, err)}}
got, in, _, err := run(t, `7 // 2 . 3 + 4`, 100, false)
if err != nil || len(in.Unknown) != 0 || got != "7" {t.Errorf("отказ от записи, интерпретация продолжается: %q %v %v", got, in.Unknown, err)} }
// Лимит шагов: по исчерпании — вопрос, продолжать ли
func TestLimit(t *testing.T) {
_, _, asked, err := run(t, `Loop : [ @true -> [ Loop ] ] . Loop`, 50, true, false)
var st Stopped
if !errors.As(err, &st) || len(asked) != 2 || st.Steps != 100 {t.Fatalf("лимит: %v, вопросов %d", err, len(asked))}
got, _, asked, err := run(t, "S.I : 0\nC : [ S.I < 30 -> [ S.I : (S.I + 1) . C ] ]\nC\nS.I", 50, true, true, true, true, true, true)
if err != nil || got != "30" || len(asked) == 0 {t.Fatalf("продолжение после лимита: %q %v %d", got, err, len(asked))} }
// Регистр значим по первой букве; буквы любого алфавита
func TestNames(t *testing.T) {
expect(t, map[string]string{
	"Мера.Число : 5\nудвоить : [ X * 2 ]\nудвоить { X : Мера.Число }": "10",
	`Мера.Число : 5 . МЕРА.ЧИСЛО + 1`:                                 "6",})
_, in, _, _ := run(t, `Мера.Число : 5 . число + 1`, 100, true)
if len(in.Unknown) != 1 {t.Errorf("«число» со строчной — препозиция, не объект «Число»: %v", in.Unknown)} }
// Алиасы препозиций: «имя : препозиция»
func TestAliases(t *testing.T) {
expect(t, map[string]string{
	`plus : + . 2 plus 3`:                 "5",
	`minus : - . minus 4`:                 "-4",
	`plus : + . summa : plus . 2 summa 3`: "5",
	`major : > . 10 major 3 -> 'ok`:       "ok",
	`F.X : (- 5) . F.X`:                   "-5",
	`plus : + . plus : - . 7 plus 2`:      "5",})
_, in, _, err := run(t, `pow : // . 2 pow 3`, 100, true)
if err != nil || len(in.Unknown) != 1 || in.Unknown[0] != "2 pow 3" {t.Errorf("алиас без интерпретатора: %v %v", in.Unknown, err)} }
// Граница предикаций — точка « . » или перевод строки; препозиция не
// может стоять в конце предикации
func TestBoundaries(t *testing.T) {
expect(t, map[string]string{
	`F.X : 5 - 4 . F.X`: "1",
	"F.X : 5\n- 4\nF.X": "5",
	"F.X : 5\n- 4":      "-4",
	`F.X : 5 . - 4`:     "-4",
	`F.X : 1 . . F.X .`: "1",
	"F.X : 1\n\n\nF.X":  "1",})
got, in, _, _ := run(t, "F.A : 2 plus\n3", 100, true)
if got != "3" || len(in.Unknown) != 1 || in.Unknown[0] != "F.A : 2 plus" {t.Errorf("препозиция в конце строки — конец предикации: %q %v", got, in.Unknown)}
_, in, _, _ = run(t, `F.I : 1 Count`, 100, true)
if len(in.Unknown) != 1 {t.Errorf("две предикации в одной части без точки — не распознано: %v", in.Unknown)} }
// Фреймы-интерпретаторы: арифметика и логика — разные фреймы
func TestFramesUsed(t *testing.T) {
in := New(100, nil)
for src, want := range map[string]string{
	`2 + 3`:         "arifm",
	`2 > 3`:         "logic",
	`( 2 + 3 ) > 4`: "arifm logic",
	`'a`:            "",} {
	if _, err := in.Run(src, NewEnv(nil)); err != nil {t.Fatal(err)}
	if got := strings.Join(in.Used, " "); got != want {t.Errorf("%q: %q, ожидалось %q", src, got, want)}} }
// Лимит шагов — на одну интерпретацию: после остановленной следующая начинается заново
func TestLimitPerRun(t *testing.T) {
in := New(50, func(string) bool { return false })
env := NewEnv(nil)
in.Run(`Loop : [ @true -> [ Loop ] ]`, env)
if _, err := in.Run(`Loop`, env); err == nil {t.Fatal("цикл должен остановиться")}
v, err := in.Run(`2 + 3`, env)
if err != nil || v.String() != "5" {t.Fatalf("после остановки: %v %v", v, err)} }
// Параметры фрейма: «Фрейм.Параметр»; параметров вне фрейма нет
func TestParams(t *testing.T) {
expect(t, map[string]string{
	`X.A : 5 . X.A * X.A`:                        "25",
	`X.A : 5 . X.B : 7 . X`:                      "{ A : 5 B : 7 }",
	`X.A : 2 . X.B : X.A * 3 . X.A : 10 . X.B`:   "30",
	`X.A : 2 . X.B : (X.A * 3) . X.A : 10 . X.B`: "6",
	`Q.A : 2 . Q.A > 1 -> 'ok`:                   "ok",})
for src, rec := range map[string]string{
	`X : 5`:         "X : 5",
	`X.B`:           "X.B",
	`X.A : 1 . X.B`: "X.B",
	`x.A : 1`:       "x.A : 1",
	`X.a : 1`:       "X.a : 1",} {
	_, in, _, err := run(t, src, 100, true)
	if err != nil || len(in.Unknown) != 1 || in.Unknown[0] != rec {t.Errorf("%q: записано %q, ошибка %v", src, in.Unknown, err)}} }
// Интерпретаторы: «и», «или» в logic; trgnm; lnrfm; алиасы к термам ядра
func TestNewInterpreters(t *testing.T) {
cases := []struct{ src, want, frames string }{
	{`( 2 > 1 ) & ( 3 > 2 )`, "@true", "logic"},
	{`( 2 > 1 ) & ( 3 < 2 )`, "@false", "logic"},
	{`( 2 > 1 ) | ( 3 < 2 )`, "@true", "logic"},
	{`sin 0`, "0.0", "trgnm"},
	{`acos 1`, "0.0", "trgnm"},
	{`atan 1 * 4`, "", ""}, // две препозиции в слоте — не распознано
	{`( atan 1 ) * 4`, "3.141592653589793", "trgnm arifm"},
	{`ln 1`, "0.0", "lnrfm"},
	{`exp 0`, "1.0", "lnrfm"},
	{`ln ( exp 2 )`, "2.0", "lnrfm"},
	{`arcsin : asin . arcsin 1`, "1.5707963267948966", "trgnm"},}
for _, c := range cases {
	in := New(100, func(string) bool { return true })
	v, err := in.Run(c.src, NewEnv(nil))
	if c.want == "" {if len(in.Unknown) == 0 {t.Errorf("%s: ожидалось «не распознано», получено %v", c.src, v)}; continue}
	if err != nil || v.String() != c.want || strings.Join(in.Used, " ") != c.frames {t.Errorf("%s = %v %v (%v), ожидалось %s (%s)", c.src, v, err, in.Used, c.want, c.frames)}}
for _, src := range []string{`ln 0`, `ln -1`, `asin 2`, `1 & 2`, `2 sin 3`} {
	in := New(100, func(string) bool { return true })
	in.Run(src, NewEnv(nil))
	if len(in.Unknown) != 1 {t.Errorf("%s: ожидалось «не распознано»", src)}} }
// Синоним концепта — параметр Alias
func TestAliasParam(t *testing.T) {
in := New(100, func(string) bool { return true })
env := NewEnv(nil)
var saved []string
in.OnParam = func(frame, param string, pred []format.Value) {saved = append(saved, frame+"."+param+" : "+format.Format(pred))}
for _, c := range []struct{ src, want string }{
	{`X.A : 5 . X.Alias : [ Y Z ]`, "@none"},
	{`Y.A * Z.A`, "25"},
	{`Z.B : 7`, "@none"},
	{`X.B`, "7"},
	{`Y`, "{ A : 5 Alias : [ Y Z ] B : 7 }"},
	{`W.A`, "@none"},} {
	v, err := in.Run(c.src, env)
	if err != nil || v.String() != c.want {t.Errorf("%s = %v %v, ожидалось %s", c.src, v, err, c.want)}}
if got := strings.Join(saved, "|"); got != "X.A : 5|X.Alias : [ Y Z ]|X.B : 7" {t.Errorf("запись: %s", got)}
if len(in.Unknown) != 1 {t.Errorf("W.A — не распознано: %v", in.Unknown)} }
// Тело фрейма не стирает его параметры (концепт — параметры и импликации)
func TestFrameKeepsParams(t *testing.T) {
in := New(100, func(string) bool { return true })
var body string
in.OnFrame = func(name string, b []format.Value) {body = name + " : " + format.Format(b)}
v, err := in.Run("F.A : 1 . F : [ F.A < 3 -> [ F.A : ( F.A + 1 ) . F ] ] . F . F.A", NewEnv(nil))
if err != nil || v.String() != "3" {t.Fatalf("F.A = %v %v", v, err)}
if body != "F : F.A < 3 -> [ F.A : ( F.A + 1 ) . F ]" {t.Fatalf("тело: %q", body)} }
// Цикл второго уровня: recur (zycle)
func TestRecur(t *testing.T) {
in := New(100, func(string) bool { return true })
env := NewEnv(nil)
v, err := in.Run("S.Sum : 0 . S.L : [ 1 2 3 4 ] . recur { In : S.L As : S.I Corpus : [ S.Sum : ( S.Sum + S.I ) ] } . S.Sum", env)
if err != nil || v.String() != "10" {t.Fatalf("S.Sum = %v %v", v, err)}
v, _ = in.Run("recur { In : [ 2 3 ] As : S.I Corpus : [ S.I * S.I ] }", env)
if v.String() != "9" || strings.Join(in.Used, " ") != "arifm zycle" {t.Fatalf("последний шаг = %v (%v)", v, in.Used)}
in.Run("recur { In : 5 As : S.I Corpus : [ S.I ] }", env)
if len(in.Unknown) != 1 {t.Fatalf("In не блок — не распознано: %v", in.Unknown)} }
