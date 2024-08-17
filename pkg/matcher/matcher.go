package matcher

import (
	"strings"

	ignore "github.com/sabhiram/go-gitignore"
)

func Match(pattern, name string) (matched bool) {
	if pattern == "" {
		pattern = "*"
	}
	lines := strings.Split(pattern, ",")
	obj := ignore.CompileIgnoreLines(lines...)
	return obj.MatchesPath(name)
}
