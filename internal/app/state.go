package app

func fieldIndexFor(reqIndex, field int) int {
	if field < globalFieldCount {
		return field
	}
	return globalFieldCount + reqIndex*requestFieldCount + (field - globalFieldCount)
}

func decodeFieldIndex(index int) (reqIndex int, field int) {
	if index < globalFieldCount {
		return 0, index
	}
	zeroBased := index - globalFieldCount
	return zeroBased / requestFieldCount, (zeroBased % requestFieldCount) + globalFieldCount
}

func (m *model) clampSelectedResult() {
	if len(m.Results) == 0 {
		m.SelectedResult = 0
		return
	}
	if m.SelectedResult < 0 {
		m.SelectedResult = 0
	}
	if m.SelectedResult >= len(m.Results) {
		m.SelectedResult = len(m.Results) - 1
	}
}

func (m *model) syncActiveRequestToSelectedResult() {
	if len(m.Results) == 0 || m.SelectedResult < 0 || m.SelectedResult >= len(m.Results) {
		return
	}
	sourceIdx := m.Results[m.SelectedResult].SourceIdx
	if sourceIdx >= 0 && sourceIdx < len(m.Forms) {
		m.ActiveReq = sourceIdx
	}
}
