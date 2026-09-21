package widgets

func (m State) NodeGrid() (width, columns, rows int) {
	width = max(20, m.Width-4)
	if m.Width == 0 {
		width = 88
	}
	// Compact cards fit two feature flags per line.
	columns = max(1, (width+1)/33)
	header := 10 + len(m.UpdateBannerLines(width))
	rows = max(1, (m.Height-header)/6)
	return
}

func (m State) CardsPerPage() int {
	_, columns, rows := m.NodeGrid()
	return columns * rows
}
