// epi/script/disp/disp_test.go
package disp
import ("errors"; "fmt"; "strings"; "testing"; "epi/script/format")
// runText: интерпретировать текст; ответы на вопросы системы — по очереди из answers
func runText(t *testing.T, text string, limit int, answers ...bool) (string, *Interp, []string, error) {
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
	got, _, _, err := runText(t, src, 100)
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
_, _, _, err := runText(t, `5 -> 'a`, 100)
if err == nil || !strings.Contains(err.Error(), "не флаг") {t.Errorf("левый слот не флаг: %v", err)} }
// Цикл — фрейм, применяющий сам себя; « : » связывает, ( ) вычисляет сейчас
func TestLoop(t *testing.T) {
src := "C.I : 0\nC.Sum : 0\nCount : [\n    C.I < 10 -> [ C.I : (C.I + 1) . C.Sum : (C.Sum + C.I) . Count ]\n]\nCount\nC.Sum"
got, _, _, err := runText(t, src, 1000)
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
	_, in, asked, err := runText(t, src, 100, true)
	if err != nil || len(asked) != 1 || len(in.Unknown) != 1 || in.Unknown[0] != rec {t.Errorf("%q: записано %q, вопросы %q, ошибка %v", src, in.Unknown, asked, err)}}
got, in, _, err := runText(t, `7 // 2 . 3 + 4`, 100, false)
if err != nil || len(in.Unknown) != 0 || got != "7" {t.Errorf("отказ от записи, интерпретация продолжается: %q %v %v", got, in.Unknown, err)} }
// Лимит шагов: по исчерпании — вопрос, продолжать ли
func TestLimit(t *testing.T) {
_, _, asked, err := runText(t, `Loop : [ @true -> [ Loop ] ] . Loop`, 50, true, false)
var st Stopped
if !errors.As(err, &st) || len(asked) != 2 || st.Steps != 100 {t.Fatalf("лимит: %v, вопросов %d", err, len(asked))}
got, _, asked, err := runText(t, "S.I : 0\nC : [ S.I < 30 -> [ S.I : (S.I + 1) . C ] ]\nC\nS.I", 50, true, true, true, true, true, true)
if err != nil || got != "30" || len(asked) == 0 {t.Fatalf("продолжение после лимита: %q %v %d", got, err, len(asked))} }
// Регистр значим по первой букве; буквы любого алфавита
func TestNames(t *testing.T) {
expect(t, map[string]string{
	"Мера.Число : 5\nудвоить : [ X * 2 ]\nудвоить { X : Мера.Число }": "10",
	`Мера.Число : 5 . МЕРА.ЧИСЛО + 1`:                                 "6",})
_, in, _, _ := runText(t, `Мера.Число : 5 . число + 1`, 100, true)
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
_, in, _, err := runText(t, `pow : // . 2 pow 3`, 100, true)
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
got, in, _, _ := runText(t, "F.A : 2 plus\n3", 100, true)
if got != "3" || len(in.Unknown) != 1 || in.Unknown[0] != "F.A : 2 plus" {t.Errorf("препозиция в конце строки — конец предикации: %q %v", got, in.Unknown)}
_, in, _, _ = runText(t, `F.I : 1 Count`, 100, true)
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
// Параметры фрейма: «Фрейм.Параметр»; параметров вне фрейма нет; «Фрейм.Параметр :» — пустое значение
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
	_, in, _, err := runText(t, src, 100, true)
	if err != nil || len(in.Unknown) != 1 || in.Unknown[0] != rec {t.Errorf("%q: записано %q, ошибка %v", src, in.Unknown, err)}}
if _, in, _, err := runText(t, `X.A : 5 . X.A :`, 100, true); err != nil || len(in.Unknown) != 0 {t.Errorf("«X.A :» — пустое значение: записано %q, ошибка %v", in.Unknown, err)} }
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
// Синоним концепта — параметр Alias: чтение через синоним, если своего фрейма нет; запись — в свой фрейм
func TestAliasParam(t *testing.T) {
in := New(100, func(string) bool { return true })
env := NewEnv(nil)
var saved []string
in.OnParam = func(frame, param string, pred []format.Value) error {saved = append(saved, frame+"."+param+" : "+format.Canon(pred)); return nil}
for _, c := range []struct{ src, want string }{
	{`X.A : 5 . X.Alias : [ Y Z ]`, "@none"},
	{`Y.A * Z.A`, "25"},
	{`Z.B : 7`, "@none"},
	{`Z.B`, "7"},
	{`X.B`, "@none"},
	{`Y`, "{ A : 5 Alias : [ Y Z ] }"},
	{`W.A`, "@none"},} {
	v, err := in.Run(c.src, env)
	if err != nil || v.String() != c.want {t.Errorf("%s = %v %v, ожидалось %s", c.src, v, err, c.want)}}
if got := strings.Join(saved, "|"); got != "X.A : 5|X.Alias : [ Y Z ]|Z.B : 7" {t.Errorf("запись: %s", got)}
if len(in.Unknown) != 2 {t.Errorf("X.B, W.A — не распознано: %v", in.Unknown)} }
// Тело фрейма не стирает его параметры (концепт — параметры и импликации)
func TestFrameKeepsParams(t *testing.T) {
in := New(100, func(string) bool { return true })
var body string
in.OnFrame = func(name string, b []format.Value) {body = name + " : " + format.Canon(b)}
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
v, _ = in.Run("S.N : 4 . recur { In : S.N As : S.I Corpus : [ S.I * 2 ] }", env)
if v.String() != "8" {t.Fatalf("In : S.N = %v", v)}
in.Run("recur { In : 2.5 As : S.I Corpus : [ S.I ] }", env)
if len(in.Unknown) != 1 {t.Fatalf("In не целое — не распознано: %v", in.Unknown)} }
// TestFrameRoles: роли входа In, позиционный и именованный вызов, умолчания, результат — Ex термином или блоком
func TestFrameRoles(t *testing.T) {
in := New(10000, nil); env := NewEnv(nil); var wrote []string
in.OnParam = func(f, p string, _ []format.Value) error {wrote = append(wrote, f+"."+p); return nil}
for _, c := range []struct{ src, want string }{
	{`Gradus.In : [ Basis Exponens ] . Gradus.Ex : Gradus`, "@none"},
	{`Gradus.Basis : 2 . Gradus.Exponens : 3`, "@none"},
	{`Gradus : [ Gradus.Gradus : 1 . recur { In : Gradus.Exponens As : Gradus.I Corpus : [ Gradus.Gradus : ( Gradus.Gradus * Gradus.Basis ) ] } ]`, "@none"},
	{`Gradus`, "8"},
	{`Gradus { Basis : 2 Exponens : 10 }`, "1024"},
	{`Gradus [ 3 4 ]`, "81"},
	{`Gradus.Basis`, "2"},
	{`Summa.In : [ A B ] . Summa.Ex : [ S D ]`, "@none"},
	{`Summa : [ Summa.S : ( Summa.A + Summa.B ) . Summa.D : ( Summa.A - Summa.B ) ]`, "@none"},
	{`Summa [ 7 2 ]`, "{ S : 9 D : 5 }"},
}{ v, err := in.Run(c.src, env); got := ""; if err == nil {got = format.Canon([]format.Value{v})} else {got = err.Error()}; t.Logf("%s => %s", c.src, got); if c.want != "" && got != c.want {t.Errorf("%s: %s, ждём %s", c.src, got, c.want)}}
if w := strings.Join(wrote, " "); w != "Gradus.In Gradus.Ex Gradus.Basis Gradus.Exponens Summa.In Summa.Ex" {t.Errorf("в файл записано: %s", w)}
if v, _ := in.Run("Gradus.I", env); v.String() != "@none" {t.Errorf("Gradus.I после вызова = %v", v)} }
// TestTriangulum: условия с & и |, переход в другой фрейм, ошибки вызова с подсказкой
func TestTriangulum(t *testing.T) {
in := New(10000, nil); env := NewEnv(nil)
for _, s := range []string{"Existens.In : [ A B C ]", "Existens : [ ( ( ( Existens.A + Existens.B ) > Existens.C ) & ( ( Existens.A + Existens.C ) > Existens.B ) ) & ( ( Existens.B + Existens.C ) > Existens.A ) ]", "Triangulum.In : [ A B C ]",
	"Triangulum : [ ( Existens [ Triangulum.A Triangulum.B Triangulum.C ] ) = @false -> 'nullum . ( ( Triangulum.A = Triangulum.B ) & ( Triangulum.B = Triangulum.C ) ) -> 'aequilaterum . ( ( ( Triangulum.A = Triangulum.B ) | ( Triangulum.B = Triangulum.C ) ) | ( Triangulum.A = Triangulum.C ) ) -> 'isosceles . 'scalenum ]", "Signum.In : [ 3 ] . Signum : [ 1 ]"} {
	if _, err := in.Run(s, env); err != nil {t.Fatal(s, err)}}
for _, c := range []struct{ src, want string }{{"Triangulum [ 1 2 10 ]", "nullum"}, {"Triangulum [ 5 5 5 ]", "aequilaterum"}, {"Triangulum [ 5 5 8 ]", "isosceles"}, {"Triangulum [ 3 4 5 ]", "scalenum"},
	{"Triangulum [ 3 4 ]", "нет значения роли C"}, {"Triangulum [ 3 4 5 6 ]", "подано значений 4, ролей 3"}, {"Signum [ 1 ]", "в In имена ролей"}} {
	v, err := in.Run(c.src, env); got := v.String(); if err != nil {got = err.Error()}
	if !strings.Contains(got, c.want) {t.Errorf("%s = %s, ждём %s", c.src, got, c.want)}}}
// TestTransitus: переход в другой фрейм — в условии (Existens, Rectus), результатом (Triangulum), в себя (рекурсия Factorialis, аргумент ( ) — сейчас)
func TestTransitus(t *testing.T) {
in := New(10000, nil); env := NewEnv(nil)
for _, s := range []string{"Existens.In : [ A B C ]", "Existens : [ ( ( ( Existens.A + Existens.B ) > Existens.C ) & ( ( Existens.A + Existens.C ) > Existens.B ) ) & ( ( Existens.B + Existens.C ) > Existens.A ) ]", "Triangulum.In : [ A B C ]",
	"Triangulum : [ ( ( Triangulum.A = Triangulum.B ) & ( Triangulum.B = Triangulum.C ) ) -> 'aequilaterum . ( ( ( Triangulum.A = Triangulum.B ) | ( Triangulum.B = Triangulum.C ) ) | ( Triangulum.A = Triangulum.C ) ) -> 'isosceles . 'scalenum ]",
	"Rectus.In : [ A B C ]", "Rectus : [ ( ( ( ( Rectus.A * Rectus.A ) + ( Rectus.B * Rectus.B ) ) = ( Rectus.C * Rectus.C ) ) | ( ( ( Rectus.A * Rectus.A ) + ( Rectus.C * Rectus.C ) ) = ( Rectus.B * Rectus.B ) ) ) | ( ( ( Rectus.B * Rectus.B ) + ( Rectus.C * Rectus.C ) ) = ( Rectus.A * Rectus.A ) ) ]",
	"Classis.In : [ A B C ]", "Classis : [ ( Existens [ Classis.A Classis.B Classis.C ] ) = @false -> 'nullum . ( Rectus [ Classis.A Classis.B Classis.C ] ) -> 'rectangulum . Triangulum [ Classis.A Classis.B Classis.C ] ]",
	"Factorialis.In : [ N ]", "Factorialis : [ ( Factorialis.N <= 1 ) -> 1 . ( Factorialis.N * Factorialis [ ( Factorialis.N - 1 ) ] ) ]"} {
	if _, err := in.Run(s, env); err != nil {t.Fatal(s, err)}}
for _, c := range []struct{ src, want string }{{"Classis [ 3 4 5 ]", "rectangulum"}, {"Classis [ 5 5 8 ]", "isosceles"}, {"Classis [ 1 2 10 ]", "nullum"}, {"Classis [ 6 6 6 ]", "aequilaterum"}, {"Factorialis [ 5 ]", "120"}, {"Factorialis [ 10 ]", "3628800"}} {
	if v, err := in.Run(c.src, env); err != nil || v.String() != c.want {t.Errorf("%s = %v %v, ждём %s", c.src, v, err, c.want)}}}
// TestSerie: интерпретатор serie — строки и списки; неверный операнд — не распознано
func TestSerie(t *testing.T) {
in := New(10000, nil); env := NewEnv(nil)
for _, c := range []struct{ src, want string }{{`dstnz "углами"`, "6"}, {`"углами" front 3`, `"угл"`}, {`"углами" post 3`, `"ами"`}, {`[ A B C ] elmnt 2`, "B"}, {`"угл" cnctn "ы"`, `"углы"`},
	{`cnvrg "Угол"`, `"угол"`}, {`"угол" front 10`, `"угол"`}, {`"угол" post 10`, `""`}, {`[ 1 2 ] elmnt 5`, "@none"}, {`[ 1 2 ] cnctn [ 3 ]`, "[ 1 2 3 ]"}, {`[ 1 2 ] cnctn 3`, "[ 1 2 3 ]"},
	{`( "Угловой" front 3 ) = ( "углами" front 3 )`, "@true"}, {`X.S : "кот" . ( X.S cnctn "ы" )`, `"коты"`}} {
	if v, err := in.Run(c.src, env); err != nil || v.String() != c.want {t.Errorf("%s = %v %v, ждём %s", c.src, v, err, c.want)}}
asked := 0; in.Ask = func(string) bool {asked++; return false}
for _, s := range []string{`"угол" front 1.5`, `dstnz 5`, `cnvrg [ A ]`, `"а" cnctn [ B ]`} {in.Run(s, env)}
if asked != 4 {t.Errorf("не распознано: %d из 4", asked)}}
// TestAnalog: wordEq (infra/lense.js, словари Defs) фреймом — слова равны по первым D литер, короче D — только точно; ожидания — ответы wordEq
func TestAnalog(t *testing.T) {
in := New(10000, nil); env := NewEnv(nil)
for _, s := range []string{"Analog.In : [ A B D ]", "Analog : [ ( ( ( dstnz Analog.A ) < Analog.D ) | ( ( dstnz Analog.B ) < Analog.D ) ) -> ( Analog.A = Analog.B ) . ( Analog.A front Analog.D ) = ( Analog.B front Analog.D ) ]"} {
	if _, err := in.Run(s, env); err != nil {t.Fatal(s, err)}}
for _, c := range []struct{ a, b string; d int; want bool }{{"угол", "углами", 3, false}, {"угол", "угл", 3, false}, {"ум", "ум", 3, true}, {"ум", "умный", 3, false}, {"Квадрат", "квадраты", 3, true},
	{"кот", "кит", 3, false}, {"модель", "модели", 5, true}, {"модель", "модели", 6, false}, {"a", "b", 1, false}, {"словарь", "слово", 4, true}} {
	v, err := in.Run(fmt.Sprintf("Analog [ %q %q %d ]", c.a, c.b, c.d), env)
	if err != nil || v.String() != flag(c.want).String() {t.Errorf("Analog [ %s %s %d ] = %v %v, wordEq — %v", c.a, c.b, c.d, v, err, c.want)}}}
// TestConcord: tokens и overlap (infra/nexus.go, темы на графе) фреймами Verb и Concord; Concord зовёт Analog; ожидания — ответы tokens и overlap
func TestConcord(t *testing.T) {
in := New(10000, nil); env := NewEnv(nil)
for _, s := range []string{"Analog.In : [ A B D ]", "Analog : [ ( ( ( dstnz Analog.A ) < Analog.D ) | ( ( dstnz Analog.B ) < Analog.D ) ) -> ( Analog.A = Analog.B ) . ( Analog.A front Analog.D ) = ( Analog.B front Analog.D ) ]",
	"Verb.In : [ S ] . Verb.Ex : Verb",
	`Verb : [ Verb.Verb : [] . Verb.W : "" . Verb.T : ( cnvrg Verb.S ) . recur { In : ( dstnz Verb.T ) As : Verb.I Corpus : [ Verb.C : ( Verb.T elmnt Verb.I ) . ( ( ( Verb.C = " " ) | ( Verb.C = "-" ) ) | ( Verb.C = "_" ) ) -> [ ( Verb.W <> "" ) -> [ Verb.Verb : ( Verb.Verb cnctn Verb.W ) . Verb.W : "" ] ] . Verb.W : ( Verb.W cnctn Verb.C ) ] } . ( Verb.W <> "" ) -> [ Verb.Verb : ( Verb.Verb cnctn Verb.W ) ] ]`,
	"Concord.In : [ A B D ] . Concord.Ex : Concord",
	"Concord : [ Concord.Concord : @false . recur { In : Concord.A As : Concord.X Corpus : [ recur { In : Concord.B As : Concord.Y Corpus : [ recur { In : Concord.X As : Concord.S Corpus : [ recur { In : Concord.Y As : Concord.T Corpus : [ ( Analog [ Concord.S Concord.T Concord.D ] ) -> [ Concord.Concord : @true ] ] } ] } ] } ] } ]"} {
	if _, err := in.Run(s, env); err != nil {t.Fatal(s, err)}}
for _, c := range []struct{ src, want string }{{`Verb [ "Cards-js" ]`, `[ "cards" "js" ]`}, {`Verb [ "I-sort  Rank" ]`, `[ "i" "sort" "rank" ]`}, {`Verb [ "Gloria" ]`, `[ "gloria" ]`}, {`Verb [ "Num_ber x" ]`, `[ "num" "ber" "x" ]`}, {`Verb [ "-a-" ]`, `[ "a" ]`}, {`Verb [ "Ум" ]`, `[ "ум" ]`}, {`Verb [ "" ]`, `[]`},
	{`Concord [ [ [ "glotta" ] ] [ [ "glossa" ] [ "lexika" ] ] 3 ]`, "@true"}, {`Concord [ [ [ "glotta" ] ] [ [ "glossa" ] [ "lexika" ] ] 4 ]`, "@false"}, {`Concord [ [ [ "ум" ] ] [ [ "ум" "x" ] ] 3 ]`, "@true"},
	{`Concord [ [ [ "dict" ] [ "lexika" ] ] [ [ "dictionary" ] ] 4 ]`, "@true"}, {`Concord [ [ [ "nun" ] ] [ [ "num" ] ] 3 ]`, "@false"}, {`Concord [ [ [ "cards" "js" ] ] [ [ "nexus" "js" ] ] 3 ]`, "@true"}} {
	if v, err := in.Run(c.src, env); err != nil || v.String() != c.want {t.Errorf("%s = %v %v, ждём %s", c.src, v, err, c.want)}}}
