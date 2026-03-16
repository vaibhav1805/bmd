// Package ast defines the internal Abstract Syntax Tree node types
// used to represent parsed markdown documents.
package ast

// NodeType identifies the kind of AST node.
type NodeType int

const (
	NodeDocument NodeType = iota
	NodeParagraph
	NodeText
	NodeCode      // inline code
	NodeHeading
	NodeCodeBlock
	NodeBlockQuote
	NodeList
	NodeListItem
	NodeTable
	NodeTableRow
	NodeTableCell
	NodeHardBreak
	NodeSoftBreak
	NodeLink
	NodeImage
	NodeHorizontalRule
)

// Node is the common interface implemented by all AST nodes.
type Node interface {
	Type() NodeType
	Children() []Node
}

// baseNode provides shared fields for all nodes.
type baseNode struct {
	children []Node
}

func (b *baseNode) Children() []Node {
	return b.children
}

func (b *baseNode) AddChild(n Node) {
	b.children = append(b.children, n)
}

// Document is the root node containing all block-level nodes.
type Document struct {
	baseNode
}

func NewDocument() *Document {
	return &Document{}
}

func (d *Document) Type() NodeType { return NodeDocument }

// Paragraph represents a block of inline content.
type Paragraph struct {
	baseNode
}

func NewParagraph() *Paragraph {
	return &Paragraph{}
}

func (p *Paragraph) Type() NodeType { return NodeParagraph }

// Text is an inline text node with optional styling flags.
type Text struct {
	baseNode
	Content       string
	Bold          bool
	Italic        bool
	Strikethrough bool
}

func NewText(content string) *Text {
	return &Text{Content: content}
}

func (t *Text) Type() NodeType { return NodeText }

// Code is an inline code span.
type Code struct {
	baseNode
	Content string
}

func NewCode(content string) *Code {
	return &Code{Content: content}
}

func (c *Code) Type() NodeType { return NodeCode }

// Heading is a block-level heading with level 1-6.
type Heading struct {
	baseNode
	Level int // 1-6
}

func NewHeading(level int) *Heading {
	return &Heading{Level: level}
}

func (h *Heading) Type() NodeType { return NodeHeading }

// CodeBlock is a fenced or indented code block.
type CodeBlock struct {
	baseNode
	Language string
	Content  string
}

func NewCodeBlock(language, content string) *CodeBlock {
	return &CodeBlock{Language: language, Content: content}
}

func (c *CodeBlock) Type() NodeType { return NodeCodeBlock }

// BlockQuote wraps block-level content as a quotation.
type BlockQuote struct {
	baseNode
}

func NewBlockQuote() *BlockQuote {
	return &BlockQuote{}
}

func (b *BlockQuote) Type() NodeType { return NodeBlockQuote }

// List is an ordered or unordered list.
type List struct {
	baseNode
	Ordered bool
	Start   int // starting number for ordered lists (usually 1)
}

func NewList(ordered bool) *List {
	return &List{Ordered: ordered, Start: 1}
}

func (l *List) Type() NodeType { return NodeList }

// ListItem is a single item in a List.
type ListItem struct {
	baseNode
}

func NewListItem() *ListItem {
	return &ListItem{}
}

func (li *ListItem) Type() NodeType { return NodeListItem }

// Table is a block-level table.
type Table struct {
	baseNode
	Alignments []string // "left", "right", "center", "" per column
}

func NewTable() *Table {
	return &Table{}
}

func (t *Table) Type() NodeType { return NodeTable }

// TableRow is a row within a Table.
type TableRow struct {
	baseNode
	IsHeader bool
}

func NewTableRow(isHeader bool) *TableRow {
	return &TableRow{IsHeader: isHeader}
}

func (tr *TableRow) Type() NodeType { return NodeTableRow }

// TableCell is a cell within a TableRow.
type TableCell struct {
	baseNode
	Alignment string // "left", "right", "center", ""
}

func NewTableCell(alignment string) *TableCell {
	return &TableCell{Alignment: alignment}
}

func (tc *TableCell) Type() NodeType { return NodeTableCell }

// HardBreak represents an explicit line break within a paragraph.
type HardBreak struct {
	baseNode
}

func NewHardBreak() *HardBreak {
	return &HardBreak{}
}

func (hb *HardBreak) Type() NodeType { return NodeHardBreak }

// SoftBreak represents a newline within a paragraph (treated as space).
type SoftBreak struct {
	baseNode
}

func NewSoftBreak() *SoftBreak {
	return &SoftBreak{}
}

func (sb *SoftBreak) Type() NodeType { return NodeSoftBreak }

// Link is an inline hyperlink.
type Link struct {
	baseNode
	URL   string
	Title string
}

func NewLink(url, title string) *Link {
	return &Link{URL: url, Title: title}
}

func (l *Link) Type() NodeType { return NodeLink }

// Image is an inline image reference.
type Image struct {
	baseNode
	URL   string
	Alt   string
	Title string
}

func NewImage(url, alt, title string) *Image {
	return &Image{URL: url, Alt: alt, Title: title}
}

func (i *Image) Type() NodeType { return NodeImage }

// HorizontalRule is a thematic break (<hr>).
type HorizontalRule struct {
	baseNode
}

func NewHorizontalRule() *HorizontalRule {
	return &HorizontalRule{}
}

func (hr *HorizontalRule) Type() NodeType { return NodeHorizontalRule }
