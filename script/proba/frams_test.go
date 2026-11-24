// epi/script/proba/frams_test.go
// тесты фреймов (disp/frams.go): фреймы, параметры, синонимы, вызов с ролями
package proba
import ("strings"; "testing"; "epi/script/disp"; "epi/script/format")
// Фрейм-интерпретатор со строчной буквы — препозиция в начале предикации;
// фрейм-объект с прописной — аппликация по имени
func TestFrames(t *testing.T) {
expect(t, map[string]string{
	"dupla : [ X * 2 ]\nF.N : 4\ndupla { X : F.N + 1 }": "10",
	`Twice : [ X * 2 ] . Twice { X : 21 }`:              "42",}) }
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
// Синоним концепта — параметр Alias: чтение через синоним, если своего фрейма нет; запись — в свой фрейм
func TestAliasParam(t *testing.T) {
in := disp.New(100, func(string) bool { return true })
env := disp.NewEnv(nil)
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
in := disp.New(100, func(string) bool { return true })
var body string
in.OnFrame = func(name string, b []format.Value) {body = name + " : " + format.Canon(b)}
v, err := in.Run("F.A : 1 . F : [ F.A < 3 -> [ F.A : ( F.A + 1 ) . F ] ] . F . F.A", disp.NewEnv(nil))
if err != nil || v.String() != "3" {t.Fatalf("F.A = %v %v", v, err)}
if body != "F : F.A < 3 -> [ F.A : ( F.A + 1 ) . F ]" {t.Fatalf("тело: %q", body)} }
// TestFrameRoles: роли входа In, позиционный и именованный вызов, умолчания, результат — Ex термином или блоком
func TestFrameRoles(t *testing.T) {
in := disp.New(10000, nil); env := disp.NewEnv(nil); var wrote []string
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
