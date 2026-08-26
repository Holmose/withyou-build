package wechatbot

import (
	"strings"
	"testing"
)

func TestDowngrade_MarkdownPassthrough(t *testing.T) {
	input := "Hello world\n\nThis is plain text."
	got := Downgrade(input)
	if got != input {
		t.Fatalf("markdown should pass through unchanged, got %q", got)
	}
}

func TestDowngrade_DshuiFenceBasic(t *testing.T) {
	input := "Before\n\n```dsh-ui\n{\"type\":\"text\",\"content\":\"Hello from dsh-ui\"}\n```\n\nAfter"
	got := Downgrade(input)
	if !strings.Contains(got, "Hello from dsh-ui") {
		t.Fatalf("dsh-ui text should appear in output, got %q", got)
	}
	if !strings.Contains(got, "Before") || !strings.Contains(got, "After") {
		t.Fatalf("markdown sections should be preserved, got %q", got)
	}
}

func TestDowngrade_TildeFence(t *testing.T) {
	input := "```dsh-ui\n{\"type\":\"badge\",\"label\":\"Active\",\"tone\":\"success\"}\n```"
	got := Downgrade(input)
	if !strings.Contains(got, "✅") || !strings.Contains(got, "Active") {
		t.Fatalf("badge should downgrade to icon+label, got %q", got)
	}
}

func TestDowngrade_IllegalJSON(t *testing.T) {
	input := "```dsh-ui\n{invalid json here}\n```"
	got := Downgrade(input)
	if got == "" {
		t.Fatalf("illegal JSON should degrade to placeholder, got empty string")
	}
}

func TestDowngrade_EmptyBody(t *testing.T) {
	input := "```dsh-ui\n\n```"
	got := Downgrade(input)
	if got != "" {
		t.Fatalf("empty dsh-ui should return empty string, got %q", got)
	}
}

func TestDowngrade_AllComponentTypes(t *testing.T) {
	cases := []struct {
		name     string
		json     string
		expected string
	}{
		{
			"text h1",
			`{"type":"text","size":"h1","content":"Title"}`,
			"§ Title",
		},
		{
			"text h2",
			`{"type":"text","size":"h2","content":"Subtitle"}`,
			"◆ Subtitle",
		},
		{
			"text h3",
			`{"type":"text","size":"h3","content":"Heading"}`,
			"› Heading",
		},
		{
			"text body",
			`{"type":"text","content":"Body text"}`,
			"Body text",
		},
		{
			"text muted",
			`{"type":"text","size":"muted","content":"caption"}`,
			"· caption",
		},
		{
			"badge success",
			`{"type":"badge","label":"Done","tone":"success"}`,
			"✅ Done",
		},
		{
			"badge warn",
			`{"type":"badge","label":"Warning","tone":"warn"}`,
			"⚠️ Warning",
		},
		{
			"badge danger",
			`{"type":"badge","label":"Error","tone":"danger"}`,
			"❌ Error",
		},
		{
			"badge accent",
			`{"type":"badge","label":"Info","tone":"accent"}`,
			"🔵 Info",
		},
		{
			"badge default",
			`{"type":"badge","label":"Neutral"}`,
			"○ Neutral",
		},
		{
			"stat",
			`{"type":"stat","label":"Score","value":"95","delta":"+3"}`,
			"Score: 95 (+3)",
		},
		{
			"stat no delta",
			`{"type":"stat","label":"Users","value":"1,234"}`,
			"Users: 1,234",
		},
		{
			"progress",
			`{"type":"progress","label":"上传","value":75,"valueLabel":"进行中"}`,
			"上传: 进行中 [75%]",
		},
		{
			"progress no label",
			`{"type":"progress","value":50}`,
			"[50%]",
		},
		{
			"callout info",
			`{"type":"callout","tone":"info","content":"Note taken"}`,
			"💡 Note taken",
		},
		{
			"callout success",
			`{"type":"callout","tone":"success","title":"Saved","content":"Data stored"}`,
			"✅ Saved\nData stored",
		},
		{
			"callout warning",
			`{"type":"callout","tone":"warning","content":"Check config"}`,
			"⚠️ Check config",
		},
		{
			"callout error",
			`{"type":"callout","tone":"error","content":"Failed"}`,
			"❌ Failed",
		},
		{
			"button",
			`{"type":"button","label":"Submit"}`,
			"[按钮: Submit]",
		},
		{
			"divider",
			`{"type":"divider"}`,
			"──",
		},
		{
			"spacer",
			`{"type":"spacer"}`,
			"",
		},
		{
			"avatar",
			`{"type":"avatar","name":"Alice"}`,
			"[Alice]",
		},
		{
			"code",
			`{"type":"code","lang":"go","code":"func main() {}"}`,
			"```go\nfunc main() {}\n```",
		},
		{
			"link",
			`{"type":"link","label":"Docs"}`,
			"🔗 Docs",
		},
		{
			"copy",
			`{"type":"copy","label":"Token","text":"abc123"}`,
			"📋 Token: abc123",
		},
		{
			"mermaid",
			`{"type":"mermaid","code":"graph TD"}`,
			"[mermaid 图表]",
		},
		{
			"scene3d",
			`{"type":"scene3d","title":"3D","meshes":[]}`,
			"[scene3d 图表]",
		},
		{
			"chart",
			`{"type":"chart","kind":"bars","title":"Stats","data":[]}`,
			"[chart 图表]",
		},
		{
			"plot",
			`{"type":"plot","series":[]}`,
			"[plot 图表]",
		},
		{
			"breadcrumb",
			`{"type":"breadcrumb","items":["Home","Settings"]}`,
			"Home > Settings",
		},
		{
			"accordion title only",
			`{"type":"accordion","title":"FAQ","items":[]}`,
			"【FAQ】",
		},
		{
			"accordion with sections",
			`{"type":"accordion","title":"Guide","items":[{"title":"Step 1","items":[{"type":"text","content":"Do this"}]}]}`,
			"【Guide】\n▶ Step 1\n  Do this",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			json := tc.json
			tmpl := "text before\n\n```dsh-ui\n%s\n```\n\ntext after"
			input := strings.NewReplacer("%", "%%").Replace(tmpl)
			input = "```dsh-ui\n" + json + "\n```"
			got := Downgrade(input)
			if !strings.Contains(got, tc.expected) {
				t.Errorf("[%s]\n  input: %s\n  expected substring: %q\n  got: %q", tc.name, json, tc.expected, got)
			}
		})
	}
}

func TestDowngrade_ContainerTypes(t *testing.T) {
	t.Run("col with items", func(t *testing.T) {
		input := "```dsh-ui\n{\"type\":\"col\",\"items\":[{\"type\":\"text\",\"content\":\"Line 1\"},{\"type\":\"text\",\"content\":\"Line 2\"}]}\n```"
		got := Downgrade(input)
		if !strings.Contains(got, "Line 1") || !strings.Contains(got, "Line 2") {
			t.Errorf("col should render nested items, got %q", got)
		}
	})

	t.Run("row with items", func(t *testing.T) {
		input := "```dsh-ui\n{\"type\":\"row\",\"items\":[{\"type\":\"text\",\"content\":\"A\"},{\"type\":\"text\",\"content\":\"B\"}]}\n```"
		got := Downgrade(input)
		if !strings.Contains(got, "A") || !strings.Contains(got, "B") {
			t.Errorf("row should render nested items, got %q", got)
		}
	})

	t.Run("card with title and items", func(t *testing.T) {
		input := "```dsh-ui\n{\"type\":\"card\",\"title\":\"Card Title\",\"items\":[{\"type\":\"text\",\"content\":\"Body\"}]}\n```"
		got := Downgrade(input)
		if !strings.Contains(got, "Card Title") || !strings.Contains(got, "Body") {
			t.Errorf("card should render title+items, got %q", got)
		}
	})
}

func TestDowngrade_List(t *testing.T) {
	input := `{"type":"col","items":[{"type":"list","items":["Item A","Item B",{"title":"Item C","desc":"description"}]}]}`
	got := Downgrade("```dsh-ui\n" + input + "\n```")
	if !strings.Contains(got, "1. Item A") {
		t.Errorf("list item 1 missing, got %q", got)
	}
	if !strings.Contains(got, "2. Item B") {
		t.Errorf("list item 2 missing, got %q", got)
	}
	if !strings.Contains(got, "Item C") || !strings.Contains(got, "description") {
		t.Errorf("list object item should show title+desc, got %q", got)
	}
}

func TestDowngrade_Table(t *testing.T) {
	input := `{"type":"col","items":[{"type":"table","columns":["Name","Age"],"rows":[["Alice",28],["Bob",34]]}]}`
	got := Downgrade("```dsh-ui\n" + input + "\n```")
	if !strings.Contains(got, "Name") || !strings.Contains(got, "Age") {
		t.Errorf("table columns missing, got %q", got)
	}
	if !strings.Contains(got, "Alice") || !strings.Contains(got, "Bob") {
		t.Errorf("table rows missing, got %q", got)
	}
}

func TestDowngrade_KeyValue(t *testing.T) {
	input := `{"type":"col","items":[{"type":"keyvalue","pairs":[{"key":"Name","value":"Alice"},{"key":"Age","value":"28"}]}]}`
	got := Downgrade("```dsh-ui\n" + input + "\n```")
	if !strings.Contains(got, "Name: Alice") || !strings.Contains(got, "Age: 28") {
		t.Errorf("keyvalue should render pairs, got %q", got)
	}
}

func TestDowngrade_Steps(t *testing.T) {
	input := `{"type":"col","items":[{"type":"steps","current":1,"steps":[{"title":"Start"},{"title":"Process"},{"title":"End"}]}]}`
	got := Downgrade("```dsh-ui\n" + input + "\n```")
	if !strings.Contains(got, "●") || !strings.Contains(got, "✓") || !strings.Contains(got, "○") {
		t.Errorf("steps should have markers, got %q", got)
	}
	if !strings.Contains(got, "Start") || !strings.Contains(got, "Process") || !strings.Contains(got, "End") {
		t.Errorf("steps titles missing, got %q", got)
	}
}

func TestDowngrade_Tabs(t *testing.T) {
	input := `{"type":"col","items":[{"type":"tabs","tabs":[{"label":"Tab A","items":[{"type":"text","content":"Content A"}]},{"label":"Tab B","items":[{"type":"text","content":"Content B"}]}]}]}`
	got := Downgrade("```dsh-ui\n" + input + "\n```")
	if !strings.Contains(got, "「Tab A」") || !strings.Contains(got, "「Tab B」") {
		t.Errorf("tabs labels missing, got %q", got)
	}
	if !strings.Contains(got, "Content A") || !strings.Contains(got, "Content B") {
		t.Errorf("tabs content missing, got %q", got)
	}
}

func TestDowngrade_Timeline(t *testing.T) {
	input := `{"type":"col","items":[{"type":"timeline","items":[{"title":"Event 1","desc":"Description 1","time":"10:00"},{"title":"Event 2"}]}]}`
	got := Downgrade("```dsh-ui\n" + input + "\n```")
	if !strings.Contains(got, "Event 1") || !strings.Contains(got, "Description 1") || !strings.Contains(got, "10:00") {
		t.Errorf("timeline content missing, got %q", got)
	}
}

func TestDowngrade_Diff(t *testing.T) {
	input := `{"type":"col","items":[{"type":"diff","diffs":[{"path":"a.go","oldText":"foo","newText":"bar"},{"path":"b.go","newText":"new line"}]}]}`
	got := Downgrade("```dsh-ui\n" + input + "\n```")
	if !strings.Contains(got, "~ a.go") || !strings.Contains(got, "+ b.go") {
		t.Errorf("diff format missing, got %q", got)
	}
}

func TestDowngrade_FormTypes(t *testing.T) {
	types := []string{"input", "select", "checkbox", "radio", "submit", "switch", "slider", "textarea"}
	for _, typ := range types {
		input := `{"type":"col","items":[{"type":"` + typ + `"}]}`
		got := Downgrade("```dsh-ui\n" + input + "\n```")
		if !strings.Contains(got, typ) {
			t.Errorf("form type %s should render as placeholder, got %q", typ, got)
		}
	}
}

func TestDowngrade_FileTree(t *testing.T) {
	input := `{"type":"col","items":[{"type":"file-tree","items":[{"name":"src","type":"dir","children":[{"name":"index.js","type":"file"}]}]}]}`
	got := Downgrade("```dsh-ui\n" + input + "\n```")
	if !strings.Contains(got, "📁") || !strings.Contains(got, "📄") {
		t.Errorf("file-tree should show icons, got %q", got)
	}
	if !strings.Contains(got, "src") || !strings.Contains(got, "index.js") {
		t.Errorf("file-tree names missing, got %q", got)
	}
}

func TestDowngrade_Quiz(t *testing.T) {
	input := `{"type":"col","items":[{"type":"quiz","question":"What is 2+2?","options":[{"label":"3"},{"label":"4","correct":true},{"label":"5"}]}]}`
	got := Downgrade("```dsh-ui\n" + input + "\n```")
	if !strings.Contains(got, "What is 2+2") {
		t.Errorf("quiz question missing, got %q", got)
	}
	if !strings.Contains(got, "A.") || !strings.Contains(got, "B.") || !strings.Contains(got, "C.") {
		t.Errorf("quiz options missing, got %q", got)
	}
}

func TestDowngrade_GenuiSpecTitle(t *testing.T) {
	input := `{"title":"Panel Title","type":"col","items":[{"type":"text","content":"Body"}]}`
	got := Downgrade("```dsh-ui\n" + input + "\n```")
	if !strings.Contains(got, "【Panel Title】") || !strings.Contains(got, "Body") {
		t.Errorf("spec title should wrap items, got %q", got)
	}
}

func TestDowngrade_UnknownType(t *testing.T) {
	input := `{"type":"foobar","content":"test"}`
	got := Downgrade("```dsh-ui\n" + input + "\n```")
	if !strings.Contains(got, "[组件: foobar]") {
		t.Errorf("unknown type should render as placeholder, got %q", got)
	}
}

func TestDowngrade_EmptyType(t *testing.T) {
	input := `{"type":""}`
	got := Downgrade("```dsh-ui\n" + input + "\n```")
	if got != "" {
		t.Errorf("empty type should return empty, got %q", got)
	}
}

func TestDowngrade_NestedContainers(t *testing.T) {
	input := `{"type":"col","items":[{"type":"card","title":"Card","items":[{"type":"col","items":[{"type":"text","content":"Nested"}]}]}]}`
	got := Downgrade("```dsh-ui\n" + input + "\n```")
	if !strings.Contains(got, "Nested") {
		t.Errorf("deeply nested content should render, got %q", got)
	}
}

func TestDowngrade_UnclosedDshui(t *testing.T) {
	input := "```dsh-ui\n{\"type\":\"text\",\"content\":\"Incomplete"
	got := Downgrade(input)
	if got == "" {
		t.Errorf("unclosed dsh-ui should not return empty, got %q", got)
	}
}

func TestDowngrade_MultipleDshuiBlocks(t *testing.T) {
	input := "First\n\n```dsh-ui\n{\"type\":\"col\",\"items\":[{\"type\":\"text\",\"content\":\"A\"}]}\n```\n\n```dsh-ui\n{\"type\":\"col\",\"items\":[{\"type\":\"badge\",\"label\":\"B\"}]}\n```\n\nLast"
	got := Downgrade(input)
	if !strings.Contains(got, "First") || !strings.Contains(got, "Last") {
		t.Errorf("markdown wrappers should be preserved, got %q", got)
	}
	if !strings.Contains(got, "A") || !strings.Contains(got, "B") {
		t.Errorf("both dsh-ui blocks should render, got %q", got)
	}
}

func TestRepairUnescapedQuotes(t *testing.T) {
	cases := []struct {
		input    string
		expected string
		ok       bool
	}{
		// 合法 JSON 原样通过（引号后紧跟结构字符 → 结构性闭合）
		{`{"a":"b"}`, `{"a":"b"}`, true},
		{`{"a":"b", "c":"d"}`, `{"a":"b", "c":"d"}`, true},
		// 引号后跟非结构字符（LLM 未转义的值内引号）→ 转义，末尾自动补闭合
		{`{"a":"b"c}`, `{"a":"b\"c}"`, true},
		// 截断的字符串 → 自动补闭合引号恢复
		{`{"a":"unclosed`, `{"a":"unclosed"`, true},
	}
	for _, tc := range cases {
		got := repairUnescapedQuotes(tc.input)
		if tc.ok && got != tc.expected {
			t.Errorf("repairUnescapedQuotes(%q) = %q, want %q", tc.input, got, tc.expected)
		}
		if !tc.ok && got != "" {
			t.Errorf("repairUnescapedQuotes(%q) should return empty, got %q", tc.input, got)
		}
	}
}
