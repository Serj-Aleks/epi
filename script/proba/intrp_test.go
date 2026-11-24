// epi/script/proba/intrp_test.go
// тесты встроенных интерпретаторов (disp/intrp.go): arifm, logic, trgnm, lnrfm
package proba
import ("strings"; "testing"; "epi/script/disp")
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
	in := disp.New(100, func(string) bool { return true })
	v, err := in.Run(c.src, disp.NewEnv(nil))
	if c.want == "" {if len(in.Unknown) == 0 {t.Errorf("%s: ожидалось «не распознано», получено %v", c.src, v)}; continue}
	if err != nil || v.String() != c.want || strings.Join(in.Used, " ") != c.frames {t.Errorf("%s = %v %v (%v), ожидалось %s (%s)", c.src, v, err, in.Used, c.want, c.frames)}}
for _, src := range []string{`ln 0`, `ln -1`, `asin 2`, `1 & 2`, `2 sin 3`} {
	in := disp.New(100, func(string) bool { return true })
	in.Run(src, disp.NewEnv(nil))
	if len(in.Unknown) != 1 {t.Errorf("%s: ожидалось «не распознано»", src)}} }
