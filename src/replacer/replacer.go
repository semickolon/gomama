package replacer

import (
	"strconv"
	"strings"

	"github.com/dlclark/regexp2"
)

func Match(str string, regex *regexp2.Regexp) (*regexp2.Match, error) {
	return regex.FindStringMatch(str)
}

func Replace(str string, regex *regexp2.Regexp, subst string) (string, error) {
	var replaced string
	var err error

	matchEvalMap := func(fn func(string) (string, error)) regexp2.MatchEvaluator {
		return func(match regexp2.Match) string {
			m := match.GroupByName("m")
			str, err := fn(m.String())
			if err != nil {
				panic(err)
			}
			return match.String()[:m.Index-match.Index] + str + match.String()[m.Index-match.Index+m.Length:]
		}
	}

	switch subst {
	case "$$++":
		replaced, err = regex.ReplaceFunc(str, matchEvalMap(func(m string) (string, error) {
			n, err := strconv.Atoi(m)
			return strconv.Itoa(n + 1), err
		}), -1, -1)
	case "$$--":
		replaced, err = regex.ReplaceFunc(str, matchEvalMap(func(m string) (string, error) {
			n, err := strconv.Atoi(m)
			return strconv.Itoa(n - 1), err
		}), -1, -1)
	case "$$~U":
		replaced, err = regex.ReplaceFunc(str, matchEvalMap(func(m string) (string, error) {
			return strings.ToUpper(m), nil
		}), -1, -1)
	case "$$~L":
		replaced, err = regex.ReplaceFunc(str, matchEvalMap(func(m string) (string, error) {
			return strings.ToLower(m), nil
		}), -1, -1)
	default:
		replaced, err = regex.Replace(str, subst, -1, -1)
	}

	return replaced, err
}
