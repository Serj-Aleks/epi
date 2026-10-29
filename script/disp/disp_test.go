package disp

import (
	"errors"
	"strings"
	"testing"
)

// run: интерпретировать текст; ответы на вопросы системы — по очереди из answers.
func run(t *testing.T, text string, limit int, answers ...bool) (string, *Interp, []string, error) {
	t.Helper()
	var asked []string
	in := New(limit, func(q string) bool {
		asked = append(asked, q)
		if len(answers) == 0 {
			t.Fatalf("неожиданный вопрос: %s", q)
		}
		a := answers[0]
		answers = answers[1:]
		return a
	})
	v, err := in.Run(text, NewEnv(nil))
	return v.String(), in, asked, err
}

func TestArithmetic(t *testing.T) {
	cases := map[string]string{
		`( 2 + 3 )`:     "5",
		`2 + 3`:         "5",
		`7 - 10`:        "-3",
		`- 4`:           "-4",
		`6 * 7`:         "42",
		`7 / 2`:         "3.5",
		`1.5 + 1`:       "2.5",
		`( 2 + 3 ) * 4`: "20",
		`10 > 3`:        "@true",
		`3 >= 4`:        "@false",
		`"a" = "A"`:     "@true",
		`2 <> 2.0`:      "@false",
		`x: 5  x * x`:   "25",
	}
	for src, want := range cases {
		got, _, _, err := run(t, src, 100)
		if err != nil || got != want {
			t.Errorf("%q: %q %v, ожидалось %q", src, got, err, want)
		}
	}
}

// Модель: истинный левый слот — правый слот и конец блока; ложный — дальше (9.1).
func TestModel(t *testing.T) {
	cases := map[string]string{
		`10 > 3 -> 'ok`:                   "ok",
		`1 > 3 -> 'a  2 > 3 -> 'b  'c`:    "c",
		`1 > 3 -> 'a  5 > 3 -> 'b  'c`:    "b",
		`x: 2  x > 1 -> [ y: 10  y + x ]`: "12",
		`x: 2  x > 1 -> y: 7  y`:          "@none",
	}
	for src, want := range cases {
		got, _, _, err := run(t, src, 100)
		if err != nil || got != want {
			t.Errorf("%q: %q %v, ожидалось %q", src, got, err, want)
		}
	}
	_, _, _, err := run(t, `5 -> 'a`, 100)
	if err == nil || !strings.Contains(err.Error(), "не флаг") {
		t.Errorf("левый слот не флаг: %v", err)
	}
}

// Цикл — фрейм, применяющий сам себя (9.2); «:» связывает, ( ) вычисляет сейчас (9.3).
func TestLoop(t *testing.T) {
	src := "i: 0\nsum: 0\ncount: [\n    i < 10 -> [ i: (i + 1)  sum: (sum + i)  count ]\n]\ncount\nsum"
	got, _, _, err := run(t, src, 1000)
	if err != nil || got != "55" {
		t.Fatalf("цикл: %q %v", got, err)
	}
}

// Связывание без вычисления: живая формула и алиас.
func TestBinding(t *testing.T) {
	got, _, _, _ := run(t, `a: 1  b: a + 10  a: 5  b`, 100)
	if got != "15" {
		t.Errorf("живая формула: %q", got)
	}
	got, _, _, _ = run(t, `a: 1  b: (a + 10)  a: 5  b`, 100)
	if got != "11" {
		t.Errorf("( ) — вычисление сейчас: %q", got)
	}
}

// Аппликация фрейма с аргументами — моделями в объектном блоке.
func TestApplication(t *testing.T) {
	src := `double: [ x * 2 ]  n: 4  double { x: n + 1 }`
	got, _, _, err := run(t, src, 100)
	if err != nil || got != "10" {
		t.Fatalf("аппликация: %q %v", got, err)
	}
}

// Нераспознанная предикация — не ошибка: вопрос и запись в базу (0.7).
func TestUnknown(t *testing.T) {
	cases := map[string]string{
		`7 // 2`:     "7 // 2",     // нет интерпретатора препозиции
		`"a" + 1`:    `"a" + 1`,    // нет интерпретации для таких аргументов
		`frobnicate`: "frobnicate", // имя не определено
		`1 +`:        "1 +",        // препозиция в конце
		`1 + 2 + 3`:  "1 + 2 + 3",  // две препозиции в одной предикации
		`1 / 0`:      "1 / 0",
	}
	for src, rec := range cases {
		_, in, asked, err := run(t, src, 100, true)
		if err != nil || len(asked) != 1 || len(in.Unknown) != 1 || in.Unknown[0] != rec {
			t.Errorf("%q: записано %q, вопросы %q, ошибка %v", src, in.Unknown, asked, err)
		}
	}
	_, in, _, err := run(t, `7 // 2  3 + 4`, 100, false)
	if err != nil || len(in.Unknown) != 0 {
		t.Errorf("отказ от записи: %v %v", in.Unknown, err)
	}
	got, _, _, _ := run(t, `7 // 2  3 + 4`, 100, false)
	if got != "7" {
		t.Errorf("после нераспознанной интерпретация продолжается: %q", got)
	}
}

// Лимит шагов (0.16): по исчерпании — вопрос, продолжать ли.
func TestLimit(t *testing.T) {
	src := `loop: [ @true -> [ loop ] ]  loop`
	_, in, asked, err := run(t, src, 50, true, false)
	var st Stopped
	if !errors.As(err, &st) || len(asked) != 2 || st.Steps != 100 {
		t.Fatalf("лимит: %v, вопросов %d", err, len(asked))
	}
	_ = in
	got, _, asked, err := run(t, "i: 0\nc: [ i < 30 -> [ i: (i + 1)  c ] ]\nc\ni", 50, true, true, true, true, true, true)
	if err != nil || got != "30" || len(asked) == 0 {
		t.Fatalf("продолжение после лимита: %q %v %d", got, err, len(asked))
	}
}
