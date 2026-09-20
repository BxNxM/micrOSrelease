package tui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/micros/microsctl/internal/tui/views"
	"github.com/micros/microsctl/internal/tui/widgets"
)

func (m model) renderState() widgets.State {
	var titles []string
	for _, item := range actions {
		titles = append(titles, item.title)
	}
	return widgets.State{
		Width:            m.width,
		Height:           m.height,
		Cursor:           m.cursor,
		Inventory:        m.inventory,
		DeviceIndex:      m.deviceIndex,
		ImageIndex:       m.imageIndex,
		FirmwareIndex:    m.firmwareIndex,
		FirmwareSelected: m.firmwareSelected,
		BoardType:        m.boardType,
		Nodes:            m.nodes,
		NodeIndex:        m.nodeIndex,
		DetailAction:     m.detailAction,
		LoadingInventory: m.loadingInventory,
		ProbingUSB:       m.probingUSB,
		UsbScanRequested: m.usbScanRequested,
		UsbScanned:       m.usbScanned,
		Discovering:      m.discovering,
		LastUpdated:      m.lastUpdated,
		Confirming:       m.confirming,
		Running:          m.running,
		Operation:        string(m.operation),
		OperationError:   m.operationError,
		Progress:         m.progress,
		Stages:           m.stages,
		SpinnerFrame:     m.spinnerFrame,
		Status:           m.status,
		Result:           m.result,
		Target:           m.selectedTarget(), Actions: titles,
	}
}

func (m model) View() tea.View {
	state := m.renderState()
	if m.showFirmware {
		return views.FirmwareView(state)
	}
	if m.showNodes {
		if m.showNodeDetails && m.nodeIndex > 0 && m.nodeIndex <= len(m.nodes) {
			return views.NodeDetailsView(state)
		}
		return views.NodesView(state)
	}
	return views.ActionsView(state)
}

func (m model) nodeGrid() (int, int, int) { return m.renderState().NodeGrid() }
func (m model) cardsPerPage() int         { return m.renderState().CardsPerPage() }
