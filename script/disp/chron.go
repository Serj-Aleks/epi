// epi/script/disp/chron.go
// chron: интерпретатор времени — nunc N: момент сейчас со сдвигом N дней (0 — сейчас, -1 — сутки назад), дата-время ASON «2026-11-26/17:31:05+1:00»
package disp
import ("fmt"; "time"; "epi/script/format")
// chronFrame: фрейм chron
func chronFrame() *Frame {
return &Frame{Name: "chron", Kernel: map[string]Prep{"nunc": func(a, b *format.Value) (format.Value, bool) {
	if a != nil || b == nil || b.Kind != format.Integer {return none, false}
	n, _, ok := num(b)
	if !ok {return none, false}
	return format.Value{Kind: format.Date, Text: dateText(Nunc().AddDate(0, 0, int(n)))}, true}}} }
// dateText: момент в записи ASON — дата/время и пояс «+1:00»
func dateText(t time.Time) string {
_, off := t.Zone()
sign := "+"
if off < 0 {sign, off = "-", -off}; return t.Format("2006-01-02/15:04:05") + fmt.Sprintf("%s%d:%02d", sign, off/3600, off%3600/60) }
