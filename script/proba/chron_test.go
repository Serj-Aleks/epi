// epi/script/proba/chron_test.go
// тесты времени (disp/chron.go): nunc при подменённых часах и имя нового сеанса фреймом Novus — как newSeans (prtcl.go)
package proba
import ("testing"; "time"; "epi/script/disp")
// TestNunc: nunc 0 и nunc -1 при часах 26.11.2026 17:31:05 Берлин; Novus — «seans-» + дата + «.uic», как time.Now().Format("2006-01-02") в newSeans
func TestNunc(t *testing.T) {
zone, _ := time.LoadLocation("Europe/Berlin")
at := time.Date(2026, 11, 26, 17, 31, 5, 0, zone)
disp.Nunc = func() time.Time {return at}
defer func() {disp.Nunc = time.Now}()
novus := "Novus.In : [ N ] . Novus.Ex : Novus . Novus : [ Novus.Novus : ( ( \"seans-\" cnctn ( ( nitxt ( nunc Novus.N ) ) front 10 ) ) cnctn \".uic\" ) ] . "
for src, want := range map[string]string{
	"nunc 0": "2026-11-26/17:31:05+1:00",
	"nunc -1": "2026-11-25/17:31:05+1:00",
	novus + "Novus [ 0 ]": `"seans-` + at.Format("2006-01-02") + `.uic"`,
	novus + "Novus [ 36 ]": `"seans-` + at.AddDate(0, 0, 36).Format("2006-01-02") + `.uic"`} {
	got, _, _, err := runText(t, src, 1000)
	if err != nil || got != want {t.Errorf("%q: %q %v, ждём %q", src, got, err, want)}} }
