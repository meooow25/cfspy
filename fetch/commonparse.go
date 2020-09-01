package fetch

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/andybalholm/cascadia"
)

var (
	handleSelec = cascadia.MustCompile("a.rated-user")

	// From https://sta.codeforces.com/s/50332/css/community.css
	colorClsMap = map[string]int{
		"user-black":     0x000000,
		"user-legendary": 0x000000,
		"user-red":       0xFF0000,
		"user-fire":      0xFF0000,
		"user-yellow":    0xBBBB00,
		"user-violet":    0xAA00AA,
		"user-orange":    0xFF8C00,
		"user-blue":      0x0000FF,
		"user-cyan":      0x03A89E,
		"user-green":     0x008000,
		"user-gray":      0x808080,
		"user-admin":     0x000000,
	}
)

func parseHandleAndColor(selec *goquery.Selection) (handle string, color int) {
	handleA := selec.FindMatcher(handleSelec).First()
	handle = handleA.Text()
	for _, cls := range strings.Fields(handleA.AttrOr("class", "?!")) {
		if col, ok := colorClsMap[cls]; ok {
			color = col
			break
		}
	}
	return
}
