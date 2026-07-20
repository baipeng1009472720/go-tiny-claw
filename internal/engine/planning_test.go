package engine

import "testing"

func TestNormalizePlanDeduplicatesAndLimitsItems(t *testing.T) {
	raw := "1. 读取 README。\n读取 README。\n2. 检查入口。\n3. 总结结构。\n4. 给出用法。\n5. 补充注意事项。\n6. 不应出现。"
	want := "1. 读取 README\n2. 检查入口\n3. 总结结构\n4. 给出用法\n5. 补充注意事项"

	if got := normalizePlan(raw); got != want {
		t.Fatalf("normalizePlan() = %q, want %q", got, want)
	}
}

func TestNormalizePlanFallsBackForEmptyContent(t *testing.T) {
	want := "1. 分析任务并按需调用可用工具"

	if got := normalizePlan(" \n\t"); got != want {
		t.Fatalf("normalizePlan() = %q, want %q", got, want)
	}
}
