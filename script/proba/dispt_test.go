// epi/script/proba/dispt_test.go
// тесты диспетчера (disp/dispt.go): модель, связывание, нераспознанное, лимит шагов, отметка интерпретаторов
package proba
import ("errors"; "strings"; "testing"; "epi/script/disp")
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
// Связывание без вычисления: живая формула и алиас
func TestBinding(t *testing.T) {
expect(t, map[string]string{
	`F.A : 1 . F.B : F.A + 10 . F.A : 5 . F.B`:   "15",
	`F.A : 1 . F.B : (F.A + 10) . F.A : 5 . F.B`: "11",}) }
// Нераспознанная предикация — не ошибка: вопрос и запись в базу
func TestUnknown(t *testing.T) {
cases := map[string]string{
	`7 // 2.5`:                  "7 // 2.5",   // // и % — только для целых
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
var st disp.Stopped
if !errors.As(err, &st) || len(asked) != 2 || st.Steps != 100 {t.Fatalf("лимит: %v, вопросов %d", err, len(asked))}
got, _, asked, err := runText(t, "S.I : 0\nC : [ S.I < 30 -> [ S.I : (S.I + 1) . C ] ]\nC\nS.I", 50, true, true, true, true, true, true)
if err != nil || got != "30" || len(asked) == 0 {t.Fatalf("продолжение после лимита: %q %v %d", got, err, len(asked))} }
// Лимит шагов — на одну интерпретацию: после остановленной следующая начинается заново
func TestLimitPerRun(t *testing.T) {
in := disp.New(50, func(string) bool { return false })
env := disp.NewEnv(nil)
in.Run(`Loop : [ @true -> [ Loop ] ]`, env)
if _, err := in.Run(`Loop`, env); err == nil {t.Fatal("цикл должен остановиться")}
v, err := in.Run(`2 + 3`, env)
if err != nil || v.String() != "5" {t.Fatalf("после остановки: %v %v", v, err)} }
// Фреймы-интерпретаторы: арифметика и логика — разные фреймы
func TestFramesUsed(t *testing.T) {
in := disp.New(100, nil)
for src, want := range map[string]string{
	`2 + 3`:         "arifm",
	`2 > 3`:         "logic",
	`( 2 + 3 ) > 4`: "arifm logic",
	`'a`:            "",} {
	if _, err := in.Run(src, disp.NewEnv(nil)); err != nil {t.Fatal(err)}
	if got := strings.Join(in.Used, " "); got != want {t.Errorf("%q: %q, ожидалось %q", src, got, want)}} }
