//go:build js && wasm

package main

import (
	"syscall/js"

	"github.com/voilelab/epub_toolbox/internal/i18n"
	"github.com/voilelab/toolgui/toolgui/tgwasm"
)

func main() {
	// ?lang= wins over the browser's language.
	lang := tgwasm.Query().Get("lang")
	if lang == "" {
		lang = js.Global().Get("navigator").Get("language").String()
	}
	i18n.Set(lang)
	langSwitch = true
	// Dispatched as toolgui:<event> on this tab; web/head.html relays to Umami.
	track = func(event string, data map[string]any) {
		_ = tgwasm.Emit(event, data)
	}
	app := newApp()
	// Static hosts can't route paths, so pages live in the hash.
	app.SetHashPageNameMode(true)
	tgwasm.NewExecutor(app).Run()
}
