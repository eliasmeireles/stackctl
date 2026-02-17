package netbird

import (
	"github.com/charmbracelet/bubbles/list"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/ui"
)

var (
	items = []list.Item{
		ui.CreateItem("Connect (up)", "Start NetBird VPN", ui.HoopAction),
		ui.CreateItem("Status", "Check NetBird status", ui.HoopAction),
		ui.CreateItem("Install", "Download NetBird binary", ui.HoopAction),
	}

	Menu = ui.CreateSubMenu("NetBird", "Manage VPN connection", items)
)
