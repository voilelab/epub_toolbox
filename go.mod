module github.com/voilelab/epub_toolbox

go 1.27.2

require (
	github.com/saintfish/chardet v0.0.0-20230101081208-5e3ef4b5456d
	github.com/voilelab/toolgui v0.14.0
	golang.org/x/text v0.42.0
)

require golang.org/x/net v0.60.0 // indirect

tool github.com/voilelab/toolgui/cmd/toolgui-wasm
