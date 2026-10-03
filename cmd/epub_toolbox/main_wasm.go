//go:build js && wasm

package main

import (
	"syscall/js"

	"github.com/voilelab/epub_toolbox/internal/i18n"
	"github.com/voilelab/toolgui/toolgui/tgwasm"
)

func main() {
	// The app runs in a worker, which sees the browser's language but not the page URL.
	i18n.Set(js.Global().Get("navigator").Get("language").String())
	app := newApp()
	// Static hosts can't route paths, so pages live in the hash.
	app.SetHashPageNameMode(true)
	tgwasm.NewExecutor(app).Run()
}
