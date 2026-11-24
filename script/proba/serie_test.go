// epi/script/proba/serie_test.go
// тесты интерпретатора serie (disp/serie.go): строки и списки
package proba
import ("testing"; "epi/script/disp")
// TestSerie: интерпретатор serie — строки и списки; неверный операнд — не распознано
func TestSerie(t *testing.T) {
in := disp.New(10000, nil); env := disp.NewEnv(nil)
for _, c := range []struct{ src, want string }{{`dstnz "углами"`, "6"}, {`"углами" front 3`, `"угл"`}, {`"углами" post 3`, `"ами"`}, {`[ A B C ] elmnt 2`, "B"}, {`"угл" cnctn "ы"`, `"углы"`},
	{`cnvrg "Угол"`, `"угол"`}, {`"угол" front 10`, `"угол"`}, {`"угол" post 10`, `""`}, {`[ 1 2 ] elmnt 5`, "@none"}, {`[ 1 2 ] cnctn [ 3 ]`, "[ 1 2 3 ]"}, {`[ 1 2 ] cnctn 3`, "[ 1 2 3 ]"},
	{`( "Угловой" front 3 ) = ( "углами" front 3 )`, "@true"}, {`X.S : "кот" . ( X.S cnctn "ы" )`, `"коты"`}} {
	if v, err := in.Run(c.src, env); err != nil || v.String() != c.want {t.Errorf("%s = %v %v, ждём %s", c.src, v, err, c.want)}}
asked := 0; in.Ask = func(string) bool {asked++; return false}
for _, s := range []string{`"угол" front 1.5`, `dstnz 5`, `cnvrg [ A ]`, `"а" cnctn [ B ]`} {in.Run(s, env)}
if asked != 4 {t.Errorf("не распознано: %d из 4", asked)}}
