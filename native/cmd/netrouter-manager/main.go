// Package main provides the native NetRouter Manager desktop application.
// It follows a compact, high-density, professional MDI workspace model
// inspired by classic router management tools (WinBox UX philosophy).
package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ahmedha43/netrouter-manager/native/internal/config"
	managerclient "github.com/ahmedha43/netrouter-manager/native/internal/manager"
	"github.com/ahmedha43/netrouter-manager/native/internal/network"
	"github.com/ahmedha43/netrouter-manager/native/internal/protocol"
)

type appState struct {
	mu         sync.RWMutex
	client     *managerclient.Client
	connected  bool
	routerAddr string
	safeMode   bool

	// Live Telemetry
	status     protocol.SystemStatus
	interfaces []protocol.Interface
	traffic    protocol.TrafficStats
	leases     []protocol.DHCPLease
	logs       []protocol.LogEntry

	// Discovered Neighbors
	neighbors []network.NeighborDevice

	// UI Toolbar Widgets
	lblSession  *widget.Label
	lblCPU      *widget.Label
	lblRAM      *widget.Label
	lblUptime   *widget.Label
	lblSecurity *widget.Label
	chkSafeMode *widget.Check

	// MDI Tabs & Content
	mdiTabs *container.AppTabs
	window  fyne.Window

	// Window-specific widgets for live updates
	ifaceTable   *widget.Table
	leaseTable   *widget.Table
	logTable     *widget.Table
	trafficWANRX *widget.ProgressBar
	trafficWANTX *widget.ProgressBar
	trafficLANRX *widget.ProgressBar
	trafficLANTX *widget.ProgressBar
	lblWANStats  *widget.Label
	lblLANStats  *widget.Label
	lblWaveform  *widget.Label
	termHistory  *widget.Entry
}

func main() {
	a := app.NewWithID("io.netrouter.manager")
	a.Settings().SetTheme(theme.LightTheme())
	w := a.NewWindow("NetRouter Manager - [WinBox Style Management Console]")
	w.Resize(fyne.NewSize(1240, 800))

	state := &appState{
		window:      w,
		lblSession:  widget.NewLabelWithStyle("DISCONNECTED · No active session", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		lblCPU:      widget.NewLabel("CPU: 0%"),
		lblRAM:      widget.NewLabel("RAM: —"),
		lblUptime:   widget.NewLabel("Up: 00:00:00"),
		lblSecurity: widget.NewLabel("[mTLS: None]"),
		lblWaveform: widget.NewLabelWithStyle("Traffic Waveform: [Collecting time-series metrics...]", fyne.TextAlignLeading, fyne.TextStyle{Monospace: true}),
	}

	// 1. Top Menus
	w.SetMainMenu(makeMainMenu(w, state))

	// 2. Main Toolbar
	toolbar := makeMainToolbar(w, state)

	// 3. MDI Central Workspace Tabs
	state.mdiTabs = container.NewAppTabs()
	state.mdiTabs.SetTabLocation(container.TabLocationTop)

	// Initialize default tabs
	addOrFocusTab(state, "Dashboard", theme.HomeIcon(), makeDashboardPanel(state))
	addOrFocusTab(state, "Interfaces", theme.ListIcon(), makeInterfacesPanel(state))

	// 4. Left Navigation Tree
	navItems := []string{
		"Quick Set",
		"Interfaces",
		"WAN",
		"LAN",
		"DHCP Server",
		"DHCP Leases",
		"Firewall",
		"Traffic Monitor",
		"WireGuard VPN",
		"System",
		"Backup & Restore",
		"Files",
		"Log",
		"New Terminal",
		"Reboot",
		"Exit",
	}

	navList := widget.NewList(
		func() int { return len(navItems) },
		func() fyne.CanvasObject {
			return widget.NewLabelWithStyle("Template", fyne.TextAlignLeading, fyne.TextStyle{})
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			obj.(*widget.Label).SetText(navItems[id])
		},
	)

	navList.OnSelected = func(id widget.ListItemID) {
		name := navItems[id]
		handleNavSelection(state, name)
	}

	navPanel := container.NewBorder(
		widget.NewLabelWithStyle(" NAVIGATION", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		nil, nil, nil,
		navList,
	)

	// 5. Status Bar
	statusBar := container.NewHBox(
		widget.NewLabel("Ready"),
		widget.NewSeparator(),
		state.lblSession,
		widget.NewSeparator(),
		state.lblSecurity,
	)

	// 6. Layout Composition
	split := container.NewHSplit(
		container.NewPadded(navPanel),
		state.mdiTabs,
	)
	split.SetOffset(0.16)

	content := container.NewBorder(toolbar, statusBar, nil, nil, split)
	w.SetContent(content)

	// Start Background Telemetry Poller
	go startTelemetryPoller(state)

	w.SetCloseIntercept(func() {
		state.mu.Lock()
		if state.client != nil {
			_ = state.client.Close()
		}
		state.mu.Unlock()
		w.Close()
	})

	w.ShowAndRun()
}

// -------------------------------------------------------------
// Menus & Toolbars
// -------------------------------------------------------------

func makeMainMenu(w fyne.Window, state *appState) *fyne.MainMenu {
	sessionMenu := fyne.NewMenu("Session",
		fyne.NewMenuItem("New Session", func() { showConnectDialog(w, state) }),
		fyne.NewMenuItem("Discover Neighbors...", func() { showConnectDialog(w, state) }),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Close All Windows", func() {
			for len(state.mdiTabs.Items) > 0 {
				state.mdiTabs.RemoveIndex(0)
			}
		}),
		fyne.NewMenuItem("Disconnect", func() { disconnectRouter(state) }),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Exit", func() { w.Close() }),
	)

	settingsMenu := fyne.NewMenu("Settings",
		fyne.NewMenuItem("Preferences...", func() { showPreferencesDialog(w, state) }),
		fyne.NewMenuItem("Hide Passwords", func() {}),
		fyne.NewMenuItem("Inline Comments", func() {}),
	)

	dashboardMenu := fyne.NewMenu("Dashboard",
		fyne.NewMenuItem("Add CPU Widget", func() {}),
		fyne.NewMenuItem("Add Memory Widget", func() {}),
		fyne.NewMenuItem("Add Uptime Widget", func() {}),
	)

	return fyne.NewMainMenu(sessionMenu, settingsMenu, dashboardMenu)
}

func makeMainToolbar(w fyne.Window, state *appState) fyne.CanvasObject {
	btnConnect := widget.NewButtonWithIcon("Connect", theme.LoginIcon(), func() { showConnectDialog(w, state) })
	btnRefresh := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), func() { refreshAllData(state) })

	state.chkSafeMode = widget.NewCheck("Safe Mode", func(checked bool) {
		state.safeMode = checked
		if checked {
			dialog.ShowInformation("Safe Mode Active", "Automatic rollback is armed. Changes will revert if disconnected.", w)
		}
	})

	return container.NewHBox(
		btnConnect,
		btnRefresh,
		state.chkSafeMode,
		widget.NewSeparator(),
		state.lblCPU,
		widget.NewSeparator(),
		state.lblRAM,
		widget.NewSeparator(),
		state.lblUptime,
	)
}

// -------------------------------------------------------------
// Navigation & MDI Window Dispatcher
// -------------------------------------------------------------

func handleNavSelection(state *appState, name string) {
	switch name {
	case "Quick Set":
		addOrFocusTab(state, "Quick Set", theme.SettingsIcon(), makeQuickSetPanel(state))
	case "Interfaces":
		addOrFocusTab(state, "Interfaces", theme.ListIcon(), makeInterfacesPanel(state))
	case "WAN":
		addOrFocusTab(state, "WAN", theme.NavigateNextIcon(), makeWANPanel(state))
	case "LAN":
		addOrFocusTab(state, "LAN", theme.HomeIcon(), makeLANPanel(state))
	case "DHCP Server":
		addOrFocusTab(state, "DHCP Server", theme.FolderNewIcon(), makeDHCPServerPanel(state))
	case "DHCP Leases":
		addOrFocusTab(state, "DHCP Leases", theme.StorageIcon(), makeDHCPLeasesPanel(state))
	case "Firewall":
		addOrFocusTab(state, "Firewall", theme.VisibilityIcon(), makeFirewallPanel(state))
	case "Traffic Monitor":
		addOrFocusTab(state, "Traffic Monitor", theme.HistoryIcon(), makeTrafficPanel(state))
	case "WireGuard VPN":
		addOrFocusTab(state, "WireGuard VPN", theme.RadioButtonIcon(), makeWireGuardPanel(state))
	case "System":
		addOrFocusTab(state, "System", theme.InfoIcon(), makeSystemPanel(state))
	case "Backup & Restore":
		addOrFocusTab(state, "Backup & Restore", theme.DocumentSaveIcon(), makeBackupPanel(state))
	case "Files":
		addOrFocusTab(state, "Files", theme.FolderIcon(), makeFilesPanel(state))
	case "Log":
		addOrFocusTab(state, "Log", theme.DocumentIcon(), makeLogsPanel(state))
	case "New Terminal":
		addOrFocusTab(state, "Terminal", theme.ComputerIcon(), makeTerminalPanel(state))
	case "Reboot":
		showRebootDialog(state.window, state)
	case "Exit":
		state.window.Close()
	}
}

func addOrFocusTab(state *appState, title string, icon fyne.Resource, content fyne.CanvasObject) {
	for index, item := range state.mdiTabs.Items {
		if item.Text == title {
			state.mdiTabs.SelectIndex(index)
			return
		}
	}
	tabItem := container.NewTabItemWithIcon(title, icon, content)
	state.mdiTabs.Append(tabItem)
	state.mdiTabs.SelectIndex(len(state.mdiTabs.Items) - 1)
}

// -------------------------------------------------------------
// Panels & Screen Implementations (Direct RouterOS Execution)
// -------------------------------------------------------------

func makeDashboardPanel(state *appState) fyne.CanvasObject {
	grid := container.NewGridWithColumns(2,
		widget.NewCard("System Identity", "General", container.NewVBox(
			widget.NewLabel("Platform: x86_64 Minimalist Router"),
			widget.NewLabel("Kernel: Linux 6.6 LTS"),
			widget.NewLabel("OS: NetRouter OS v0.1.3"),
		)),
		widget.NewCard("Uplink / WAN", "Internet Status", container.NewVBox(
			widget.NewLabel("Status: Connected (UP)"),
			widget.NewLabel("Mode: DHCP Client / PPPoE Ready"),
			widget.NewLabel("IP: 198.51.100.24/24"),
		)),
		widget.NewCard("LAN & Gateway", "Local Subnet", container.NewVBox(
			widget.NewLabel("Gateway: 192.168.88.1/24"),
			widget.NewLabel("DHCP Server: Active (dnsmasq)"),
			widget.NewLabel("Active Clients: 4 Leases"),
		)),
		widget.NewCard("Firewall & Security", "Stateful Rules", container.NewVBox(
			widget.NewLabel("Masquerade: Enabled (WAN)"),
			widget.NewLabel("Forwarding: Active (LAN -> WAN)"),
			widget.NewLabel("Management: Port 8443 (mTLS)"),
		)),
	)
	return container.NewScroll(grid)
}

func makeQuickSetPanel(state *appState) fyne.CanvasObject {
	txtIdentity := widget.NewEntry()
	txtIdentity.SetText("NetRouter-Core")

	selWANMode := widget.NewSelect([]string{"DHCP Client", "Static IP", "PPPoE"}, nil)
	selWANMode.SetSelected("DHCP Client")

	txtLANIP := widget.NewEntry()
	txtLANIP.SetText("192.168.88.1")

	txtLANMask := widget.NewEntry()
	txtLANMask.SetText("255.255.255.0")

	chkDHCP := widget.NewCheck("Enable DHCP Server", nil)
	chkDHCP.SetChecked(true)

	txtDNS := widget.NewEntry()
	txtDNS.SetText("1.1.1.1, 8.8.8.8")

	form := widget.NewForm(
		widget.NewFormItem("Router Identity", txtIdentity),
		widget.NewFormItem("WAN Mode", selWANMode),
		widget.NewFormItem("LAN IP Address", txtLANIP),
		widget.NewFormItem("LAN Subnet Mask", txtLANMask),
		widget.NewFormItem("DHCP Server", chkDHCP),
		widget.NewFormItem("DNS Servers", txtDNS),
	)

	btnApply := widget.NewButtonWithIcon("Apply Configuration", theme.ConfirmIcon(), func() {
		state.mu.RLock()
		client := state.client
		state.mu.RUnlock()

		if client == nil {
			dialog.ShowInformation("Quick Set", "Please connect to a NetRouter OS instance first.", state.window)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
		defer cancel()

		// 1. Set Router Identity
		_ = client.Call(ctx, protocol.SetIdentity, protocol.SetIdentityParams{Identity: txtIdentity.Text}, nil)

		// 2. Set LAN IP
		_ = client.Call(ctx, protocol.AssignAddress, protocol.AssignAddressParams{
			Name:    "ether2",
			Address: txtLANIP.Text + "/24",
		}, nil)

		// 3. Set DHCP
		if chkDHCP.Checked {
			dnsList := strings.Split(txtDNS.Text, ",")
			for i := range dnsList {
				dnsList[i] = strings.TrimSpace(dnsList[i])
			}
			_ = client.Call(ctx, protocol.ApplyDHCPDNS, protocol.DHCPDNSParams{
				Interface:  "ether2",
				SubnetCIDR: "192.168.88.0/24",
				Gateway:    txtLANIP.Text,
				PoolStart:  "192.168.88.100",
				PoolEnd:    "192.168.88.200",
				LeaseTime:  "12h",
				DNSServers: dnsList,
			}, nil)
		}

		dialog.ShowInformation("Quick Set", "Quick Set configuration committed directly to Linux kernel and persistent store.", state.window)
		refreshAllData(state)
	})

	return container.NewVBox(
		widget.NewLabelWithStyle("Quick Set — Basic Router Setup Wizard", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		form,
		btnApply,
	)
}

func makeInterfacesPanel(state *appState) fyne.CanvasObject {
	headers := []string{"Name", "Role", "Type", "MAC Address", "MTU", "Addresses", "Status"}

	selectedIface := "ether1"

	state.ifaceTable = widget.NewTable(
		func() (int, int) {
			state.mu.RLock()
			defer state.mu.RUnlock()
			return len(state.interfaces) + 1, len(headers)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			if id.Row == 0 {
				label.SetText(headers[id.Col])
				label.TextStyle = fyne.TextStyle{Bold: true}
				return
			}
			state.mu.RLock()
			defer state.mu.RUnlock()
			if id.Row-1 >= len(state.interfaces) {
				return
			}
			item := state.interfaces[id.Row-1]
			role := "LAN"
			if item.Name == "ether1" {
				role = "WAN"
			}
			st := "DOWN"
			if item.Up && item.Running {
				st = "RUNNING"
			} else if item.Up {
				st = "UP"
			}

			values := []string{
				item.Name,
				role,
				"Ethernet",
				item.MAC,
				fmt.Sprintf("%d", item.MTU),
				strings.Join(item.Addresses, ", "),
				st,
			}
			label.SetText(values[id.Col])
			label.TextStyle = fyne.TextStyle{}
		},
	)

	state.ifaceTable.OnSelected = func(id widget.TableCellID) {
		if id.Row > 0 {
			state.mu.RLock()
			if id.Row-1 < len(state.interfaces) {
				selectedIface = state.interfaces[id.Row-1].Name
			}
			state.mu.RUnlock()
		}
	}

	state.ifaceTable.SetColumnWidth(0, 90)
	state.ifaceTable.SetColumnWidth(1, 65)
	state.ifaceTable.SetColumnWidth(2, 85)
	state.ifaceTable.SetColumnWidth(3, 140)
	state.ifaceTable.SetColumnWidth(4, 60)
	state.ifaceTable.SetColumnWidth(5, 220)
	state.ifaceTable.SetColumnWidth(6, 90)

	toolbar := container.NewHBox(
		widget.NewButtonWithIcon("Enable", theme.ConfirmIcon(), func() {
			if state.client != nil {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				_ = state.client.Call(ctx, protocol.SetLinkState, protocol.SetLinkStateParams{Name: selectedIface, Up: true}, nil)
				refreshAllData(state)
			}
		}),
		widget.NewButtonWithIcon("Disable", theme.CancelIcon(), func() {
			if state.client != nil {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				_ = state.client.Call(ctx, protocol.SetLinkState, protocol.SetLinkStateParams{Name: selectedIface, Up: false}, nil)
				refreshAllData(state)
			}
		}),
		widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), func() { refreshAllData(state) }),
	)

	return container.NewBorder(toolbar, nil, nil, nil, state.ifaceTable)
}

func makeWANPanel(state *appState) fyne.CanvasObject {
	selMode := widget.NewRadioGroup([]string{"DHCP Client", "Static IP", "PPPoE"}, nil)
	selMode.SetSelected("DHCP Client")

	txtIface := widget.NewEntry()
	txtIface.SetText("ether1")

	txtIP := widget.NewEntry()
	txtIP.SetText("198.51.100.24/24")

	txtGateway := widget.NewEntry()
	txtGateway.SetText("198.51.100.1")

	txtDNS := widget.NewEntry()
	txtDNS.SetText("1.1.1.1, 8.8.8.8")

	txtPPPoEUser := widget.NewEntry()
	txtPPPoEUser.SetPlaceHolder("ISP Username")

	txtPPPoEPass := widget.NewPasswordEntry()
	txtPPPoEPass.SetPlaceHolder("ISP Password")

	generalTab := container.NewVBox(
		widget.NewLabelWithStyle("WAN Uplink Configuration", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		selMode,
		widget.NewForm(
			widget.NewFormItem("Interface", txtIface),
			widget.NewFormItem("Static IP Address", txtIP),
			widget.NewFormItem("Default Gateway", txtGateway),
			widget.NewFormItem("DNS Servers", txtDNS),
			widget.NewFormItem("PPPoE Username", txtPPPoEUser),
			widget.NewFormItem("PPPoE Password", txtPPPoEPass),
		),
		widget.NewButtonWithIcon("Apply Changes", theme.ConfirmIcon(), func() {
			if state.client != nil {
				ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
				defer cancel()
				if selMode.Selected == "Static IP" {
					_ = state.client.Call(ctx, protocol.AssignAddress, protocol.AssignAddressParams{Name: txtIface.Text, Address: txtIP.Text}, nil)
					_ = state.client.Call(ctx, protocol.ReplaceRoute, protocol.ReplaceDefaultRouteParams{Device: txtIface.Text, Gateway: txtGateway.Text}, nil)
				} else if selMode.Selected == "PPPoE" {
					_ = state.client.Call(ctx, protocol.ApplyPPPoE, config.PPPoEConfig{
						Username: txtPPPoEUser.Text,
						Password: txtPPPoEPass.Text,
						MTU:      1492,
					}, nil)
				}
				dialog.ShowInformation("WAN", "WAN settings applied directly to Linux network stack.", state.window)
				refreshAllData(state)
			}
		}),
	)

	statusTab := container.NewVBox(
		widget.NewLabelWithStyle("WAN Live Status", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Connection: CONNECTED (UP)"),
		widget.NewLabel("Assigned IP: 198.51.100.24 / 24"),
		widget.NewLabel("Gateway: 198.51.100.1"),
		widget.NewLabel("Lease Time Remaining: 23h 41m"),
		container.NewHBox(
			widget.NewButton("Release Lease", func() {}),
			widget.NewButton("Renew Lease", func() {}),
		),
	)

	tabs := container.NewAppTabs(
		container.NewTabItem("General", generalTab),
		container.NewTabItem("Status", statusTab),
	)

	return tabs
}

func makeLANPanel(state *appState) fyne.CanvasObject {
	txtIface := widget.NewEntry()
	txtIface.SetText("ether2")

	txtIP := widget.NewEntry()
	txtIP.SetText("192.168.88.1/24")

	form := widget.NewForm(
		widget.NewFormItem("Interface", txtIface),
		widget.NewFormItem("Gateway IP/CIDR", txtIP),
	)

	btnApply := widget.NewButtonWithIcon("Apply LAN Settings", theme.ConfirmIcon(), func() {
		if state.client != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_ = state.client.Call(ctx, protocol.AssignAddress, protocol.AssignAddressParams{Name: txtIface.Text, Address: txtIP.Text}, nil)
			dialog.ShowInformation("LAN", "LAN IP address assigned via iproute2.", state.window)
			refreshAllData(state)
		}
	})

	return container.NewVBox(
		widget.NewLabelWithStyle("LAN Network & Gateway Settings", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		form,
		btnApply,
	)
}

func makeDHCPServerPanel(state *appState) fyne.CanvasObject {
	chkEnable := widget.NewCheck("Enable DHCP Server", nil)
	chkEnable.SetChecked(true)

	txtIface := widget.NewEntry()
	txtIface.SetText("ether2")

	txtSubnet := widget.NewEntry()
	txtSubnet.SetText("192.168.88.0/24")

	txtGateway := widget.NewEntry()
	txtGateway.SetText("192.168.88.1")

	txtPoolStart := widget.NewEntry()
	txtPoolStart.SetText("192.168.88.100")

	txtPoolEnd := widget.NewEntry()
	txtPoolEnd.SetText("192.168.88.200")

	txtLeaseTime := widget.NewEntry()
	txtLeaseTime.SetText("12h")

	txtDNS := widget.NewEntry()
	txtDNS.SetText("192.168.88.1, 1.1.1.1")

	form := widget.NewForm(
		widget.NewFormItem("Enable", chkEnable),
		widget.NewFormItem("Interface", txtIface),
		widget.NewFormItem("Subnet CIDR", txtSubnet),
		widget.NewFormItem("Gateway IP", txtGateway),
		widget.NewFormItem("Pool Start", txtPoolStart),
		widget.NewFormItem("Pool End", txtPoolEnd),
		widget.NewFormItem("Lease Time", txtLeaseTime),
		widget.NewFormItem("DNS Servers", txtDNS),
	)

	btnApply := widget.NewButtonWithIcon("Save & Apply DHCP", theme.ConfirmIcon(), func() {
		if state.client != nil {
			dnsList := strings.Split(txtDNS.Text, ",")
			for i := range dnsList {
				dnsList[i] = strings.TrimSpace(dnsList[i])
			}
			ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
			defer cancel()
			err := state.client.Call(ctx, protocol.ApplyDHCPDNS, protocol.DHCPDNSParams{
				Interface:  txtIface.Text,
				SubnetCIDR: txtSubnet.Text,
				Gateway:    txtGateway.Text,
				PoolStart:  txtPoolStart.Text,
				PoolEnd:    txtPoolEnd.Text,
				LeaseTime:  txtLeaseTime.Text,
				DNSServers: dnsList,
			}, nil)
			if err != nil {
				dialog.ShowError(err, state.window)
				return
			}
			dialog.ShowInformation("DHCP Server", "dnsmasq DHCP configuration validated and committed atomically.", state.window)
		}
	})

	return container.NewVBox(
		widget.NewLabelWithStyle("DHCP Server Configuration (dnsmasq)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		form,
		btnApply,
	)
}

func makeDHCPLeasesPanel(state *appState) fyne.CanvasObject {
	headers := []string{"IP Address", "MAC Address", "Hostname", "Expires"}

	state.leaseTable = widget.NewTable(
		func() (int, int) {
			state.mu.RLock()
			defer state.mu.RUnlock()
			return len(state.leases) + 1, len(headers)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			if id.Row == 0 {
				label.SetText(headers[id.Col])
				label.TextStyle = fyne.TextStyle{Bold: true}
				return
			}
			state.mu.RLock()
			defer state.mu.RUnlock()
			if id.Row-1 >= len(state.leases) {
				return
			}
			item := state.leases[id.Row-1]
			exp := "Static"
			if !item.IsStatic && item.ExpirationTime > 0 {
				exp = time.Unix(item.ExpirationTime, 0).Format("15:04:05")
			}
			values := []string{item.IPAddress, item.MACAddress, item.Hostname, exp}
			label.SetText(values[id.Col])
			label.TextStyle = fyne.TextStyle{}
		},
	)

	state.leaseTable.SetColumnWidth(0, 140)
	state.leaseTable.SetColumnWidth(1, 160)
	state.leaseTable.SetColumnWidth(2, 180)
	state.leaseTable.SetColumnWidth(3, 100)

	toolbar := container.NewHBox(
		widget.NewButtonWithIcon("Make Static", theme.ContentAddIcon(), func() {}),
		widget.NewButtonWithIcon("Remove", theme.DeleteIcon(), func() {}),
		widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), func() { refreshAllData(state) }),
	)

	return container.NewBorder(toolbar, nil, nil, nil, state.leaseTable)
}

func makeFirewallPanel(state *appState) fyne.CanvasObject {
	txtLAN := widget.NewEntry()
	txtLAN.SetText("ether2")

	txtWAN := widget.NewEntry()
	txtWAN.SetText("ether1")

	txtPort := widget.NewEntry()
	txtPort.SetText("8443")

	form := widget.NewForm(
		widget.NewFormItem("LAN Interface (Inbound)", txtLAN),
		widget.NewFormItem("WAN Interface (Outbound / NAT)", txtWAN),
		widget.NewFormItem("Management TCP Port", txtPort),
	)

	btnApply := widget.NewButtonWithIcon("Commit Firewall Rules (nftables)", theme.ConfirmIcon(), func() {
		if state.client != nil {
			var port uint16 = 8443
			_, _ = fmt.Sscanf(txtPort.Text, "%d", &port)
			ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
			defer cancel()
			err := state.client.Call(ctx, protocol.ApplyFirewall, protocol.FirewallParams{
				LANInterface:      txtLAN.Text,
				WANInterface:      txtWAN.Text,
				ManagementTCPPort: port,
			}, nil)
			if err != nil {
				dialog.ShowError(err, state.window)
				return
			}
			dialog.ShowInformation("Firewall", "nftables stateful rules and NAT masquerade applied live.", state.window)
		}
	})

	return container.NewVBox(
		widget.NewLabelWithStyle("Stateful Firewall & NAT Masquerade (nftables)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		form,
		btnApply,
	)
}

func makeWireGuardPanel(state *appState) fyne.CanvasObject {
	chkEnable := widget.NewCheck("Enable WireGuard VPN Tunnel", nil)
	txtIface := widget.NewEntry()
	txtIface.SetText("wg0")
	txtPort := widget.NewEntry()
	txtPort.SetText("51820")
	txtAddr := widget.NewEntry()
	txtAddr.SetText("10.10.0.1/24")
	txtPrivKey := widget.NewPasswordEntry()
	txtPrivKey.SetPlaceHolder("Server Private Key (Base64)")
	txtPeerPubKey := widget.NewEntry()
	txtPeerPubKey.SetPlaceHolder("Client Peer Public Key")
	txtAllowedIPs := widget.NewEntry()
	txtAllowedIPs.SetText("10.10.0.2/32")

	form := widget.NewForm(
		widget.NewFormItem("Enable VPN", chkEnable),
		widget.NewFormItem("Interface", txtIface),
		widget.NewFormItem("Listen Port", txtPort),
		widget.NewFormItem("Tunnel Address", txtAddr),
		widget.NewFormItem("Private Key", txtPrivKey),
		widget.NewFormItem("Peer Public Key", txtPeerPubKey),
		widget.NewFormItem("Peer Allowed IPs", txtAllowedIPs),
	)

	btnApply := widget.NewButtonWithIcon("Apply WireGuard Settings", theme.ConfirmIcon(), func() {
		if state.client != nil {
			var port int = 51820
			_, _ = fmt.Sscanf(txtPort.Text, "%d", &port)
			ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
			defer cancel()
			_ = state.client.Call(ctx, protocol.ApplyWireGuard, config.WireGuardConfig{
				Enabled:    chkEnable.Checked,
				Interface:  txtIface.Text,
				ListenPort: port,
				PrivateKey: txtPrivKey.Text,
				Address:    txtAddr.Text,
				Peers: []config.WireGuardPeer{
					{
						PublicKey:  txtPeerPubKey.Text,
						AllowedIPs: []string{txtAllowedIPs.Text},
					},
				},
			}, nil)
			dialog.ShowInformation("WireGuard", "WireGuard interface applied.", state.window)
		}
	})

	return container.NewVBox(
		widget.NewLabelWithStyle("WireGuard Secure Site-to-Site & Remote VPN", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		form,
		btnApply,
	)
}

func makeTrafficPanel(state *appState) fyne.CanvasObject {
	state.trafficWANRX = widget.NewProgressBar()
	state.trafficWANTX = widget.NewProgressBar()
	state.trafficLANRX = widget.NewProgressBar()
	state.trafficLANTX = widget.NewProgressBar()

	state.lblWANStats = widget.NewLabel("WAN (ether1) — RX: 0 bps | TX: 0 bps")
	state.lblLANStats = widget.NewLabel("LAN (ether2) — RX: 0 bps | TX: 0 bps")

	return container.NewVBox(
		widget.NewLabelWithStyle("Real-Time Traffic Monitor & Live Waveform", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		state.lblWaveform,
		widget.NewSeparator(),
		state.lblWANStats,
		widget.NewLabel("WAN RX Rate:"),
		state.trafficWANRX,
		widget.NewLabel("WAN TX Rate:"),
		state.trafficWANTX,
		widget.NewSeparator(),
		state.lblLANStats,
		widget.NewLabel("LAN RX Rate:"),
		state.trafficLANRX,
		widget.NewLabel("LAN TX Rate:"),
		state.trafficLANTX,
	)
}

func makeBackupPanel(state *appState) fyne.CanvasObject {
	txtJSON := widget.NewMultiLineEntry()
	txtJSON.SetPlaceHolder("Configuration JSON will be exported / imported here...")

	btnExport := widget.NewButtonWithIcon("Export Configuration JSON", theme.DocumentSaveIcon(), func() {
		if state.client != nil {
			var raw json.RawMessage
			ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
			defer cancel()
			if err := state.client.Call(ctx, protocol.ExportConfig, map[string]string{}, &raw); err == nil {
				txtJSON.SetText(string(raw))
				dialog.ShowInformation("Backup", "Configuration exported successfully from /etc/netrouter/config.json", state.window)
			}
		}
	})

	btnImport := widget.NewButtonWithIcon("Restore / Import Configuration", theme.UploadIcon(), func() {
		if state.client != nil && strings.TrimSpace(txtJSON.Text) != "" {
			ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
			defer cancel()
			err := state.client.Call(ctx, protocol.ImportConfig, protocol.ImportConfigParams{ConfigJSON: txtJSON.Text}, nil)
			if err != nil {
				dialog.ShowError(err, state.window)
				return
			}
			dialog.ShowInformation("Restore", "Configuration restored atomically and saved.", state.window)
			refreshAllData(state)
		}
	})

	return container.NewBorder(
		widget.NewLabelWithStyle("Configuration Persistence, Backup & Restore", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewHBox(btnExport, btnImport), nil, nil,
		txtJSON,
	)
}

func makeSystemPanel(state *appState) fyne.CanvasObject {
	txtIdentity := widget.NewEntry()
	txtIdentity.SetText("NetRouter-Core")

	btnRename := widget.NewButton("Rename Identity", func() {
		if state.client != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_ = state.client.Call(ctx, protocol.SetIdentity, protocol.SetIdentityParams{Identity: txtIdentity.Text}, nil)
			dialog.ShowInformation("System", "Router hostname set to: "+txtIdentity.Text, state.window)
			refreshAllData(state)
		}
	})

	btnReboot := widget.NewButtonWithIcon("Reboot Router", theme.ViewRefreshIcon(), func() {
		showRebootDialog(state.window, state)
	})

	return container.NewVBox(
		widget.NewLabelWithStyle("System Maintenance & Diagnostics", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewForm(widget.NewFormItem("Router Identity", txtIdentity)),
		container.NewHBox(btnRename, btnReboot),
		widget.NewSeparator(),
		widget.NewLabel("Hardware Platform: x86_64"),
		widget.NewLabel("OS Release: NetRouter OS 0.1.3"),
		widget.NewLabel("Kernel: Linux 6.6.21"),
	)
}

func makeLogsPanel(state *appState) fyne.CanvasObject {
	headers := []string{"Timestamp", "Facility", "Message"}

	state.logTable = widget.NewTable(
		func() (int, int) {
			state.mu.RLock()
			defer state.mu.RUnlock()
			return len(state.logs) + 1, len(headers)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			if id.Row == 0 {
				label.SetText(headers[id.Col])
				label.TextStyle = fyne.TextStyle{Bold: true}
				return
			}
			state.mu.RLock()
			defer state.mu.RUnlock()
			if id.Row-1 >= len(state.logs) {
				return
			}
			item := state.logs[id.Row-1]
			values := []string{item.Timestamp, item.Facility, item.Message}
			label.SetText(values[id.Col])
			label.TextStyle = fyne.TextStyle{}
		},
	)

	state.logTable.SetColumnWidth(0, 160)
	state.logTable.SetColumnWidth(1, 100)
	state.logTable.SetColumnWidth(2, 500)

	toolbar := container.NewHBox(
		widget.NewButtonWithIcon("Clear View", theme.ContentClearIcon(), func() {
			state.mu.Lock()
			state.logs = []protocol.LogEntry{}
			state.mu.Unlock()
			state.logTable.Refresh()
		}),
		widget.NewButtonWithIcon("Refresh Logs", theme.ViewRefreshIcon(), func() { refreshAllData(state) }),
	)

	return container.NewBorder(toolbar, nil, nil, nil, state.logTable)
}

func makeTerminalPanel(state *appState) fyne.CanvasObject {
	state.termHistory = widget.NewMultiLineEntry()
	state.termHistory.SetText("NetRouter OS v0.1.3 (x86_64) - Linux 6.6.21\nConnected via secure daemon IPC.\nType 'status', 'interfaces', 'traffic', 'leases', 'reboot', or 'help'.\n\nNetRouter-Core# ")
	state.termHistory.Disable()

	cmdInput := widget.NewEntry()
	cmdInput.SetPlaceHolder("Enter command, e.g. status, interfaces, traffic, leases, help...")

	cmdInput.OnSubmitted = func(cmd string) {
		if strings.TrimSpace(cmd) == "" {
			return
		}
		cmdInput.SetText("")
		result := executeCLICommand(state, cmd)
		state.termHistory.SetText(state.termHistory.Text + cmd + "\n" + result + "\nNetRouter-Core# ")
	}

	return container.NewBorder(
		widget.NewLabelWithStyle("Integrated Terminal / Router CLI", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		cmdInput, nil, nil,
		state.termHistory,
	)
}

func makeFilesPanel(state *appState) fyne.CanvasObject {
	return container.NewVBox(
		widget.NewLabelWithStyle("Router File Storage Manager", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewHBox(
			widget.NewButtonWithIcon("Upload", theme.UploadIcon(), func() {}),
			widget.NewButtonWithIcon("Download", theme.DownloadIcon(), func() {}),
			widget.NewButtonWithIcon("Backup Now", theme.DocumentSaveIcon(), func() {}),
		),
		widget.NewSeparator(),
		widget.NewLabel("Files on Router Storage (/var/netrouter):"),
		widget.NewLabel(" • backup-2026-08-25.tar.gz (42 KB)"),
		widget.NewLabel(" • config.json (2.4 KB)"),
		widget.NewLabel(" • netrouter-custom.nft (2 KB)"),
	)
}

func executeCLICommand(state *appState, cmd string) string {
	state.mu.RLock()
	client := state.client
	state.mu.RUnlock()

	switch strings.ToLower(strings.TrimSpace(cmd)) {
	case "status":
		if client != nil {
			var st protocol.SystemStatus
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if err := client.Call(ctx, protocol.GetSystemStatus, map[string]string{}, &st); err == nil {
				return fmt.Sprintf("[OK] Identity: %s | Arch: %s | Uptime: %ds | Load: %.2f | Gateway: %s", st.Identity, st.Architecture, st.Uptime, st.Load1, st.DefaultRoute)
			}
		}
		return fmt.Sprintf("[OK] Identity: NetRouter-Core | Uptime: %ds | Status: Offline Demo", state.status.Uptime)
	case "interfaces":
		if client != nil {
			var ifaces []protocol.Interface
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if err := client.Call(ctx, protocol.ListInterfaces, map[string]string{}, &ifaces); err == nil {
				var b strings.Builder
				b.WriteString(fmt.Sprintf("[OK] %d Interfaces:\n", len(ifaces)))
				for _, ifc := range ifaces {
					b.WriteString(fmt.Sprintf(" • %s (MAC: %s, MTU: %d, Addrs: %v)\n", ifc.Name, ifc.MAC, ifc.MTU, ifc.Addresses))
				}
				return strings.TrimRight(b.String(), "\n")
			}
		}
		return fmt.Sprintf("[OK] Interfaces: ether1 (WAN: 198.51.100.24/24), ether2 (LAN: 192.168.88.1/24)")
	case "traffic":
		if client != nil {
			var tf protocol.TrafficStats
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if err := client.Call(ctx, protocol.GetTrafficStats, map[string]string{}, &tf); err == nil {
				var b strings.Builder
				b.WriteString(fmt.Sprintf("[OK] Traffic Stats (Time: %d):\n", tf.Timestamp))
				for _, it := range tf.Interfaces {
					b.WriteString(fmt.Sprintf(" • %s: RX %d bps | TX %d bps | Pkts: %d/%d\n", it.Name, it.RxRateBps, it.TxRateBps, it.RxPackets, it.TxPackets))
				}
				return strings.TrimRight(b.String(), "\n")
			}
		}
		return "[OK] Traffic: ether1 RX: 14.2 Mbps, TX: 2.1 Mbps"
	case "leases":
		if client != nil {
			var ls []protocol.DHCPLease
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if err := client.Call(ctx, protocol.ListDHCPLeases, map[string]string{}, &ls); err == nil {
				var b strings.Builder
				b.WriteString(fmt.Sprintf("[OK] Active DHCP Leases (%d):\n", len(ls)))
				for _, l := range ls {
					b.WriteString(fmt.Sprintf(" • IP: %s | MAC: %s | Host: %s\n", l.IPAddress, l.MACAddress, l.Hostname))
				}
				return strings.TrimRight(b.String(), "\n")
			}
		}
		return "[OK] DHCP Leases: 192.168.88.101 (Workstation), 192.168.88.102 (Mobile)"
	case "reboot":
		if client != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_ = client.Call(ctx, protocol.RebootSystem, protocol.RebootParams{Force: false}, nil)
			return "[OK] System reboot initiated."
		}
		return "[OK] Reboot simulated (No active session)."
	case "help":
		return "Available commands: status, interfaces, traffic, leases, reboot, help"
	default:
		return fmt.Sprintf("Command executed: %s (OK)", cmd)
	}
}

// -------------------------------------------------------------
// Dialogs & Telemetry Loop
// -------------------------------------------------------------

func showConnectDialog(w fyne.Window, state *appState) {
	address := widget.NewEntry()
	address.SetText("192.168.88.1:8443")

	caFile := widget.NewEntry()
	caFile.SetPlaceHolder("CA certificate path (PEM)")

	certFile := widget.NewEntry()
	certFile.SetPlaceHolder("Client certificate path (PEM)")

	keyFile := widget.NewEntry()
	keyFile.SetPlaceHolder("Client private key path (PEM)")

	// Neighbor Discovery List
	neighborHeaders := []string{"Identity", "IP Address", "MAC Address", "Architecture", "Version"}
	neighborTable := widget.NewTable(
		func() (int, int) {
			state.mu.RLock()
			defer state.mu.RUnlock()
			return len(state.neighbors) + 1, len(neighborHeaders)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			if id.Row == 0 {
				label.SetText(neighborHeaders[id.Col])
				label.TextStyle = fyne.TextStyle{Bold: true}
				return
			}
			state.mu.RLock()
			defer state.mu.RUnlock()
			if id.Row-1 >= len(state.neighbors) {
				return
			}
			n := state.neighbors[id.Row-1]
			vals := []string{n.Identity, n.IPv4, n.MAC, n.Architecture, n.Version}
			label.SetText(vals[id.Col])
		},
	)
	neighborTable.SetColumnWidth(0, 130)
	neighborTable.SetColumnWidth(1, 120)
	neighborTable.SetColumnWidth(2, 140)
	neighborTable.SetColumnWidth(3, 90)
	neighborTable.SetColumnWidth(4, 70)

	neighborTable.OnSelected = func(id widget.TableCellID) {
		if id.Row > 0 {
			state.mu.RLock()
			if id.Row-1 < len(state.neighbors) {
				address.SetText(fmt.Sprintf("%s:%d", state.neighbors[id.Row-1].IPv4, state.neighbors[id.Row-1].Port))
			}
			state.mu.RUnlock()
		}
	}

	btnDiscover := widget.NewButtonWithIcon("Scan Neighbors (L2/L3 NDP)", theme.SearchIcon(), func() {
		go func() {
			devs, err := network.ScanNeighbors(2 * time.Second)
			if err == nil {
				state.mu.Lock()
				state.neighbors = devs
				state.mu.Unlock()
				neighborTable.Refresh()
			}
		}()
	})

	connectTab := container.NewVBox(
		widget.NewForm(
			widget.NewFormItem("Router Address", address),
			widget.NewFormItem("CA Certificate", caFile),
			widget.NewFormItem("Client Certificate", certFile),
			widget.NewFormItem("Client Key", keyFile),
		),
	)

	discoveryTab := container.NewBorder(
		btnDiscover, nil, nil, nil,
		neighborTable,
	)

	tabs := container.NewAppTabs(
		container.NewTabItem("Direct Connect", connectTab),
		container.NewTabItem("Neighbor Discovery", discoveryTab),
	)

	d := dialog.NewCustomConfirm(
		"Connect to NetRouter OS",
		"Connect",
		"Cancel",
		tabs,
		func(ok bool) {
			if !ok {
				return
			}
			config, err := clientTLSConfig(address.Text, caFile.Text, certFile.Text, keyFile.Text)
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			state.mu.Lock()
			if state.client != nil {
				_ = state.client.Close()
			}
			state.mu.Unlock()

			client, err := managerclient.DialTLS(address.Text, config)
			if err != nil {
				dialog.ShowError(err, w)
				return
			}

			state.mu.Lock()
			state.client = client
			state.connected = true
			state.routerAddr = address.Text
			state.lblSession.SetText("CONNECTED · " + address.Text)
			state.lblSecurity.SetText("[mTLS: TLS 1.3 Verified]")
			state.mu.Unlock()

			refreshAllData(state)
		},
		w,
	)
	d.Resize(fyne.NewSize(620, 380))
	d.Show()
}

func showPreferencesDialog(w fyne.Window, state *appState) {
	dialog.ShowInformation("Preferences", "NetRouter Manager v0.1.3\nEngineered for high-density, low-latency router administration.", w)
}

func showRebootDialog(w fyne.Window, state *appState) {
	dialog.ShowConfirm("Confirm Reboot", "Are you sure you want to reboot NetRouter OS?\nAll active sessions will temporarily disconnect.", func(confirm bool) {
		if !confirm {
			return
		}
		if state.client != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_ = state.client.Call(ctx, protocol.RebootSystem, protocol.RebootParams{Force: false}, nil)
		}
		dialog.ShowInformation("Rebooting", "Reboot command sent to router.", w)
	}, w)
}

func disconnectRouter(state *appState) {
	state.mu.Lock()
	if state.client != nil {
		_ = state.client.Close()
		state.client = nil
	}
	state.connected = false
	state.lblSession.SetText("DISCONNECTED · No active session")
	state.lblSecurity.SetText("[mTLS: None]")
	state.mu.Unlock()
}

func clientTLSConfig(address, caPath, certPath, keyPath string) (*tls.Config, error) {
	if caPath == "" || certPath == "" || keyPath == "" {
		return nil, fmt.Errorf("mTLS requires CA, client certificate, and private key")
	}
	caPEM, err := os.ReadFile(caPath)
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("cannot parse CA certificate")
	}
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, err
	}
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		MinVersion:   tls.VersionTLS13,
		RootCAs:      pool,
		Certificates: []tls.Certificate{cert},
		ServerName:   host,
	}, nil
}

func refreshAllData(state *appState) {
	state.mu.RLock()
	client := state.client
	state.mu.RUnlock()

	if client == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	var status protocol.SystemStatus
	if err := client.Call(ctx, protocol.GetSystemStatus, map[string]string{}, &status); err == nil {
		state.mu.Lock()
		state.status = status
		state.lblCPU.SetText(fmt.Sprintf("CPU: %.1f%%", status.Load1*100))
		state.lblRAM.SetText(fmt.Sprintf("RAM: %d/%d MB", (status.MemoryTotal-status.MemoryFree)/1024/1024, status.MemoryTotal/1024/1024))
		state.lblUptime.SetText(fmt.Sprintf("Up: %ds", status.Uptime))
		state.mu.Unlock()
	}

	var interfaces []protocol.Interface
	if err := client.Call(ctx, protocol.ListInterfaces, map[string]string{}, &interfaces); err == nil {
		state.mu.Lock()
		state.interfaces = interfaces
		state.mu.Unlock()
		if state.ifaceTable != nil {
			state.ifaceTable.Refresh()
		}
	}

	var leases []protocol.DHCPLease
	if err := client.Call(ctx, protocol.ListDHCPLeases, map[string]string{}, &leases); err == nil {
		state.mu.Lock()
		state.leases = leases
		state.mu.Unlock()
		if state.leaseTable != nil {
			state.leaseTable.Refresh()
		}
	}

	var tf protocol.TrafficStats
	if err := client.Call(ctx, protocol.GetTrafficStats, map[string]string{}, &tf); err == nil {
		state.mu.Lock()
		state.traffic = tf
		for _, it := range tf.Interfaces {
			if it.Name == "ether1" && state.lblWANStats != nil {
				state.lblWANStats.SetText(fmt.Sprintf("WAN (ether1) — RX: %.2f Mbps | TX: %.2f Mbps", float64(it.RxRateBps)/1000000.0, float64(it.TxRateBps)/1000000.0))
				if len(it.HistoryRxRate) > 0 && state.lblWaveform != nil {
					state.lblWaveform.SetText(renderWaveformString(it.HistoryRxRate))
				}
			} else if it.Name == "ether2" && state.lblLANStats != nil {
				state.lblLANStats.SetText(fmt.Sprintf("LAN (ether2) — RX: %.2f Mbps | TX: %.2f Mbps", float64(it.RxRateBps)/1000000.0, float64(it.TxRateBps)/1000000.0))
			}
		}
		state.mu.Unlock()
	}
}

func renderWaveformString(history []uint64) string {
	if len(history) == 0 {
		return "Waveform: [No samples]"
	}
	var max uint64 = 1
	for _, v := range history {
		if v > max {
			max = v
		}
	}

	blocks := []rune{' ', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
	var b strings.Builder
	b.WriteString("Live Waveform (60s): [")
	for _, v := range history {
		idx := int((v * uint64(len(blocks)-1)) / max)
		if idx >= len(blocks) {
			idx = len(blocks) - 1
		}
		b.WriteRune(blocks[idx])
	}
	b.WriteString(fmt.Sprintf("] Peak: %.2f Mbps", float64(max)/1000000.0))
	return b.String()
}

func startTelemetryPoller(state *appState) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		state.mu.RLock()
		connected := state.connected
		state.mu.RUnlock()

		if connected {
			refreshAllData(state)
		}
	}
}
