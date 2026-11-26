package http_handlers

import (
	"encoding/json"
	"html/template"
	"strings"
)

// TemplateFuncMap is the FuncMap the HTML templates are rendered with. It is
// shared by the router and the handler tests so the tests exercise the exact
// production escaping.
//
// json encodes v for embedding inside an inline <script> block. json.Marshal
// already escapes <, > and &; "</" and "<!--" are additionally neutralised so
// a value can never terminate the script element or open an HTML comment.
func TemplateFuncMap() template.FuncMap {
	return template.FuncMap{
		"json": func(v interface{}) template.JS {
			a, _ := json.Marshal(v)
			s := string(a)
			s = strings.ReplaceAll(s, "</", `<\/`)
			s = strings.ReplaceAll(s, "<!--", `<\!--`)
			return template.JS(s)
		},
	}
}
