package gantt

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRender_ChineseText(t *testing.T) {
	mermaid := `
gantt
    title 【PC】收购功能第1期-极速收购
    dateFormat YYYY-MM-DD
    excludes weekends
    section 研发
    【触屏端】极速收购-其他页面功能以及交互 :T8何俊文, 2025-11-24, 2d
    【触屏端】极速收购-落地页以及交互 :T8何俊文, 2025-11-20, 2d
    【PC】极速收购-其他页面功能以及交互 :T8李政恩, 2025-11-25, 2d
    【PC】极速收购-落地页以及交互 :T8李政恩, 2025-11-21, 4d
    【PC】极速收购-聊一聊部分 :T8李政恩, 2025-11-19, 2d
    【触屏端】极速收购-聊一聊部分 :T8何俊文, 2025-11-19, 2d
    【后端】聊一聊商品列表接口 :T8翁晓彤, 2025-11-25, 1d
    【后端】聊一聊洽谈权限 :T8翁晓彤, 2025-11-24, 1d
    【后端】回收商列表接口 :T8翁晓彤, 2025-11-21, 1d
    【后端】商品待出售列表&商品详情改造用于展示收购banner :T8翁晓彤, 2025-11-20, 1d
    【后端】极速收购开关配置 :T8翁晓彤, 2025-11-19, 1d
    section 测试
    【测试】整体部分验证 :T8曾令涛, 2025-11-28, 1d
    【测试】聊一聊部分&后台 :T8曾令涛, 2025-11-27, 1d
    【测试】引導入口落地頁部分 :T8曾令涛, 2025-11-26, 1d
    【测试】后端接口测试、sql注入测试 :T8曾令涛, 2025-11-25, 1d
`
	out := filepath.Join(os.TempDir(), "gantt.png")
	in := Input{
		Source:     mermaid,
		OutputPath: out,
		Scale:      1,
		// FontPath 可留空使用默认字体，确保不 panic
	}
	_, err := Render(t.Context(), in)
	if err != nil {
		t.Fatalf("render chinese failed: %v", err)
	}
	//_ = os.Remove(out)
	println(out)
}
