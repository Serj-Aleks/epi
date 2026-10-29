package ason

import (
	"strings"
	"testing"
)

// Примеры со страницы спецификации ASON (с поправками раздела 2.4:
// дата и время через «/»).
var authorExamples = []string{
	`[ 1 2 3 4 5 ]`,
	`[ "A" "B" "C" ]`,
	`[ 123 4.56 $78.90 ]`,
	`[ Mr "John" "Smith" 1990-12-1 john@jsmith.org ]`,
	"[\n  \"Reno\" NV\n  \"Eureka\" CA\n  \"Boston\" MA\n  \"Miami\" FL\n]",
	`[ <p> 2020-6-18/12:43:50 </p> ]`,
	`[ if time > 10:30 [ turn-off back-light ]]`,
	`[ at [ 10:22 15:30 ] activate [ front side back ]]`,
	`[ TCP address 138.197.200.83 for http://altscript.com ]`,
	`{ name: "John Smith" age: 27 }`,
	`{ site: "Altscript" url: http://altscript.com }`,
	`{ directions: [ up down left right ] }`,
	"{\n    first-name:  \"John\"\n    last-name:   \"Smith\"\n    birth-date:  1990-7-4\n    clock-color: 255.80.30\n    wake-time:   [ 7:44 weekdays 9:00 weekends ]\n    services:    [ http://spotify.com http://pandora.com ]\n    song-id:     #\"C23A9F87D457\"\n}",
	"{\n    start-time: 10:30\n    stop-time: start-time + 3:00\n}",
	`( 123 + 456 )`,
	`( 10:30 - 3:00 )`,
	`( 2021-2-1 - 365 )`,
	`( 1.20.00 + 0.0.1 )`,
	`( length? first-name )`,
	`( "Bob" + " " + "Smith" )`,
	`((degrees - 32) / 9) * 5`,
	`[ (123 + 456) (10.7 + 8.95) ]`,
}

// Предикации из раздела 9 спецификации.
var epiExamples = []string{
	`A > B -> C`,
	"count: [\n    i < 10 -> [ print i  i: (i + 1)  count ]\n]",
	`each { in: items  as: x  do: [ print x ] }`,
	`x + 0 -> x`,
}

func TestExamplesParseAndRoundTrip(t *testing.T) {
	for _, src := range append(authorExamples, epiExamples...) {
		v, err := Parse(src)
		if err != nil {
			t.Errorf("%q: %v", src, err)
			continue
		}
		canon := Format(v)
		v2, err := Parse(canon)
		if err != nil {
			t.Errorf("канонический текст не читается: %q: %v", canon, err)
			continue
		}
		if !same(v, v2) {
			t.Errorf("обход меняет дерево: %q → %q", src, canon)
		}
		if Format(v2) != canon {
			t.Errorf("канонический текст меняется при повторном обходе: %q", canon)
		}
	}
}

func same(a, b []Value) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Kind != b[i].Kind || a[i].Text != b[i].Text || !same(a[i].Items, b[i].Items) {
			return false
		}
	}
	return true
}

// Каждый вид значения из перечня 2.2 и форма терма из 2.3.
func TestKinds(t *testing.T) {
	cases := map[string]Kind{
		`123`: Integer, `-7`: Integer, `1.23`: Decimal, `$1.23`: Money, `1.2.3`: Tuple,
		`1..20`: Range, `12x340`: Coord, `#1A2B`: Hex,
		`2021-4-5`: Date, `10:23:45`: Time, `2021-4-5/10:23:45`: Date,
		`2021-4-5/10:23:45-8:00`: Date, `2021-4-5/10:23:45+2:00`: Date,
		`"abc"`: String, `''abc''`: LongString, `<abc>`: Tag, `%abc`: File,
		`abc://`: URL, `ab@cde`: Email, `#"a1b2c"`: Binary16, `##"atbfc"`: Binary64,
		`abc`: Term, `abc:`: SetTerm, `:abc`: GetTerm, `'abc`: LitTerm,
		`.abc`: Selector, `/abc`: Refinement, `@true`: Literal,
		`abc.def`: Field, `abc.def:`: SetField, `:abc.def`: GetField, `abc/def`: Path,
		`+`: Op, `/`: Op, `<`: Op, `>=`: Op, `<>`: Op, `->`: Op, `-`: Op,
		`l-n`: Term, `length?`: Term,
	}
	for src, want := range cases {
		v, err := Parse(src)
		if err != nil {
			t.Errorf("%q: %v", src, err)
			continue
		}
		if len(v) != 1 || v[0].Kind != want {
			t.Errorf("%q: получено %v, ожидалось %v", src, v, want)
		}
	}
}

func TestStrings(t *testing.T) {
	v, err := Parse(`"a \"b\" \\ c\nd"`)
	if err != nil || v[0].Text != "a \"b\" \\ c\nd" {
		t.Fatalf("экранирование: %v %q", err, v)
	}
	v, err = Parse("''строка\nс \"кавычками\" и \\''")
	if err != nil || v[0].Text != "строка\nс \"кавычками\" и \\" {
		t.Fatalf("длинная строка: %v %q", err, v)
	}
	v, err = Parse(`[ <a href="x"> </a> ]`)
	if err != nil || v[0].Items[0].Text != `a href="x"` || v[0].Items[1].Text != "/a" {
		t.Fatalf("тег с атрибутом: %v %v", err, v)
	}
}

func TestObjectEntries(t *testing.T) {
	v, err := Parse(`{ start-time: 10:30 stop-time: start-time + 3:00 }`)
	if err != nil {
		t.Fatal(err)
	}
	es := v[0].Entries()
	if len(es) != 2 || es[1].Key.Text != "stop-time:" || Format(es[1].Value) != "start-time + 3:00" {
		t.Fatalf("значение до следующего терма-определения: %+v", es)
	}
}

func TestTermsCaseInsensitive(t *testing.T) {
	if !SameTerm("First-Name", "first-name") || SameTerm("a", "b") {
		t.Fatal("термы регистронезависимы")
	}
}

// Ошибочный ввод отвергается целиком, с местом ошибки.
func TestErrors(t *testing.T) {
	cases := map[string]string{
		`[ 1 2`:             "строка 1, позиция 1: блок не закрыт",
		`[ 1 ) ]`:           "строка 1, позиция 5: лишняя закрывающая скобка",
		"[ 1\n  12abc ]":    "строка 2, позиция 3: непонятный токен",
		`1,23`:              "непонятный токен",
		`2021-4-5-10:23:45`: "непонятный токен",
		`{ 1 a: 2 }`:        "объектный блок должен начинаться",
		`{ a: b: 2 }`:       "у «a:» нет значения",
		`{ a: }`:            "у «a:» нет значения",
		`"abc`:              "строка не закрыта",
		"\"a\nb\"":          "перевод строки внутри",
		`"a\q"`:             "неизвестное экранирование",
		`"a"b`:              "после значения нужен пробел",
		`''abc`:             "длинная строка не закрыта",
		`<abc`:              "тег не закрыт",
		`#"xyz"`:            "недопустимые символы",
		`ab"c"`:             "кавычка внутри токена",
	}
	for src, want := range cases {
		v, err := Parse(src)
		if err == nil {
			t.Errorf("%q: ошибка не найдена, получено %v", src, v)
			continue
		}
		if v != nil {
			t.Errorf("%q: при ошибке есть частичный результат", src)
		}
		if !strings.Contains(err.Error(), want) {
			t.Errorf("%q: %q, ожидалось %q", src, err, want)
		}
	}
}
