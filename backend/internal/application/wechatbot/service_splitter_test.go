package wechatbot

import (
	"strings"
	"testing"
)

// collectFeed 把文本以固定小 chunk(size=17,常切断 fence/表格)喂给 splitter,
// 返回全部发出的消息(含尾部 done)。
func collectFeed(sp *mdSplitter, text string) []string {
	var out []string
	r := []rune(text)
	for i := 0; i < len(r); i += 17 {
		end := i + 17
		if end > len(r) {
			end = len(r)
		}
		sp.feed(string(r[i:end]), func(c string) { out = append(out, c) })
	}
	sp.done(func(c string) { out = append(out, c) })
	return out
}

func TestShortTextNoSplit(t *testing.T) {
	sp := newMDSplitter(200)
	got := collectFeed(sp, "你好世界，这是一段简短消息。")
	if len(got) != 1 || got[0] != "你好世界，这是一段简短消息。" {
		t.Fatalf("short text should stay one message, got %q", got)
	}
}

func TestTableWholeWhenSmall(t *testing.T) {
	sp := newMDSplitter(500)
	table := "| 名称 | 数量 |\n|---|---|\n| A | 1 |\n| B | 2 |\n"
	got := collectFeed(sp, "前文\n\n"+table+"\n后文")
	if len(got) != 1 {
		t.Fatalf("small content should be one message, got %d", len(got))
	}
	if !strings.Contains(got[0], "| A | 1 |") {
		t.Fatalf("table must survive whole: %q", got[0])
	}
}

func TestTableSplitRestoresHeader(t *testing.T) {
	sp := newMDSplitter(60)
	var rows []string
	// 50 行表格整体超过 tableLimit(约 2200 字符),强制块内拆分以校验补表头。
	for i := 0; i < 50; i++ {
		rows = append(rows, "| "+strings.Repeat("v", 40)+" |\n")
	}
	table := "| 名称 | 值 |\n|---|:---:|\n" + strings.Join(rows, "")
	got := collectFeed(sp, table)
	if len(got) < 2 {
		t.Fatalf("oversized table should split into ≥2 messages, got %d", len(got))
	}
	for i, msg := range got {
		if !strings.HasPrefix(msg, "| 名称 | 值 |\n|---|:---:|\n") {
			t.Fatalf("segment %d must reattach header, got:\n%q", i, msg)
		}
	}
}

func TestCodeFenceBlankLineKeptInBlock(t *testing.T) {
	sp := newMDSplitter(200)
	code := "```go\nfunc main() {\n\n\tprintln(1)\n}\n```\n"
	got := collectFeed(sp, code)
	if len(got) != 1 {
		t.Fatalf("small code block should not split by inner blank line: %d msgs", len(got))
	}
	if !strings.HasPrefix(got[0], "```") || !strings.Contains(got[0], "\n\n") {
		t.Fatalf("code fence must stay whole with inner blank line: %q", got[0])
	}
}

func TestCodeBlockLong(t *testing.T) {
	sp := newMDSplitter(60)
	var lines []string
	for i := 0; i < 8; i++ {
		lines = append(lines, "log "+strings.Repeat("x", 12)+"\n")
	}
	code := "```go\n" + strings.Join(lines, "") + "```\n"
	collectFeed(sp, code) // 仅确保不崩、可切分
}

func TestMathBlockSurvives(t *testing.T) {
	sp := newMDSplitter(100)
	eq := "$$\na^2 + b^2 = c^2\n$$\n"
	got := collectFeed(sp, eq)
	if len(got) != 1 || !strings.Contains(got[0], "$$") {
		t.Fatalf("small math block should be one message with $$, got %q", got)
	}
}
