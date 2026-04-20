// Package layout implements a flex-like layout engine for terminal UIs.
package layout

import (
	"strings"

	"github.com/mattn/go-runewidth"
	"github.com/smasonuk/earlgray/internal/node"
	"github.com/smasonuk/earlgray/internal/style"
	"github.com/smasonuk/earlgray/internal/textflow"
)

// Constraints bounds the available space for a node.
type Constraints struct {
	MinW, MaxW int
	MinH, MaxH int
}

// Result holds the computed position and content area for a node.
type Result struct {
	// Rect is the full bounding box (including border and padding) in parent coords.
	Rect style.Rect
	// Content is the inner area available to children (after padding and border).
	Content style.Rect
}

// Tree is a computed layout tree, parallel to the node tree.
type Tree struct {
	Result   Result
	Children []*Tree
}

// Layout computes the layout for a node tree given constraints.
// The resulting Tree roots at (0,0).
func Layout(n *node.Node, c Constraints) *Tree {
	return layoutNode(n, c, 0, 0)
}

// childInfo holds per-child layout data during the flex pass.
type childInfo struct {
	nd        *node.Node
	mainSize  int
	crossSize int // intrinsic cross-axis size (0 = use parent cross size)
	flexGrow  int
	isFixed   bool // true for DimCells or measured intrinsic children
}

// clampSize constrains v to [lo, hi]. hi==0 means unbounded.
func clampSize(v, lo, hi int) int {
	if v < lo {
		v = lo
	}
	if hi > 0 && v > hi {
		v = hi
	}
	return v
}

// sizeOnAxis returns the main-axis and cross-axis sizes given direction.
func sizeOnAxis(dir style.Direction, w, h int) (main, cross int) {
	if dir == style.Row {
		return w, h
	}
	return h, w
}

// makeSize reconstructs (w, h) from (main, cross) given direction.
func makeSize(dir style.Direction, main, cross int) (w, h int) {
	if dir == style.Row {
		return main, cross
	}
	return cross, main
}

// layoutNode computes the layout for a node placed at (ox, oy).
func layoutNode(nd *node.Node, c Constraints, ox, oy int) *Tree {
	// Component/Keyed nodes: delegate to their child.
	if nd.Kind == node.ComponentKind || nd.Kind == node.KeyedKind {
		if len(nd.Children) == 1 {
			return layoutNode(nd.Children[0], c, ox, oy)
		}
		// Empty component.
		return &Tree{Result: Result{
			Rect:    style.Rect{X: ox, Y: oy, W: c.MaxW, H: c.MaxH},
			Content: style.Rect{X: ox, Y: oy, W: c.MaxW, H: c.MaxH},
		}}
	}

	if nd.Kind == node.TextPanelKind || nd.Kind == node.TextAreaKind {
		result := styledBoxResult(nd.Style, c, ox, oy)
		return &Tree{Result: result}
	}

	// Text nodes: size is determined by constraints.
	if nd.Kind == node.TextKind || nd.Kind == node.RichTextKind {
		w := c.MaxW
		h := c.MaxH
		return &Tree{Result: Result{
			Rect:    style.Rect{X: ox, Y: oy, W: w, H: h},
			Content: style.Rect{X: ox, Y: oy, W: w, H: h},
		}}
	}

	// Overlay node.
	if nd.Kind == node.OverlayKind {
		result := styledBoxResult(nd.Style, c, ox, oy)
		tree := &Tree{Result: result}
		if len(nd.Children) == 0 {
			return tree
		}
		childC := Constraints{
			MinW: result.Content.W,
			MaxW: result.Content.W,
			MinH: result.Content.H,
			MaxH: result.Content.H,
		}
		tree.Children = make([]*Tree, len(nd.Children))
		for i, child := range nd.Children {
			tree.Children[i] = layoutNode(child, childC, result.Content.X, result.Content.Y)
		}
		return tree
	}

	// View node.
	s := nd.Style
	result := styledBoxResult(s, c, ox, oy)

	tree := &Tree{Result: result}

	if len(nd.Children) == 0 {
		return tree
	}

	tree.Children = layoutChildren(nd.Children, result.Content, s)
	return tree
}

func styledBoxResult(s style.Style, c Constraints, ox, oy int) Result {
	w := resolveConstrainedDim(s.Width, c.MinW, c.MaxW)
	h := resolveConstrainedDim(s.Height, c.MinH, c.MaxH)

	// Apply style min/max overrides.
	if s.MinWidth > 0 && w < s.MinWidth {
		w = s.MinWidth
	}
	if s.MaxWidth > 0 && w > s.MaxWidth {
		w = s.MaxWidth
	}
	if s.MinHeight > 0 && h < s.MinHeight {
		h = s.MinHeight
	}
	if s.MaxHeight > 0 && h > s.MaxHeight {
		h = s.MaxHeight
	}

	// Border + padding insets.
	borderIns := s.Border.Insets()
	innerIns := addInsets(s.Padding, borderIns)

	contentW := w - innerIns.Left - innerIns.Right
	contentH := h - innerIns.Top - innerIns.Bottom
	if contentW < 0 {
		contentW = 0
	}
	if contentH < 0 {
		contentH = 0
	}

	return Result{
		Rect:    style.Rect{X: ox, Y: oy, W: w, H: h},
		Content: style.Rect{X: ox + innerIns.Left, Y: oy + innerIns.Top, W: contentW, H: contentH},
	}
}

func layoutStyle(nd *node.Node) style.Style {
	for (nd.Kind == node.ComponentKind || nd.Kind == node.KeyedKind) && len(nd.Children) == 1 {
		nd = nd.Children[0]
	}
	return nd.Style
}

// layoutChildren performs flex layout of children within the content rect.
func layoutChildren(children []*node.Node, content style.Rect, s style.Style) []*Tree {
	dir := s.Direction
	gap := s.Gap
	mainContent, crossContent := sizeOnAxis(dir, content.W, content.H)

	infos := make([]childInfo, len(children))
	totalFixed := 0
	totalGrow := 0

	for i, child := range children {
		cs := layoutStyle(child)
		info := childInfo{nd: child, flexGrow: cs.FlexGrow}

		var mainDim, crossDim style.Dimension
		if dir == style.Row {
			mainDim = cs.Width
			crossDim = cs.Height
		} else {
			mainDim = cs.Height
			crossDim = cs.Width
		}

		if mainDim.Kind == style.DimCells {
			info.mainSize = mainDim.Value
			info.isFixed = true
			totalFixed += mainDim.Value
		} else if cs.FlexGrow == 0 {
			// Auto-sized non-flex child: measure intrinsic size so it shrinks
			// to fit its content rather than collapsing to zero.
			// Convert (mainContent, crossContent) back to (w, h) before measuring.
			mw, mh := makeSize(dir, mainContent, crossContent)
			iw, ih := measureIntrinsic(child, mw, mh)
			var iMain, iCross int
			if dir == style.Row {
				iMain, iCross = iw, ih
			} else {
				iMain, iCross = ih, iw
			}
			info.mainSize = iMain
			info.isFixed = true
			info.crossSize = iCross
			totalFixed += iMain
		}

		// Cross-axis explicit override.
		if crossDim.Kind == style.DimCells {
			info.crossSize = crossDim.Value
		}

		totalGrow += cs.FlexGrow
		infos[i] = info
	}

	gapTotal := 0
	if len(children) > 1 {
		gapTotal = gap * (len(children) - 1)
	}
	remaining := mainContent - totalFixed - gapTotal
	if remaining < 0 {
		remaining = 0
	}

	// Distribute remaining space to flex-grow children.
	if totalGrow > 0 && remaining > 0 {
		distributed := 0
		for i := range infos {
			if infos[i].isFixed || infos[i].flexGrow == 0 {
				continue
			}
			share := (infos[i].flexGrow * remaining) / totalGrow
			infos[i].mainSize = share
			distributed += share
		}
		// Remainder to last flex child.
		leftover := remaining - distributed
		if leftover > 0 {
			for i := len(infos) - 1; i >= 0; i-- {
				if !infos[i].isFixed && infos[i].flexGrow > 0 {
					infos[i].mainSize += leftover
					break
				}
			}
		}
	}

	// Justify-content: extra start offset and spacing between items.
	startOffset, extraSpacing := computeJustify(s.Justify, infos, mainContent, gap, totalGrow)

	cursor := startOffset
	trees := make([]*Tree, len(children))
	for i, info := range infos {
		mainSize := info.mainSize

		// Cross axis size: prefer explicit cells, then intrinsic, then full cross.
		crossSize := crossContent
		if info.crossSize > 0 {
			crossSize = info.crossSize
		}
		crossSize = clampSize(crossSize, 0, crossContent)

		// Cross axis alignment offset.
		crossOffset := crossAxisOffset(s.AlignItems, crossContent, crossSize)

		// Exact child constraints.
		cw, ch := makeSize(dir, mainSize, crossSize)
		childC := Constraints{
			MinW: cw, MaxW: cw,
			MinH: ch, MaxH: ch,
		}

		var cx, cy int
		if dir == style.Row {
			cx = content.X + cursor
			cy = content.Y + crossOffset
		} else {
			cx = content.X + crossOffset
			cy = content.Y + cursor
		}

		trees[i] = layoutNode(info.nd, childC, cx, cy)

		cursor += mainSize
		if i < len(children)-1 {
			cursor += gap + extraSpacing
		}
	}

	return trees
}

// computeJustify returns (startOffset, extraSpacing) for justify-content.
func computeJustify(justify style.Justify, infos []childInfo, mainContent, gap, totalGrow int) (startOffset, extraSpacing int) {
	if justify == style.JustifyStart || totalGrow > 0 {
		return 0, 0
	}
	totalUsed := 0
	for _, info := range infos {
		totalUsed += info.mainSize
	}
	gapTotal := 0
	if len(infos) > 1 {
		gapTotal = gap * (len(infos) - 1)
	}
	totalUsed += gapTotal
	free := mainContent - totalUsed
	if free <= 0 {
		return 0, 0
	}
	switch justify {
	case style.JustifyCenter:
		return free / 2, 0
	case style.JustifyEnd:
		return free, 0
	case style.JustifySpaceBetween:
		if len(infos) > 1 {
			return 0, free / (len(infos) - 1)
		}
		return free / 2, 0
	}
	return 0, 0
}

// crossAxisOffset returns the offset on the cross axis for AlignItems.
func crossAxisOffset(align style.Align, crossContent, crossSize int) int {
	switch align {
	case style.AlignCenter:
		off := (crossContent - crossSize) / 2
		if off < 0 {
			return 0
		}
		return off
	case style.AlignEnd:
		off := crossContent - crossSize
		if off < 0 {
			return 0
		}
		return off
	default: // AlignStart, AlignStretch
		return 0
	}
}

// resolveConstrainedDim resolves a Dimension against min/max constraints.
func resolveConstrainedDim(d style.Dimension, minV, maxV int) int {
	switch d.Kind {
	case style.DimCells:
		v := d.Value
		if v < minV {
			v = minV
		}
		if maxV > 0 && v > maxV {
			v = maxV
		}
		return v
	default: // DimAuto — take all available space
		return maxV
	}
}

// addInsets combines two Insets.
func addInsets(a, b style.Insets) style.Insets {
	return style.Insets{
		Top:    a.Top + b.Top,
		Right:  a.Right + b.Right,
		Bottom: a.Bottom + b.Bottom,
		Left:   a.Left + b.Left,
	}
}

// measureIntrinsic returns the natural (w, h) of a node given available space.
// Used to size auto, non-flex children so they shrink to fit their content.
func measureIntrinsic(nd *node.Node, maxW, maxH int) (w, h int) {
	switch nd.Kind {
	case node.TextKind:
		return measureText(nd.Text, maxW, maxH)

	case node.RichTextKind:
		return measureRichText(nd.Spans, maxW, maxH)

	case node.TextPanelKind:
		s := nd.Style
		bIns := s.Border.Insets()
		ins := addInsets(s.Padding, bIns)

		innerMaxW := maxW - ins.Left - ins.Right
		innerMaxH := maxH - ins.Top - ins.Bottom
		if innerMaxW < 0 {
			innerMaxW = 0
		}
		if innerMaxH < 0 {
			innerMaxH = 0
		}

		cw, ch := measureTextPanelIntrinsic(nd.Text, nd.TextPanelOpts, innerMaxW, innerMaxH)

		w = cw + ins.Left + ins.Right
		h = ch + ins.Top + ins.Bottom

		if s.Width.Kind == style.DimCells {
			w = s.Width.Value
		}
		if s.Height.Kind == style.DimCells {
			h = s.Height.Value
		}

		w, h = applyMeasuredMinMaxAndClamp(s, w, h, maxW, maxH)
		return w, h

	case node.TextAreaKind:
		s := nd.Style
		bIns := s.Border.Insets()
		ins := addInsets(s.Padding, bIns)

		innerMaxW := maxW - ins.Left - ins.Right
		innerMaxH := maxH - ins.Top - ins.Bottom
		if innerMaxW < 0 {
			innerMaxW = 0
		}
		if innerMaxH < 0 {
			innerMaxH = 0
		}

		text := nd.Text
		if text == "" {
			text = nd.TextAreaOpts.Placeholder
		}

		cw, ch := measureTextPanelIntrinsic(text, node.TextPanelOptions{
			WordWrap:      nd.TextAreaOpts.WordWrap,
			ShowScrollbar: nd.TextAreaOpts.ShowScrollbar,
		}, innerMaxW, innerMaxH)

		w = cw + ins.Left + ins.Right
		h = ch + ins.Top + ins.Bottom

		if s.Width.Kind == style.DimCells {
			w = s.Width.Value
		}
		if s.Height.Kind == style.DimCells {
			h = s.Height.Value
		}

		w, h = applyMeasuredMinMaxAndClamp(s, w, h, maxW, maxH)
		return w, h

	case node.ViewKind:
		s := nd.Style
		bIns := s.Border.Insets()
		ins := addInsets(s.Padding, bIns)

		innerMaxW := maxW - ins.Left - ins.Right
		innerMaxH := maxH - ins.Top - ins.Bottom
		if innerMaxW < 0 {
			innerMaxW = 0
		}
		if innerMaxH < 0 {
			innerMaxH = 0
		}

		cw, ch := measureChildrenIntrinsic(nd.Children, innerMaxW, innerMaxH, s.Direction, s.Gap)

		measuredW := cw + ins.Left + ins.Right
		measuredH := ch + ins.Top + ins.Bottom

		w = measuredW
		h = measuredH

		if s.Width.Kind == style.DimCells {
			w = s.Width.Value
		}
		if s.Height.Kind == style.DimCells {
			h = s.Height.Value
		}

		w, h = applyMeasuredMinMaxAndClamp(s, w, h, maxW, maxH)
		return w, h

	case node.OverlayKind:
		s := nd.Style
		bIns := s.Border.Insets()
		ins := addInsets(s.Padding, bIns)

		innerMaxW := maxW - ins.Left - ins.Right
		innerMaxH := maxH - ins.Top - ins.Bottom
		if innerMaxW < 0 {
			innerMaxW = 0
		}
		if innerMaxH < 0 {
			innerMaxH = 0
		}

		cw, ch := 0, 0
		for _, child := range nd.Children {
			ccw, cch := measureIntrinsic(child, innerMaxW, innerMaxH)
			if ccw > cw {
				cw = ccw
			}
			if cch > ch {
				ch = cch
			}
		}

		w = cw + ins.Left + ins.Right
		h = ch + ins.Top + ins.Bottom

		if s.Width.Kind == style.DimCells {
			w = s.Width.Value
		}
		if s.Height.Kind == style.DimCells {
			h = s.Height.Value
		}

		w, h = applyMeasuredMinMaxAndClamp(s, w, h, maxW, maxH)
		return w, h

	case node.ComponentKind, node.KeyedKind:
		if len(nd.Children) == 1 {
			return measureIntrinsic(nd.Children[0], maxW, maxH)
		}
		return 0, 0
	}
	return 0, 0
}

func applyMeasuredMinMaxAndClamp(s style.Style, w, h, maxW, maxH int) (int, int) {
	if s.MinWidth > 0 && w < s.MinWidth {
		w = s.MinWidth
	}
	if s.MaxWidth > 0 && w > s.MaxWidth {
		w = s.MaxWidth
	}
	if s.MinHeight > 0 && h < s.MinHeight {
		h = s.MinHeight
	}
	if s.MaxHeight > 0 && h > s.MaxHeight {
		h = s.MaxHeight
	}

	if maxW > 0 && w > maxW {
		w = maxW
	}
	if maxH > 0 && h > maxH {
		h = maxH
	}

	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}

	return w, h
}

// measureText returns the display dimensions of a plain text string.
func measureText(text string, maxW, maxH int) (w, h int) {
	lines := strings.Split(text, "\n")
	maxLine := 0
	for _, l := range lines {
		n := runewidth.StringWidth(l)
		if n > maxLine {
			maxLine = n
		}
	}
	rh := len(lines)
	if maxW > 0 && maxLine > maxW {
		maxLine = maxW
	}
	if maxH > 0 && rh > maxH {
		rh = maxH
	}
	return maxLine, rh
}

func measureRichText(spans []node.TextSpan, maxW, maxH int) (w, h int) {
	lines := node.SplitTextSpansLines(spans)
	maxLine := 0
	for _, line := range lines {
		width := node.RichTextLineWidth(line)
		if width > maxLine {
			maxLine = width
		}
	}
	rh := len(lines)
	if maxW > 0 && maxLine > maxW {
		maxLine = maxW
	}
	if maxH > 0 && rh > maxH {
		rh = maxH
	}
	return maxLine, rh
}

// measureChildrenIntrinsic measures the aggregate intrinsic size of children.
func measureChildrenIntrinsic(children []*node.Node, maxW, maxH int, dir style.Direction, gap int) (w, h int) {
	for i, child := range children {
		cw, ch := measureIntrinsic(child, maxW, maxH)
		if dir == style.Row {
			w += cw
			if i > 0 {
				w += gap
			}
			if ch > h {
				h = ch
			}
		} else {
			h += ch
			if i > 0 {
				h += gap
			}
			if cw > w {
				w = cw
			}
		}
	}
	return w, h
}

func measureTextPanelIntrinsic(text string, opts node.TextPanelOptions, maxW, maxH int) (w, h int) {
	visual := textflow.VisualLines(text, opts.WordWrap, maxW)

	rawH := len(visual)
	w = textflow.MaxLineWidth(visual)
	h = rawH

	overflowY := opts.ShowScrollbar && maxH > 0 && rawH > maxH
	if overflowY {
		if maxW <= 0 || w < maxW {
			w++
		}
	}

	if maxW > 0 && w > maxW {
		w = maxW
	}
	if maxH > 0 && h > maxH {
		h = maxH
	}

	return w, h
}
