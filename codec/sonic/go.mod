module github.com/go-fries/fries/codec/sonic/v4

go 1.26.0

replace github.com/go-fries/fries/codec/v4 => ../

require (
	github.com/bytedance/sonic v1.15.2
	github.com/go-fries/fries/codec/v4 v4.0.0
	github.com/stretchr/testify v1.12.1
)

require (
	github.com/bytedance/gopkg v0.1.4 // indirect
	github.com/bytedance/sonic/loader v0.5.2 // indirect
	github.com/cloudwego/base64x v0.1.7 // indirect
	github.com/klauspost/cpuid/v2 v2.4.0 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/arch v0.30.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
)
