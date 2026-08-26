package wechatbot

import "strings"

// blockKind 描述一行 markdown 所属的块类型,用于决定切点是否安全。
type blockKind int

const (
	bPlain blockKind = iota
	bCode            // 围栏代码块 ``` / ~~~
	bMath            // $$ 数学块
	bTable           // GFM 表格
)

// mdSplitter 把流式 markdown 拆分成消息,同时保护可渲染的块级结构(表格、围栏
// 代码块、数学块)不被切坏。
//
// 切分原则:
//   - 普通文本:按行/段落边界切,每条近似不超过 limit。
//   - 表格/围栏/数学块:整块优先一条发(可超 limit,但不超过 tableLimit),
//     避免把一张表/一个代码块拆烂。
//   - 仅当整块超过 tableLimit(极长)才在块内拆分;被拆的续表段会自动补齐
//     表头+分隔行(pending 标记),保证每一段都能独立渲染成表格。
//
// 微信 openclaw 完整渲染 markdown,故块级结构都必须保护(参考 think-strip 思路)。
type mdSplitter struct {
	limit      int
	tableLimit int // 单个表格/代码块可整条发送的最大长度
	buf        []rune
	draining   bool // 收尾阶段已开始,允许强制切分以免无限等待

	fence   rune   // 当前围栏字符('`'或'~'),0=不在代码围栏
	math    bool   // 是否在 $$ 块内
	table   bool   // 是否在表格数据行区(表头已被本条消费)
	head    string // 最近表格 "表头行\n分隔行\n"
	pending bool   // 上一条以未闭合表结尾,本条需补表头
}

func newMDSplitter(limit int) *mdSplitter {
	return &mdSplitter{limit: limit, tableLimit: 2000}
}

// feed 累积 delta,在安全切点把完整消息交给 flush。
func (m *mdSplitter) feed(delta string, flush func(string)) {
	m.buf = append(m.buf, []rune(delta)...)
	for {
		cut := m.cut()
		if cut == 0 {
			return
		}
		flush(m.msg(m.buf[:cut]))
		m.pending = m.endsInTable(m.buf[:cut])
		m.advance(m.buf[:cut])
		m.buf = m.buf[cut:]
	}
}

// done 冲刷收尾残留。
func (m *mdSplitter) done(flush func(string)) {
	if len(m.buf) == 0 {
		return
	}
	flush(m.msg(m.buf))
	m.buf = nil
	m.pending = false
}

// msg 组装一条消息;若这是续表段(上一段以未闭合表格结尾),补上表头。
func (m *mdSplitter) msg(seg []rune) string {
	if m.pending && m.head != "" && !m.hasHeader(seg) {
		return m.head + string(seg)
	}
	return string(seg)
}

// hasHeader 判断这段开头是否已经带表头+分隔行(与缓存表头一致)。
func (m *mdSplitter) hasHeader(seg []rune) bool {
	lines := splitLines(seg)
	if len(lines) < 2 || !isPipeStart(lines[0].core) {
		return false
	}
	return string(lines[0].txt)+"\n"+string(lines[1].txt) == strings.TrimRight(m.head, "\n")
}

// cut 返回切点(rune 索引);无需切时返回 0(继续累积)。
func (m *mdSplitter) cut() int {
	if len(m.buf) <= m.limit {
		return 0
	}
	lines := splitLines(m.buf)
	kinds, _, _, _, _ := annotate(lines, m.fence, m.math, m.table)
	n := len(lines)

	// 1) 普通文本切点:只认 plain 行边界,绝不在块中间切。
	lastPlain := -1
	for i := 0; i < n && lines[i].after <= m.limit; i++ {
		if kinds[i] == bPlain {
			lastPlain = lines[i].after
		}
	}
	if lastPlain > 0 {
		return lastPlain
	}

	// 2) 整个窗口都被一个块占据。
	end := 0
	for end < n && kinds[end] != bPlain {
		end++
	}
	if end == 0 {
		// 全是普通文本且单行超 long → 硬切(原样兜底)。
		return m.limit
	}
	closed := end < n // 有后续非块行
	blkLen := lines[end-1].after
	var kind blockKind
	if kinds[0] == bTable {
		kind = bTable
	}
	if !closed {
		// 块开到 buf 末尾:尚未见闭合。
		if blkLen <= m.tableLimit {
			return 0 // 等它闭合(不超上限就整体发)
		}
		return m.blockSplit(lines, kinds) // 超上限,块内拆
	}
	if blkLen <= m.tableLimit {
		return blkLen // 整块(闭合)一条发
	}
	_ = kind
	return m.blockSplit(lines, kinds)
}

// blockSplit 在块内部切一刀(仅当整块超 tableLimit 时),结果各段自行补头。
func (m *mdSplitter) blockSplit(lines []mline, kinds []blockKind) int {
	last := -1
	for i, ln := range lines {
		if ln.after > m.limit {
			break
		}
		last = i
	}
	if last < 0 {
		return m.limit
	}
	return lines[last].after
}

// endsInTable 判断 seg 末尾是否处于表格数据行内(用于下段补头)。
func (m *mdSplitter) endsInTable(seg []rune) bool {
	lines := splitLines(seg)
	_, _, _, endTable, _ := annotate(lines, m.fence, m.math, m.table)
	return endTable
}

// advance 依据已发出的 seg 推进 fence/math/table/head 状态(供剩余 buf 续用)。
func (m *mdSplitter) advance(seg []rune) {
	lines := splitLines(seg)
	_, f, math, table, head := annotate(lines, m.fence, m.math, m.table)
	m.fence, m.math, m.table = f, math, table
	if head != "" { // 表头跨段保留;续段无表头时不覆盖
		m.head = head
	}
}

// --- 行结构 ---

// mline 描述一行:after 是该行(含末尾换行)之后的索引;txt/core 为该行内容。
type mline struct {
	after int
	blank bool
	txt   []rune // 完整内容(保留空格,用于重建表头)
	core  []rune // 去除空白字符后的内容(用于块识别)
}

func splitLines(runes []rune) []mline {
	var out []mline
	var i, j int
	for i <= len(runes) {
		for j = i; j < len(runes) && runes[j] != '\n'; j++ {
		}
		after := j
		if j < len(runes) {
			after = j + 1
		}
		full := runes[i:j]
		// 输入以 \n 结尾时,哨兵空行不入列(避免误判表/段落已在末尾闭合)。
		if j == len(runes) && i == len(runes) {
			break
		}
		out = append(out, mline{after: after, txt: full, core: stripWS(full), blank: len(stripWS(full)) == 0})
		if j == len(runes) {
			break
		}
		i = j + 1
	}
	return out
}

func stripWS(r []rune) []rune {
	out := make([]rune, 0, len(r))
	for _, c := range r {
		if c != ' ' && c != '\t' {
			out = append(out, c)
		}
	}
	return out
}

// --- 块识别 ---

func isFenceOpen(core []rune) (rune, bool) {
	if len(core) < 3 {
		return 0, false
	}
	ch := core[0]
	if ch != '`' && ch != '~' {
		return 0, false
	}
	cnt := 0
	for _, r := range core {
		if r != ch {
			break
		}
		cnt++
	}
	if cnt >= 3 {
		return ch, true
	}
	return 0, false
}

func isFenceClose(core []rune, ch rune) bool {
	if ch == 0 || len(core) < 3 {
		return false
	}
	cnt := 0
	for _, r := range core {
		if r != ch {
			break
		}
		cnt++
	}
	return cnt >= 3
}

func isMathLine(core []rune) bool {
	return strings.HasPrefix(string(core), "$$")
}

func isPipeStart(core []rune) bool {
	return len(core) > 0 && core[0] == '|'
}

// isTableDelim 判断 |---|---| 分隔行。
func isTableDelim(core []rune) bool {
	s := strings.TrimSpace(string(core))
	if s == "" {
		return false
	}
	anyDash := false
	for _, cell := range strings.Split(s, "|") {
		c := strings.TrimSpace(cell)
		if c == "" {
			continue
		}
		has := false
		for _, b := range c {
			if b == '-' {
				has = true
			} else if b != ':' && b != ' ' {
				return false
			}
		}
		if !has {
			return false
		}
		anyDash = true
	}
	return anyDash
}

// annotate 逐行标注块类型,并返回末尾上下文(endFence/endMath/endTable)与扫描中发现的表头。
func annotate(lines []mline, startFence rune, startMath, startTable bool) (kinds []blockKind, endFence rune, endMath, endTable bool, head string) {
	kinds = make([]blockKind, len(lines))
	fence := startFence
	math := startMath
	table := startTable

	for i := 0; i < len(lines); i++ {
		core := lines[i].core
		if fence != 0 {
			kinds[i] = bCode
			if !lines[i].blank && isFenceClose(core, fence) {
				fence = 0
			}
			continue
		}
		if math {
			kinds[i] = bMath
			if isMathLine(core) {
				math = false
			}
			continue
		}
		if fc, ok := isFenceOpen(core); ok {
			fence = fc
			kinds[i] = bCode
			continue
		}
		if isMathLine(core) {
			math = true
			kinds[i] = bMath
			continue
		}
		if table {
			// 延续表：数据行全部带 '|';遇到非 '|' 行则表结束。
			if isPipeStart(core) {
				kinds[i] = bTable
				continue
			}
			table = false
			// 落到普通文本处理
		}
		if isPipeStart(core) {
			if i+1 < len(lines) && isPipeStart(lines[i+1].core) && isTableDelim(lines[i+1].core) {
				head = string(lines[i].txt) + "\n" + string(lines[i+1].txt) + "\n"
				table = true
				for i < len(lines) && isPipeStart(lines[i].core) {
					kinds[i] = bTable
					i++
				}
				i--
				continue
			}
			kinds[i] = bPlain
			continue
		}
		kinds[i] = bPlain
	}

	endFence = fence
	endMath = math
	endTable = table
	return kinds, endFence, endMath, endTable, head
}
