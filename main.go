package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"unicode"
)

// число с возможной дробной частью (точка или запятая)
var numRe = regexp.MustCompile(`\d+(?:[.,]\d+)?`)

var useColor = true

// paint оборачивает текст в ANSI-код цвета. Если вывод не в терминал
// (перенаправлен в файл/пайп) или задан NO_COLOR — возвращает текст как есть.
func paint(code, s string) string {
	if !useColor {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

func initColor() {
	if os.Getenv("NO_COLOR") != "" {
		useColor = false
		return
	}
	if os.Getenv("CLICOLOR_FORCE") != "" {
		useColor = true
		return
	}
	fi, err := os.Stdout.Stat()
	if err != nil || fi.Mode()&os.ModeCharDevice == 0 {
		useColor = false
	}
}

type hemisphere struct {
	letter string // N / S / E / W
	isLat  bool
}

// полушарие по букве. Принимаем и кириллицу, и латиницу, и похожие глифы
// (С/C, В/B) — их часто путают при копипасте.
func hemiOf(r rune) (hemisphere, bool) {
	switch r {
	case 'N', 'n', 'С', 'с', 'C', 'c': // север
		return hemisphere{"N", true}, true
	case 'S', 's', 'Ю', 'ю': // юг
		return hemisphere{"S", true}, true
	case 'E', 'e', 'В', 'в', 'B', 'b': // восток
		return hemisphere{"E", false}, true
	case 'W', 'w', 'З', 'з': // запад
		return hemisphere{"W", false}, true
	}
	return hemisphere{}, false
}

type coord struct {
	hemi    hemisphere
	deg     int
	minutes float64 // десятичные минуты для вывода (округлены до 3 знаков)
	dd      float64 // десятичные градусы со знаком — для ссылок на карты
}

func toFloat(s string) float64 {
	f, _ := strconv.ParseFloat(strings.Replace(s, ",", ".", 1), 64)
	return f
}

// Букву считаем обозначением полушария только если рядом нет других букв.
// Иначе ловили бы буквы из слов вроде «latitude» или «North».
func standaloneHemi(rs []rune, i int) bool {
	if _, ok := hemiOf(rs[i]); !ok {
		return false
	}
	if i > 0 && unicode.IsLetter(rs[i-1]) {
		return false
	}
	if i < len(rs)-1 && unicode.IsLetter(rs[i+1]) {
		return false
	}
	return true
}

// Вытаскивает все координаты из текста. Понимает и одну строку с широтой и
// долготой сразу (48°51'30.81"N 2°21'32.10"E), и отдельные строки —
// разделителем служит буква полушария в конце каждой координаты.
func parseAll(text string) []coord {
	rs := []rune(text)
	var out []coord
	start := 0
	for i := range rs {
		if !standaloneHemi(rs, i) {
			continue
		}
		if c, ok := buildCoord(string(rs[start:i]), rs[i]); ok {
			out = append(out, c)
		}
		start = i + 1
	}
	return out
}

func buildCoord(numbers string, hemiRune rune) (coord, bool) {
	nums := numRe.FindAllString(numbers, -1)
	if len(nums) < 2 {
		return coord{}, false
	}
	h, _ := hemiOf(hemiRune)

	deg := int(toFloat(nums[0]))
	min := toFloat(nums[1])
	sec := 0.0
	if len(nums) >= 3 {
		sec = toFloat(nums[2])
	}

	decMin := min + sec/60.0
	dd := float64(deg) + decMin/60.0
	if h.letter == "S" || h.letter == "W" {
		dd = -dd
	}

	// для показа округляем минуты до 3 знаков, при переполнении переносим в градусы
	decMin = math.Round(decMin*1000) / 1000
	if decMin >= 60 {
		deg++
		decMin -= 60
	}

	return coord{hemi: h, deg: deg, minutes: decMin, dd: dd}, true
}

func (c coord) String() string {
	return fmt.Sprintf("%s%d %.3f", c.hemi.letter, c.deg, c.minutes)
}

func googleLink(lat, lon coord) string {
	return fmt.Sprintf("https://www.google.com/maps/search/?api=1&query=%.6f,%.6f", lat.dd, lon.dd)
}

func yandexLink(lat, lon coord) string {
	// у Яндекса порядок — долгота, широта
	return fmt.Sprintf("https://yandex.ru/maps/?ll=%.6f,%.6f&z=17&pt=%.6f,%.6f,pm2rdm", lon.dd, lat.dd, lon.dd, lat.dd)
}

// copyToClipboard кладёт текст в системный буфер обмена через штатные утилиты.
func copyToClipboard(s string) bool {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "windows":
		cmd = exec.Command("clip")
	default:
		cmd = exec.Command("xclip", "-selection", "clipboard")
	}
	cmd.Stdin = strings.NewReader(s)
	return cmd.Run() == nil
}

// copyNote копирует результат в буфер и возвращает приглушённую пометку об этом.
func copyNote(s string) string {
	if copyToClipboard(s) {
		return "   " + paint("2", "(скопировано в буфер)")
	}
	return ""
}

func main() {
	setupConsole()
	initColor()

	in := bufio.NewScanner(os.Stdin)
	in.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	fmt.Println(paint("2", "Пример:  48°51'30.81\"N 2°21'32.10\"E"))
	fmt.Println()

	var lat, lon *coord
	emptyStreak := 0

	for {
		fmt.Print(paint("1;36", "> "))
		if !in.Scan() {
			break
		}
		line := in.Text()

		if strings.TrimSpace(line) == "" {
			if lat == nil && lon == nil {
				emptyStreak++
				if emptyStreak >= 2 {
					break
				}
				continue
			}
			// введена только одна координата — покажем что есть
			only := lat
			if only == nil {
				only = lon
			}
			res := only.String()
			fmt.Printf("\n  %s%s\n\n", paint("1;32", res), copyNote(res))
			lat, lon = nil, nil
			continue
		}
		emptyStreak = 0

		for _, c := range parseAll(line) {
			cc := c
			if c.hemi.isLat {
				lat = &cc
			} else {
				lon = &cc
			}
		}

		switch {
		case lat == nil && lon == nil:
			fmt.Printf("\n  %s\n\n", paint("33", "Не удалось распознать координаты. Попробуйте ещё раз."))
			continue
		case lat == nil || lon == nil:
			continue // ждём вторую координату на следующей строке
		}

		result := lat.String() + " " + lon.String()
		dd := fmt.Sprintf("%.6f, %.6f", lat.dd, lon.dd)
		fmt.Printf("\n  %s%s\n", paint("1;32", result), copyNote(result))
		fmt.Printf("  %s     %s\n", paint("2", "DD:"), dd)
		fmt.Printf("  %s %s\n", paint("2", "Google:"), paint("4;34", googleLink(*lat, *lon)))
		fmt.Printf("  %s %s\n\n", paint("2", "Яндекс:"), paint("4;34", yandexLink(*lat, *lon)))
		lat, lon = nil, nil
	}
}
