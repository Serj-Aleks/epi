// epi/script/proba/zycle_test.go
// тесты циклов (disp/zycle.go): цикл первого уровня, recur
package proba
import ("strings"; "testing"; "epi/script/disp")
// Цикл — фрейм, применяющий сам себя; « : » связывает, ( ) вычисляет сейчас
func TestLoop(t *testing.T) {
src := "C.I : 0\nC.Sum : 0\nCount : [\n    C.I < 10 -> [ C.I : (C.I + 1) . C.Sum : (C.Sum + C.I) . Count ]\n]\nCount\nC.Sum"
got, _, _, err := runText(t, src, 1000)
if err != nil || got != "55" {t.Fatalf("цикл: %q %v", got, err)} }
// Цикл второго уровня: recur (zycle)
func TestRecur(t *testing.T) {
in := disp.New(100, func(string) bool { return true })
env := disp.NewEnv(nil)
v, err := in.Run("S.Sum : 0 . S.L : [ 1 2 3 4 ] . recur { In : S.L As : S.I Corpus : [ S.Sum : ( S.Sum + S.I ) ] } . S.Sum", env)
if err != nil || v.String() != "10" {t.Fatalf("S.Sum = %v %v", v, err)}
v, _ = in.Run("recur { In : [ 2 3 ] As : S.I Corpus : [ S.I * S.I ] }", env)
if v.String() != "9" || strings.Join(in.Used, " ") != "arifm zycle" {t.Fatalf("последний шаг = %v (%v)", v, in.Used)}
v, _ = in.Run("S.N : 4 . recur { In : S.N As : S.I Corpus : [ S.I * 2 ] }", env)
if v.String() != "8" {t.Fatalf("In : S.N = %v", v)}
in.Run("recur { In : 2.5 As : S.I Corpus : [ S.I ] }", env)
if len(in.Unknown) != 1 {t.Fatalf("In не целое — не распознано: %v", in.Unknown)} }
