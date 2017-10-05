package templatefunctions

import (
	"testing"

	"go.aoe.com/flamingo/framework/template"

	"github.com/stretchr/testify/assert"
)

func TestJsJSON(t *testing.T) {
	var jsJSON template.Function = new(JsJSON)

	assert.Equal(t, jsJSON.Name(), "JSON")

	json := jsJSON.Func().(func() JSON)()
	assert.Equal(t, `{"foo":123}`, json.Stringify(map[string]int{"foo": 123}))
}