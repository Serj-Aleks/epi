// epi/script/proba/clstr_test.go
// тесты разбора строки (disp/clstr.go): границы предикаций, регистр имён
package proba
import ("testing")
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
// Регистр значим по первой букве; буквы любого алфавита
func TestNames(t *testing.T) {
expect(t, map[string]string{
	"Мера.Число : 5\nудвоить : [ X * 2 ]\nудвоить { X : Мера.Число }": "10",
	`Мера.Число : 5 . МЕРА.ЧИСЛО + 1`:                                 "6",})
_, in, _, _ := runText(t, `Мера.Число : 5 . число + 1`, 100, true)
if len(in.Unknown) != 1 {t.Errorf("«число» со строчной — препозиция, не объект «Число»: %v", in.Unknown)} }
