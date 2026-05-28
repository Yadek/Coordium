package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// число с возможной дробной частью (точка или запятая)
var numRe = regexp.MustCompile(`\d+(?:[.,]\d+)?`)

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
	minutes float64 // десятичные минуты
}

// разбор строки типа 55°45'21.0"С
func parseLine(line string) (coord, bool) {
	nums := numRe.FindAllString(line, -1)
	if len(nums) < 2 {
		return coord{}, false
	}

	toF := func(s string) float64 {
		f, _ := strconv.ParseFloat(strings.Replace(s, ",", ".", 1), 64)
		return f
	}

	deg := toF(nums[0])
	min := toF(nums[1])
	sec := 0.0
	if len(nums) >= 3 {
		sec = toF(nums[2])
	}

	// буква полушария идёт в конце, поэтому берём последнюю подходящую
	var hemi hemisphere
	found := false
	for _, r := range line {
		if h, ok := hemiOf(r); ok {
			hemi = h
			found = true
		}
	}
	if !found {
		return coord{}, false
	}

	decMin := min + sec/60.0
	decMin = math.Round(decMin*1000) / 1000
	d := int(deg)
	if decMin >= 60 {
		d++
		decMin -= 60
	}

	return coord{hemi: hemi, deg: d, minutes: decMin}, true
}

func (c coord) String() string {
	return fmt.Sprintf("%s%d %.3f", c.hemi.letter, c.deg, c.minutes)
}

// из произвольного текста вытаскиваем широту и долготу и склеиваем ответ
func convert(text string) (string, bool) {
	var lat, lon *coord
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		c, ok := parseLine(line)
		if !ok {
			continue
		}
		cc := c
		if c.hemi.isLat {
			lat = &cc
		} else {
			lon = &cc
		}
	}

	switch {
	case lat != nil && lon != nil:
		return lat.String() + " " + lon.String(), true
	case lat != nil:
		return lat.String(), true
	case lon != nil:
		return lon.String(), true
	}
	return "", false
}

func main() {
	setupConsole()

	in := bufio.NewScanner(os.Stdin)
	in.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	fmt.Println("=== Coordium — конвертер координат  DMS -> DMM ===")
	fmt.Println("Вставьте координаты (широта и долгота), затем пустую строку.")
	fmt.Println("Пример ввода:")
	fmt.Println("  широта 55°45'21.0\"С")
	fmt.Println("  долгота 37°37'02.0\"В")
	fmt.Println("Выход — Ctrl+C или пустой ввод дважды подряд.")
	fmt.Println()

	var block []string
	emptyStreak := 0

	for {
		fmt.Print("> ")
		if !in.Scan() {
			break
		}
		line := in.Text()

		if strings.TrimSpace(line) == "" {
			if len(block) == 0 {
				emptyStreak++
				if emptyStreak >= 2 {
					break
				}
				continue
			}
			emptyStreak = 0
			res, ok := convert(strings.Join(block, "\n"))
			block = block[:0]
			if ok {
				fmt.Print("\n  " + res + "\n\n")
			} else {
				fmt.Print("\n  Не удалось распознать координаты. Попробуйте ещё раз.\n\n")
			}
			continue
		}

		emptyStreak = 0
		block = append(block, line)
	}
}
