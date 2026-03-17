// Substitute replaces {{name}} placeholders in the template with values from args (REQ-04.009).
// Only placeholders that have a key in args are replaced. If a placeholder has no arg, returns error.
package toolcatalog

import (
	"fmt"
	"regexp"
)

// placeholder matches {{ident}} where ident is one or more word chars.
var placeholderRE = regexp.MustCompile(`\{\{(\w+)\}\}`)

// Substitute returns the template with each {{name}} replaced by args[name].
// Values are formatted as strings. If any {{name}} has no corresponding key in args, returns an error.
func Substitute(template string, args map[string]any) (string, error) {
	return substituteWithFormat(template, args, argToString)
}

func substituteWithFormat(template string, args map[string]any, format func(any) string) (string, error) {
	missing := placeholderRE.FindAllStringSubmatch(template, -1)
	for _, m := range missing {
		if len(m) != 2 {
			continue
		}
		name := m[1]
		if _, ok := args[name]; !ok {
			return "", fmt.Errorf("template: missing argument for placeholder %q", name)
		}
	}
	result := placeholderRE.ReplaceAllStringFunc(template, func(ph string) string {
		m := placeholderRE.FindStringSubmatch(ph)
		if len(m) != 2 {
			return ph
		}
		name := m[1]
		v := args[name]
		return format(v)
	})
	return result, nil
}

// SubstituteMust is like Substitute but panics on error (for tests or when args are known complete).
func SubstituteMust(template string, args map[string]any) string {
	out, err := Substitute(template, args)
	if err != nil {
		panic(err)
	}
	return out
}
