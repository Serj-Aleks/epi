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

func expect(t *testing.T, cases map[string]string) {
	t.Helper()
	for src, want := range cases {
		got, _, _, err := run(t, src, 100)
		if err != nil || got != want {
			t.Errorf("%q: %q %v, ожидалось %q", src, got, err, want)
		}
	}
}

func TestArithmetic(t *testing.T) {
	expect(t, map[string]string{
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
		`X : 5  X * X`:  "25",
	})
}

// Модель: истинный левый слот — правый слот и конец блока; ложный — дальше (9.1).
func TestModel(t *testing.T) {
	expect(t, map[string]string{
		`10 > 3 -> 'ok`:                     "ok",
		`1 > 3 -> 'a  2 > 3 -> 'b  'c`:      "c",
		`1 > 3 -> 'a  5 > 3 -> 'b  'c`:      "b",
		`X : 2  X > 1 -> [ Y : 10  Y + X ]`: "12",
		`X : 2  X > 1 -> Y : 7  Y`:          "@none",
	})
	_, _, _, err := run(t, `5 -> 'a`, 100)
	if err == nil || !strings.Contains(err.Error(), "не флаг") {
		t.Errorf("левый слот не флаг: %v", err)
	}
}

// Цикл — фрейм, применяющий сам себя (9.2); « : » связывает, ( ) вычисляет сейчас (9.3).
func TestLoop(t *testing.T) {
	src := "I : 0\nSum : 0\nCount : [\n    I < 10 -> [ I : (I + 1)  Sum : (Sum + I)  Count ]\n]\nCount\nSum"
	got, _, _, err := run(t, src, 1000)
	if err != nil || got != "55" {
		t.Fatalf("цикл: %q %v", got, err)
	}
}

// Связывание без вычисления: живая формула и алиас.
func TestBinding(t *testing.T) {
	expect(t, map[string]string{
		`A : 1  B : A + 10  A : 5  B`:   "15",
		`A : 1  B : (A + 10)  A : 5  B`: "11",
	})
}

// Фрейм-интерпретатор со строчной буквы — препозиция в начале предикации;
// фрейм-объект с прописной — аппликация по имени.
func TestFrames(t *testing.T) {
	expect(t, map[string]string{
		"double : [ X * 2 ]\nN : 4\ndouble { X : N + 1 }": "10",
		`Twice : [ X * 2 ]  Twice { X : 21 }`:             "42",
	})
}

// Нераспознанная предикация — не ошибка: вопрос и запись в базу (0.7).
func TestUnknown(t *testing.T) {
	cases := map[string]string{
		`7 // 2`:                     "7 // 2",     // нет интерпретатора символа
		`7 foo 2`:                    "7 foo 2",    // нет интерпретатора препозиции-терма
		`"a" + 1`:                    `"a" + 1`,    // нет интерпретации для таких аргументов
		`Frobnicate`:                 "Frobnicate", // имя не определено
		`1 +`:                        "1 +",        // препозиция в конце
		`1 + 2 + 3`:                  "1 + 2 + 3",  // две препозиции в слоте
		`1 / 0`:                      "1 / 0",
		`print`:                      "print",      // препозиция без операнда
		`x : 5`:                      "x : 5",      // препозиция определяется алиасом или фреймом
		`double : [ X ]  1 double 2`: "1 double 2", // фрейм-препозиция — только в начале
	}
	for src, rec := range cases {
		_, in, asked, err := run(t, src, 100, true)
		if err != nil || len(asked) != 1 || len(in.Unknown) != 1 || in.Unknown[0] != rec {
			t.Errorf("%q: записано %q, вопросы %q, ошибка %v", src, in.Unknown, asked, err)
		}
	}
	got, in, _, err := run(t, `7 // 2  3 + 4`, 100, false)
	if err != nil || len(in.Unknown) != 0 || got != "7" {
		t.Errorf("отказ от записи, интерпретация продолжается: %q %v %v", got, in.Unknown, err)
	}
}

// Лимит шагов (0.16): по исчерпании — вопрос, продолжать ли.
func TestLimit(t *testing.T) {
	_, _, asked, err := run(t, `Loop : [ @true -> [ Loop ] ]  Loop`, 50, true, false)
	var st Stopped
	if !errors.As(err, &st) || len(asked) != 2 || st.Steps != 100 {
		t.Fatalf("лимит: %v, вопросов %d", err, len(asked))
	}
	got, _, asked, err := run(t, "I : 0\nC : [ I < 30 -> [ I : (I + 1)  C ] ]\nC\nI", 50, true, true, true, true, true, true)
	if err != nil || got != "30" || len(asked) == 0 {
		t.Fatalf("продолжение после лимита: %q %v %d", got, err, len(asked))
	}
}

// Регистр значим по первой букве (9.9); буквы любого алфавита (0.17).
func TestNames(t *testing.T) {
	expect(t, map[string]string{
		"Число : 5\nудвоить : [ X * 2 ]\nудвоить { X : Число }": "10",
		`Число : 5  ЧИСЛО + 1`:                                  "6",
	})
	_, in, _, _ := run(t, `Число : 5  число + 1`, 100, true)
	if len(in.Unknown) != 1 {
		t.Errorf("«число» со строчной — препозиция, не объект «Число»: %v", in.Unknown)
	}
}

// Алиасы препозиций (9.8): «имя : препозиция».
func TestAliases(t *testing.T) {
	expect(t, map[string]string{
		`plus : +  2 plus 3`:                "5",
		`minus : -  minus 4`:                "-4",
		`plus : +  summa : plus  2 summa 3`: "5",
		`major : >  10 major 3 -> 'ok`:      "ok",
		`X : (- 5)  X`:                      "-5",
		`plus : +  plus : -  7 plus 2`:      "5",
	})
	_, in, _, err := run(t, `pow : //  2 pow 3`, 100, true)
	if err != nil || len(in.Unknown) != 1 || in.Unknown[0] != "2 pow 3" {
		t.Errorf("алиас без интерпретатора: %v %v", in.Unknown, err)
	}
}

func TestLexicon(t *testing.T) {
	in := New(100, nil)
	env, err := in.Global()
	if err != nil {
		t.Fatal(err)
	}
	v, err := in.Run(`( 2 plus 3 ) minus 1`, env)
	if err != nil || v.String() != "4" {
		t.Fatalf("лексикон: %v %v", v, err)
	}
}

// Препозиция в начале строки начинает новую предикацию; на той же строке —
// продолжает текущую (9.9).
func TestLineRule(t *testing.T) {
	expect(t, map[string]string{
		`X : 5  - 4  X`:              "1", // «X : 5 - 4», затем X
		"X : 5\n- 4\nX":              "5", // «X : 5», «- 4», X
		"X : 5\n- 4":                 "-4",
		"A : 2 plus\n3":              "@none",
		"plus : +\nA : 2 plus\n3\nA": "5", // препозиция в конце строки — операнд на следующей
	})
}
