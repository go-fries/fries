module github.com/go-fries/fries/codec/sonic/v3

go 1.23.0

replace github.com/go-fries/fries/codec/v3 => ../

require (
	github.com/bytedance/sonic v1.14.0
	github.com/go-fries/fries/codec/v3 v3.8.0
	github.com/stretchr/testify v1.10.0
)

require (
	github.com/bytedance/sonic/loader v0.3.0 // indirect
	github.com/cloudwego/base64x v0.1.6 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	golang.org/x/arch v0.19.0 // indirect
	golang.org/x/sys v0.34.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
