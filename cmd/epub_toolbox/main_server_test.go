//go:build !(js && wasm)

package main

import (
	"encoding/json"
	"io/fs"
	"testing"

	"github.com/voilelab/toolgui/toolgui/tgexec"
)

// TestWebConfig checks the manifest parses and its icons exist.
func TestWebConfig(t *testing.T) {
	if err := setWebConfig(tgexec.NewWebExecutor(newApp())); err != nil {
		t.Fatal(err)
	}

	bs, err := web.ReadFile("web/manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	var m tgexec.Manifest
	if err := json.Unmarshal(bs, &m); err != nil {
		t.Fatal(err)
	}
	if len(m.Icons) == 0 {
		t.Fatal("no icons")
	}
	for _, icon := range m.Icons {
		if _, err := fs.Stat(web, "web/"+icon.Src); err != nil {
			t.Errorf("icon %s: %v", icon.Src, err)
		}
	}
}
