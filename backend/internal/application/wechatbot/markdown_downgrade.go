package wechatbot

import (
	"encoding/json"
	"fmt"
	"strings"
)

// DowngradeConfig controls optional downgrade behaviors.
type DowngradeConfig struct {
	MaxTextLen int // max runes per text node before truncation (0 = no limit)
}

// DefaultDowngradeConfig is the recommended production config.
var DefaultDowngradeConfig = DowngradeConfig{MaxTextLen: 0}

// Downgrade converts dsh-ui fence blocks in raw markdown to WeChat-readable plain text/markdown.
// It is a pure text transform: markdown sections pass through, dsh-ui blocks are converted to
// readable representations per component type. Illegal JSON never panics — it degrades to a
// placeholder message.
func Downgrade(raw string, cfg ...DowngradeConfig) string {
	config := DefaultDowngradeConfig
	if len(cfg) > 0 {
		config = cfg[0]
	}
	segs := scanDshuiSegments(raw)
	var out strings.Builder
	for _, seg := range segs {
		if seg.kind == kindMarkdown {
			out.WriteString(seg.content)
		} else {
			out.WriteString(downgradeDshui(seg.content, config))
		}
	}
	return out.String()
}

type segKind int

const (
	kindMarkdown segKind = iota
	kindDshui
)

type segment struct {
	kind    segKind
	content string
	open    bool // true = dsh-ui fence not yet closed
}

// scanDshuiSegments mirrors the JS scanner: char-by-char, JSON-string-aware, no regex.
// Identifies ```dsh-ui and ~~~dsh-ui fences and extracts their body content.
func scanDshuiSegments(raw string) []segment {
	var segs []segment
	var mode struct {
		kind   segKind
		buf    strings.Builder
		open   bool
		info   string
		inStr  bool
		inEsc  bool
		lineHasContent bool
	}
	mode.kind = kindMarkdown

	flush := func() {
		if mode.buf.Len() == 0 {
			return
		}
		segs = append(segs, segment{kind: mode.kind, content: mode.buf.String(), open: mode.open})
		mode.buf.Reset()
		mode.open = false
		mode.inStr = false
		mode.inEsc = false
		mode.lineHasContent = false
	}

	i := 0
	n := len(raw)
	for i < n {
		ch := raw[i]

		if mode.kind == kindMarkdown {
			if ch == '\n' {
				mode.lineHasContent = false
				mode.buf.WriteByte(ch)
				i++
				continue
			}
			if !mode.lineHasContent && (ch == '`' || ch == '~') {
				fc := ch
				j := i
				for j < n && raw[j] == fc {
					j++
				}
				if j-i >= 3 {
					k := j
					for k < n && (raw[k] == ' ' || raw[k] == '\t') {
						k++
					}
					langEnd := k
					for k < n && raw[k] != '\n' && raw[k] != ' ' && raw[k] != '\t' {
						k++
					}
					lang := raw[langEnd:k]
					if lang == "dsh-ui" {
						flush()
						mode.kind = kindDshui
						mode.open = true
						mode.lineHasContent = true
						eol := strings.Index(raw[k:], "\n")
						if eol == -1 {
							mode.info = raw[k:]
							i = n
						} else {
							mode.info = raw[k : k+eol]
							i = k + eol + 1
						}
						continue
					}
				}
			}
			mode.lineHasContent = true
			mode.buf.WriteByte(ch)
			i++
			continue
		}

		// kindDshui body
		if mode.inEsc {
			mode.inEsc = false
			mode.buf.WriteByte(ch)
			i++
			continue
		}
		if ch == '\\' {
			mode.inEsc = true
			mode.buf.WriteByte(ch)
			i++
			continue
		}
		if ch == '"' {
			mode.inStr = !mode.inStr
			mode.buf.WriteByte(ch)
			i++
			continue
		}
		if ch == '\n' {
			mode.lineHasContent = false
			mode.buf.WriteByte(ch)
			i++
			continue
		}
		if !mode.inStr && !mode.lineHasContent && (ch == '`' || ch == '~') {
			fc := ch
			j := i
			for j < n && raw[j] == fc {
				j++
			}
			if j-i >= 3 {
				k := j
				for k < n && (raw[k] == ' ' || raw[k] == '\t') {
					k++
				}
				if k == n || raw[k] == '\n' {
					flush()
					mode.kind = kindMarkdown
					mode.lineHasContent = false
					eol := strings.Index(raw[k:], "\n")
					if eol == -1 {
						i = n
					} else {
						i = k + eol + 1
					}
					continue
				}
			}
		}
		mode.lineHasContent = true
		mode.buf.WriteByte(ch)
		i++
	}

	if mode.buf.Len() > 0 {
		if mode.kind == kindDshui {
			mode.open = true
		}
		flush()
	}

	return segs
}

// downgradeDshui converts a dsh-ui fence body to WeChat-readable text.
// Malformed JSON returns a safe placeholder — never panics.
func downgradeDshui(body string, cfg DowngradeConfig) string {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		return ""
	}

	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(trimmed), &raw); err != nil {
		repaired := repairUnescapedQuotes(trimmed)
		if repaired == "" {
			return "[界面预览]\n"
		}
		if err2 := json.Unmarshal([]byte(repaired), &raw); err2 != nil {
			return "[界面预览]\n"
		}
	}

	spec := genuiSpec{}
	if v, ok := raw["type"].(string); ok {
		spec.Type = v
	}
	if v, ok := raw["title"].(string); ok {
		spec.Title = v
	}
	if v, ok := raw["items"]; ok {
		spec.Items = mustMarshal(v)
	}
	if v, ok := raw["panel"].(bool); ok {
		spec.Panel = v
	}
	if v, ok := raw["append"].(bool); ok {
		spec.Append = v
	}
	if v, ok := raw["closed"].(bool); ok {
		spec.Closed = v
	}
	if v, ok := raw["gap"].(float64); ok {
		spec.Gap = int(v)
	}

	n := make(genuiNode)
	for k, v := range raw {
		n[k] = mustMarshal(v)
	}

	return renderSpecWithRaw(&spec, n, cfg)
}

func renderSpecWithRaw(spec *genuiSpec, raw genuiNode, cfg DowngradeConfig) string {
	var lines []string
	if spec.Title != "" {
		lines = append(lines, "【"+spec.Title+"】")
	}
	items, ok := parseItems(spec.Items)
	if ok && len(items) > 0 {
		if spec.Type == "accordion" {
			if text := renderAccordion(raw, cfg); text != "" {
				lines = append(lines, text)
			}
		} else {
			for _, item := range items {
				if text := renderNode(item, cfg); text != "" {
					lines = append(lines, text)
				}
			}
		}
	} else if spec.Type != "" {
		if text := renderNode(raw, cfg); text != "" {
			lines = append(lines, text)
		}
	}
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}

// repairUnescapedQuotes fixes the LLM's common mistake of unescaped ASCII quotes inside JSON strings.
// Returns empty string if convergence fails (unclosed string).
func repairUnescapedQuotes(raw string) string {
	var out strings.Builder
	inString := false
	for i := 0; i < len(raw); i++ {
		ch := raw[i]
		if ch == '\\' && inString {
			out.WriteByte(ch)
			if i+1 < len(raw) {
				out.WriteByte(raw[i+1])
				i++
			}
			continue
		}
		if ch != '"' {
			out.WriteByte(ch)
			continue
		}
		if !inString {
			inString = true
			out.WriteByte(ch)
			continue
		}
		j := i + 1
		for j < len(raw) && (raw[j] == ' ' || raw[j] == '\t') {
			j++
		}
		next := byte(0)
		if j < len(raw) {
			next = raw[j]
		}
		if j >= len(raw) || next == ',' || next == '}' || next == ']' || next == ':' {
			inString = false
			out.WriteByte(ch)
		} else {
			out.WriteString("\\\"")
		}
	}
	if inString {
		// 字符串未闭合（LLM 截断/漏引号）：自动补闭合引号，尽力恢复而非整体降级。
		out.WriteByte('"')
	}
	return out.String()
}

// ─────────────────────────────────────────────────────────────────────────────
// Genui type definitions (mirrors frontend dsh-ui-renderer.tsx)
// ─────────────────────────────────────────────────────────────────────────────

type genuiSpec struct {
	Type    string          `json:"type"`
	Title   string          `json:"title"`
	Items   json.RawMessage `json:"items,omitempty"`
	Panel   bool            `json:"panel,omitempty"`
	Append  bool            `json:"append,omitempty"`
	Closed  bool            `json:"closed,omitempty"`
	Gap     int             `json:"gap,omitempty"`
}

func renderSpec(spec *genuiSpec, cfg DowngradeConfig) string {
	var lines []string
	if spec.Title != "" {
		lines = append(lines, "【"+spec.Title+"】")
	}
	items, ok := parseItems(spec.Items)
	if ok && len(items) > 0 {
		if spec.Type == "accordion" {
			n := make(genuiNode)
			b, _ := json.Marshal(spec)
			json.Unmarshal(b, &n)
			if text := renderAccordion(n, cfg); text != "" {
				lines = append(lines, text)
			}
		} else {
			for _, item := range items {
				if text := renderNode(item, cfg); text != "" {
					lines = append(lines, text)
				}
			}
		}
	} else if spec.Type != "" {
		n := make(genuiNode)
		b, _ := json.Marshal(spec)
		json.Unmarshal(b, &n)
		if text := renderNode(n, cfg); text != "" {
			lines = append(lines, text)
		}
	}
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}

func isContainerType(t string) bool {
	switch t {
	case "col", "row", "grid", "card", "accordion", "tabs", "text":
		return true
	}
	return false
}

func parseItems(raw json.RawMessage) ([]genuiNode, bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, true
	}
	var arr []json.RawMessage
	if err := json.Unmarshal(raw, &arr); err != nil {
		return nil, false
	}
	nodes := make([]genuiNode, 0, len(arr))
	for _, el := range arr {
		var n genuiNode
		if json.Unmarshal(el, &n) == nil {
			nodes = append(nodes, n)
		}
	}
	return nodes, true
}

type genuiNode map[string]json.RawMessage

func (n genuiNode) getString(key string) string {
	raw, ok := n[key]
	if !ok {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) != nil {
		return ""
	}
	return s
}

func (n genuiNode) getInt(key string) int {
	raw, ok := n[key]
	if !ok {
		return 0
	}
	var i int
	json.Unmarshal(raw, &i)
	return i
}

func (n genuiNode) getBool(key string) bool {
	raw, ok := n[key]
	if !ok {
		return false
	}
	var b bool
	json.Unmarshal(raw, &b)
	return b
}

func (n genuiNode) typeStr() string {
	return n.getString("type")
}

func renderNode(node genuiNode, cfg DowngradeConfig) string {
	t := node.typeStr()
	switch t {
	case "text":
		return renderText(node)
	case "row", "col", "grid", "card":
		return renderContainer(node, cfg)
	case "badge":
		return renderBadge(node)
	case "stat":
		return renderStat(node)
	case "progress":
		return renderProgress(node)
	case "callout":
		return renderCallout(node)
	case "button":
		return renderButton(node)
	case "tabs":
		return renderTabs(node, cfg)
	case "list":
		return renderList(node)
	case "keyvalue":
		return renderKeyValue(node)
	case "steps":
		return renderSteps(node)
	case "table":
		return renderTable(node)
	case "divider":
		return "──"
	case "spacer":
		return ""
	case "avatar":
		return "[" + node.getString("name") + "]"
	case "code":
		return "```" + node.getString("lang") + "\n" + node.getString("code") + "\n```"
	case "json":
		return "```json\n" + string(node["value"]) + "\n```"
	case "diff":
		return renderDiff(node)
	case "copy":
		return "📋 " + node.getString("label") + ": " + node.getString("text")
	case "mermaid", "scene3d", "plot", "chart":
		return "[" + t + " 图表]"
	case "timeline":
		return renderTimeline(node)
	case "file-tree":
		return renderFileTree(node, cfg, 0)
	case "breadcrumb":
		raw, ok := node["items"]
		if !ok {
			return ""
		}
		var items []interface{}
		if json.Unmarshal(raw, &items) != nil {
			return ""
		}
		var parts []string
		for _, it := range items {
			switch v := it.(type) {
			case string:
				parts = append(parts, v)
			case float64:
				parts = append(parts, fmt.Sprintf("%v", v))
			}
		}
		return strings.Join(parts, " > ")
	case "quiz":
		return renderQuiz(node)
	case "link":
		return "🔗 " + node.getString("label")
	case "input", "select", "checkbox", "radio", "submit", "switch", "slider", "textarea":
		return "[表单: " + t + "]"
	case "accordion":
		return renderAccordion(node, cfg)
	default:
		if t == "" {
			return ""
		}
		return "[组件: " + t + "]"
	}
}

// ─── Component renderers ────────────────────────────────────────────────────

func renderText(node genuiNode) string {
	size := node.getString("size")
	content := node.getString("content")
	if content == "" {
		return ""
	}
	var prefix string
	switch size {
	case "h1":
		prefix = "§ "
	case "h2":
		prefix = "◆ "
	case "h3":
		prefix = "› "
	case "caption", "muted":
		prefix = "· "
	}
	return prefix + content
}

func renderContainer(node genuiNode, cfg DowngradeConfig) string {
	items, _ := parseItems(node["items"])
	var lines []string
	for _, item := range items {
		if text := renderNode(item, cfg); text != "" {
			lines = append(lines, text)
		}
	}
	return strings.Join(lines, "\n")
}

func renderBadge(node genuiNode) string {
	label := node.getString("label")
	tone := node.getString("tone")
	var icon string
	switch tone {
	case "success":
		icon = "✅"
	case "warn":
		icon = "⚠️"
	case "danger":
		icon = "❌"
	case "accent":
		icon = "🔵"
	default:
		icon = "○"
	}
	return icon + " " + label
}

func renderStat(node genuiNode) string {
	label := node.getString("label")
	value := node.getString("value")
	delta := node.getString("delta")
	if label == "" && value == "" {
		return ""
	}
	text := label + ": " + value
	if delta != "" {
		text += " (" + delta + ")"
	}
	return text
}

func renderProgress(node genuiNode) string {
	label := node.getString("label")
	value := node.getInt("value")
	valueLabel := node.getString("valueLabel")
	if label == "" && valueLabel == "" {
		return fmt.Sprintf("[%d%%]", value)
	}
	return label + ": " + valueLabel + fmt.Sprintf(" [%d%%]", value)
}

func renderCallout(node genuiNode) string {
	tone := node.getString("tone")
	title := node.getString("title")
	content := node.getString("content")
	var icon string
	switch tone {
	case "success":
		icon = "✅"
	case "warning":
		icon = "⚠️"
	case "error":
		icon = "❌"
	default:
		icon = "💡"
	}
	if title != "" {
		return icon + " " + title + "\n" + content
	}
	return icon + " " + content
}

func renderButton(node genuiNode) string {
	label := node.getString("label")
	return "[按钮: " + label + "]"
}

func renderTabs(node genuiNode, cfg DowngradeConfig) string {
	var tabs []json.RawMessage
	if json.Unmarshal(node["tabs"], &tabs) != nil {
		return ""
	}
	var lines []string
	for i, tabRaw := range tabs {
		var tab struct {
			Label string `json:"label"`
			Items json.RawMessage `json:"items,omitempty"`
		}
		if json.Unmarshal(tabRaw, &tab) != nil {
			continue
		}
		lines = append(lines, fmt.Sprintf("「%s」", tab.Label))
		if tab.Items != nil {
			items, _ := parseItems(tab.Items)
			for _, item := range items {
				if text := renderNode(item, cfg); text != "" {
					lines = append(lines, "  "+text)
				}
			}
		}
		if i < len(tabs)-1 {
			lines = append(lines, "──")
		}
	}
	return strings.Join(lines, "\n")
}

func renderList(node genuiNode) string {
	var items []interface{}
	if json.Unmarshal(node["items"], &items) != nil {
		return ""
	}
	var lines []string
	for i, item := range items {
		switch v := item.(type) {
		case string:
			lines = append(lines, fmt.Sprintf("%d. %s", i+1, v))
		case map[string]interface{}:
			title, _ := v["title"].(string)
			desc, _ := v["desc"].(string)
			lines = append(lines, fmt.Sprintf("%d. %s", i+1, title))
			if desc != "" {
				lines = append(lines, "   "+desc)
			}
		}
	}
	return strings.Join(lines, "\n")
}

func renderKeyValue(node genuiNode) string {
	var pairs []map[string]string
	if json.Unmarshal(node["pairs"], &pairs) != nil {
		return ""
	}
	var lines []string
	for _, p := range pairs {
		lines = append(lines, p["key"]+": "+p["value"])
	}
	return strings.Join(lines, "\n")
}

func renderSteps(node genuiNode) string {
	var steps []map[string]interface{}
	if json.Unmarshal(node["steps"], &steps) != nil {
		return ""
	}
	current := node.getInt("current")
	var lines []string
	for i, s := range steps {
		title, _ := s["title"].(string)
		desc, _ := s["desc"].(string)
		mark := "○"
		if current == i {
			mark = "●"
		} else if current > i {
			mark = "✓"
		}
		line := mark + " " + title
		if desc != "" {
			line += "\n  " + desc
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func renderTable(node genuiNode) string {
	var columns []string
	var rows [][]interface{}
	if json.Unmarshal(node["columns"], &columns) != nil {
		return ""
	}
	if json.Unmarshal(node["rows"], &rows) != nil {
		return ""
	}
	var lines []string
	if len(columns) > 0 {
		lines = append(lines, strings.Join(columns, " | "))
		lines = append(lines, strings.Repeat("---", len(columns)))
	}
	for _, row := range rows {
		var cells []string
		for _, cell := range row {
			cells = append(cells, fmt.Sprintf("%v", cell))
		}
		lines = append(lines, strings.Join(cells, " | "))
	}
	return strings.Join(lines, "\n")
}

func renderDiff(node genuiNode) string {
	var diffs []map[string]interface{}
	if json.Unmarshal(node["diffs"], &diffs) != nil {
		return ""
	}
	var lines []string
	for _, d := range diffs {
		path, _ := d["path"].(string)
		oldText, _ := d["oldText"].(string)
		newText, _ := d["newText"].(string)
		if oldText == "" {
			lines = append(lines, "+ "+path+": "+newText)
		} else if newText == "" {
			lines = append(lines, "- "+path+": "+oldText)
		} else {
			lines = append(lines, "~ "+path+": "+oldText+" → "+newText)
		}
	}
	return strings.Join(lines, "\n")
}

func renderTimeline(node genuiNode) string {
	var items []map[string]interface{}
	if json.Unmarshal(node["items"], &items) != nil {
		return ""
	}
	var lines []string
	for _, item := range items {
		title, _ := item["title"].(string)
		desc, _ := item["desc"].(string)
		time, _ := item["time"].(string)
		line := "• " + title
		if time != "" {
			line += " [" + time + "]"
		}
		if desc != "" {
			line += "\n  " + desc
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func renderFileTree(node genuiNode, cfg DowngradeConfig, depth int) string {
	indent := strings.Repeat("  ", depth)
	var lines []string

	if node["name"] != nil {
		name := node.getString("name")
		itemType := node.getString("type")
		mark := "📄"
		if itemType == "dir" {
			mark = "📁"
		}
		lines = append(lines, indent+mark+" "+name)
	}

	if rawItems, ok := node["items"]; ok && string(rawItems) != "null" && string(rawItems) != "[]" {
		var items []map[string]interface{}
		if json.Unmarshal(rawItems, &items) != nil {
			return strings.Join(lines, "\n")
		}
		for _, item := range items {
			childName, _ := item["name"].(string)
			childType, _ := item["type"].(string)
			childMark := "📄"
			if childType == "dir" {
				childMark = "📁"
			}
			lines = append(lines, indent+"  "+childMark+" "+childName)
			if childRaw, ok := item["children"]; ok && childRaw != nil {
				switch c := childRaw.(type) {
				case []map[string]interface{}:
					for _, ch := range c {
						sub := genuiNode{
							"name": mustMarshal(ch["name"]),
							"type": mustMarshal(ch["type"]),
						}
						if children, hasChildren := ch["children"]; hasChildren && children != nil {
							sub["items"] = mustMarshal(children)
						}
						if text := renderFileTree(sub, cfg, depth+2); text != "" {
							lines = append(lines, text)
						}
					}
				case []interface{}:
					for _, ci := range c {
						if cm, ok := ci.(map[string]interface{}); ok {
							sub := genuiNode{
								"name": mustMarshal(cm["name"]),
								"type": mustMarshal(cm["type"]),
							}
							if children, hasChildren := cm["children"]; hasChildren && children != nil {
								sub["items"] = mustMarshal(children)
							}
							if text := renderFileTree(sub, cfg, depth+2); text != "" {
								lines = append(lines, text)
							}
						}
					}
				}
			}
		}
	}
	return strings.Join(lines, "\n")
}

func mustMarshal(v interface{}) json.RawMessage {
	if v == nil {
		return nil
	}
	b, _ := json.Marshal(v)
	return b
}

func renderQuiz(node genuiNode) string {
	question := node.getString("question")
	var options []map[string]interface{}
	if json.Unmarshal(node["options"], &options) != nil {
		return question
	}
	var lines []string
	lines = append(lines, question)
	for i, opt := range options {
		label, _ := opt["label"].(string)
		lines = append(lines, fmt.Sprintf("  %c. %s", 'A'+i, label))
	}
	return strings.Join(lines, "\n")
}

func renderAccordion(node genuiNode, cfg DowngradeConfig) string {
	title := node.getString("title")
	var rawItems json.RawMessage
	if json.Unmarshal(node["items"], &rawItems) != nil {
		return title
	}
	var items []interface{}
	if json.Unmarshal(rawItems, &items) != nil {
		return title
	}
	var lines []string
	if title != "" {
		lines = append(lines, "【"+title+"】")
	}
	for i, item := range items {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		secTitle, _ := m["title"].(string)
		lines = append(lines, fmt.Sprintf("▶ %s", secTitle))
		if secItemsRaw, ok := m["items"].([]interface{}); ok {
			for _, si := range secItemsRaw {
				siMap, ok := si.(map[string]interface{})
				if !ok {
					continue
				}
				siNode := genuiNode{}
				for k, v := range siMap {
					siNode[k] = mustMarshal(v)
				}
				if siType := siNode.typeStr(); siType != "" {
					if text := renderNode(siNode, cfg); text != "" {
						lines = append(lines, "  "+text)
					}
				} else if content, _ := siMap["content"].(string); content != "" {
					lines = append(lines, "  "+content)
				}
			}
		}
		if i < len(items)-1 {
			lines = append(lines, "──")
		}
	}
	return strings.Join(lines, "\n")
}
