package template

/*

WARNING!!!

This is a work in progress!

Please do not judge this file! Please :)

*/

import (
	"bytes"
	"encoding/json"
	"flamingo/core"
	"flamingo/core/template/pug-ast"
	"flamingo/core/web"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
	"sync"
	"time"
)

var (
	assetrewrites map[string]string
	templates     map[string]*template.Template
	templatesLock sync.Mutex
)

func init() {
	loadTemplates()
}

func loadTemplates() {
	start := time.Now()

	var err error

	templatesLock.Lock()
	defer templatesLock.Unlock()

	manifest, _ := ioutil.ReadFile("frontend/dist/manifest.json")
	json.Unmarshal(manifest, &assetrewrites)

	pugast := node.PugAst{
		Path: "frontend/dist/templates",
	}
	templates, err = compile(&pugast, "frontend/dist/templates", "")

	if err != nil {
		panic(err)
	}

	log.Println("Compiled templates in", time.Since(start))
}

func compile(pugast *node.PugAst, root, dirname string) (map[string]*template.Template, error) {
	result := make(map[string]*template.Template)

	dir, _ := os.Open(path.Join(root, dirname))

	filenames, _ := dir.Readdir(-1)

	for _, filename := range filenames {
		if filename.IsDir() {
			tpls, _ := compile(pugast, root, path.Join(dirname, filename.Name()))
			for k, v := range tpls {
				if result[k] == nil {
					result[k] = v
				}
			}
		} else {
			if strings.HasSuffix(filename.Name(), ".ast.json") {
				name := path.Join(dirname, filename.Name())
				name = name[:len(name)-len(".ast.json")]
				result[name] = pugast.TokenToTemplate(name, pugast.Parse(name))
			}
		}
	}

	return result, nil
}

// Render via hmtl/template
func Render(app *core.App, ctx web.Context, tpl string, data interface{}) io.Reader {
	buf := new(bytes.Buffer)

	// recompile
	if app.Debug {
		loadTemplates()
	}

	t, _ := templates[tpl].Clone()

	t.Funcs(template.FuncMap{
		"asset": func(a string) template.URL {
			url := app.Url("_static")
			aa := strings.Split(a, "/")
			aaa := aa[len(aa)-1]
			var result string
			if assetrewrites[aaa] != "" {
				result = url.String() + "/" + assetrewrites[aaa]
			} else {
				result = url.String() + a
			}
			ctx.Push(result, nil)
			return template.URL(result)
		},
		"__": fmt.Sprintf, // todo translate
		"get": func(what string) interface{} {
			if what == "user.name" {
				return "testuser"
			}
			return []map[string]string{{"url": "url1", "name": "item1"}, {"url": "url2", "name": "name2"}}
		},
	})

	err := t.ExecuteTemplate(buf, tpl, map[string]interface{}{
		"isProductionBuild": true,
		"classBody":         "default",
		"title":             "Home",
		"site": map[string]interface{}{
			"title": "Auckland Airport",
		},
	})
	if err != nil {
		panic(err)
	}

	return buf
}
