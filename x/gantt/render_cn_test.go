package gantt

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRender_ChineseText(t *testing.T) {
	src := `
gantt
    title 【示例】中文任务甘特图
    dateFormat YYYY-MM-DD
    excludes weekends
    section 研发
    【客户端】移动端重构 :端团队, 2025-11-24, 2d
    【客户端】落地页交互 :端团队, 2025-11-20, 2d
    【桌面】页面优化 :桌面团队, 2025-11-25, 2d
    【桌面】落地页开发 :桌面团队, 2025-11-21, 4d
    【桌面】聊天功能 :桌面团队, 2025-11-19, 2d
    【客户端】聊天功能 :端团队, 2025-11-19, 2d
    【后端】列表接口 :后端团队, 2025-11-25, 1d
    【后端】权限校验 :后端团队, 2025-11-24, 1d
    【后端】回收商列表接口 :后端团队, 2025-11-21, 1d
    【后端】商品详情改造 :后端团队, 2025-11-20, 1d
    【后端】配置开关 :后端团队, 2025-11-19, 1d
    section 测试
    【测试】整体验证 :QA团队, 2025-11-28, 1d
    【测试】聊天与后台 :QA团队, 2025-11-27, 1d
    【测试】引導入口 :QA团队, 2025-11-26, 1d
    【测试】接口与安全 :QA团队, 2025-11-25, 1d
`
	out := filepath.Join(os.TempDir(), "gantt_cn.png")
	res, err := Render(t.Context(), Input{
		Source:     src,
		OutputPath: out,
	})
	if err != nil {
		t.Fatalf("render chinese failed: %v", err)
	}
	if res.OutputPath == "" {
		t.Fatalf("expected output path")
	}
	if info, err := os.Stat(out); err != nil || info.Size() == 0 {
		t.Fatalf("output file missing or empty: %v", err)
	}
	t.Logf("rendered chinese case to %s", out)
}
