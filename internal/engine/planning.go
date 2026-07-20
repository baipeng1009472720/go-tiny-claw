package engine

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	planningMaxTokens int64 = 512
	actionMaxTokens   int64 = 4096
	maxPlanItems            = 5
	maxActionTurns          = 10
)

const planningInstruction = "请先制定简洁的执行规划，不要回答最终结果，也不要假装已经调用工具。每行一个具体步骤，输出 3 到 5 步，不要重复。"

var planItemPrefix = regexp.MustCompile(`^\s*(?:[-*•]|\d+[.)、])\s*`)

func normalizePlan(raw string) string {
	items := make([]string, 0, maxPlanItems)
	seen := make(map[string]struct{}, maxPlanItems)

	for _, line := range strings.Split(raw, "\n") {
		item := planItemPrefix.ReplaceAllString(line, "")
		item = strings.TrimSpace(strings.TrimRight(item, "。.!！？ \t\r"))
		if item == "" {
			continue
		}
		if _, exists := seen[item]; exists {
			continue
		}

		seen[item] = struct{}{}
		items = append(items, item)
		if len(items) == maxPlanItems {
			break
		}
	}

	if len(items) == 0 {
		items = append(items, "分析任务并按需调用可用工具")
	}

	formatted := make([]string, len(items))
	for i, item := range items {
		formatted[i] = fmt.Sprintf("%d. %s", i+1, item)
	}
	return strings.Join(formatted, "\n")
}
