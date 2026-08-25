// Design reminder — the independent Windows client follows NetRouter's compact
// technical desktop language: terse status, dense data, and no SaaS layout.
package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	managerclient "github.com/ahmedha43/netrouter-manager/native/internal/manager"
	"github.com/ahmedha43/netrouter-manager/native/internal/protocol"
)

type desktopState struct {
	client     *managerclient.Client
	status     *widget.Label
	systemRows []*widget.Label
	interfaces []protocol.Interface
	table      *widget.Table
}

func main() {
	application := app.NewWithID("io.netrouter.manager")
	application.Settings().SetTheme(theme.LightTheme())
	window := application.NewWindow("NetRouter Manager")
	window.Resize(fyne.NewSize(1120, 720))
	state := &desktopState{status: widget.NewLabel("DISCONNECTED · no router session")}
	state.status.TextStyle = fyne.TextStyle{Bold: true}
	connect := widget.NewButtonWithIcon("Connect", theme.LoginIcon(), func() { showConnectDialog(window, state) })
	refresh := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), func() { refreshRouter(state, window) })
	safeMode := widget.NewCheck("Safe Mode", nil)
	toolbar := container.NewHBox(connect, refresh, safeMode, widget.NewSeparator(), state.status)
	dashboard := makeDashboard(state)
	interfaces := makeInterfaceTable(state)
	tabs := container.NewAppTabs(container.NewTabItemWithIcon("Dashboard", theme.HomeIcon(), dashboard), container.NewTabItemWithIcon("Interfaces", theme.ListIcon(), interfaces), container.NewTabItemWithIcon("Traffic", theme.HistoryIcon(), widget.NewLabel("Live traffic charts will subscribe to telemetry in the next protocol revision.")))
	navigation := widget.NewList(func() int { return 10 }, func() fyne.CanvasObject { return widget.NewLabel("template") }, func(id widget.ListItemID, object fyne.CanvasObject) {
		object.(*widget.Label).SetText([]string{"Quick Set", "Interfaces", "WAN", "LAN", "DHCP Server", "Firewall", "Traffic Monitor", "System", "Log", "New Terminal"}[id])
	})
	navigation.OnSelected = func(id widget.ListItemID) {
		if id == 1 {
			tabs.SelectIndex(1)
		} else if id == 6 {
			tabs.SelectIndex(2)
		} else {
			tabs.SelectIndex(0)
		}
	}
	window.SetContent(container.NewBorder(toolbar, nil, container.NewVBox(widget.NewLabelWithStyle("NAVIGATION", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), navigation), nil, tabs))
	window.SetCloseIntercept(func() {
		if state.client != nil {
			_ = state.client.Close()
		}
		window.Close()
	})
	window.ShowAndRun()
}

func makeDashboard(state *desktopState) fyne.CanvasObject {
	keys := []string{"Identity", "Architecture", "Kernel", "Uptime", "Memory", "Load 1m", "Default Route"}
	grid := container.NewGridWithColumns(2)
	for _, key := range keys {
		grid.Add(widget.NewLabelWithStyle(key, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
		value := widget.NewLabel("—")
		state.systemRows = append(state.systemRows, value)
		grid.Add(value)
	}
	return container.NewPadded(widget.NewCard("System", "Connected router status", grid))
}

func makeInterfaceTable(state *desktopState) fyne.CanvasObject {
	headers := []string{"Name", "MAC Address", "MTU", "State", "Addresses"}
	state.table = widget.NewTable(func() (int, int) { return len(state.interfaces) + 1, len(headers) }, func() fyne.CanvasObject { return widget.NewLabel("template") }, func(id widget.TableCellID, object fyne.CanvasObject) {
		label := object.(*widget.Label)
		if id.Row == 0 {
			label.SetText(headers[id.Col])
			label.TextStyle = fyne.TextStyle{Bold: true}
			return
		}
		item := state.interfaces[id.Row-1]
		values := []string{item.Name, item.MAC, fmt.Sprintf("%d", item.MTU), stateText(item), strings.Join(item.Addresses, ", ")}
		label.SetText(values[id.Col])
		label.TextStyle = fyne.TextStyle{}
	})
	state.table.SetColumnWidth(0, 115)
	state.table.SetColumnWidth(1, 150)
	state.table.SetColumnWidth(2, 65)
	state.table.SetColumnWidth(3, 95)
	state.table.SetColumnWidth(4, 330)
	return container.NewBorder(nil, nil, nil, nil, state.table)
}

func stateText(item protocol.Interface) string {
	if item.Up && item.Running {
		return "RUNNING"
	}
	if item.Up {
		return "UP"
	}
	return "DOWN"
}

func showConnectDialog(window fyne.Window, state *desktopState) {
	address := widget.NewEntry()
	address.SetText("192.168.88.1:8443")
	caFile := widget.NewEntry()
	caFile.SetPlaceHolder("CA PEM path")
	certFile := widget.NewEntry()
	certFile.SetPlaceHolder("Manager certificate PEM path")
	keyFile := widget.NewEntry()
	keyFile.SetPlaceHolder("Manager private key PEM path")
	form := dialog.NewForm("Connect to NetRouter", "Connect", "Cancel", []*widget.FormItem{widget.NewFormItem("Router address", address), widget.NewFormItem("CA certificate", caFile), widget.NewFormItem("Client certificate", certFile), widget.NewFormItem("Client private key", keyFile)}, func(ok bool) {
		if !ok {
			return
		}
		config, err := clientTLSConfig(address.Text, caFile.Text, certFile.Text, keyFile.Text)
		if err != nil {
			dialog.ShowError(err, window)
			return
		}
		if state.client != nil {
			_ = state.client.Close()
		}
		client, err := managerclient.DialTLS(address.Text, config)
		if err != nil {
			dialog.ShowError(err, window)
			return
		}
		state.client = client
		state.status.SetText("CONNECTED · " + address.Text)
		refreshRouter(state, window)
	}, window)
	form.Resize(fyne.NewSize(520, 280))
	form.Show()
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
	certificate, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, err
	}
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	return &tls.Config{MinVersion: tls.VersionTLS13, RootCAs: pool, Certificates: []tls.Certificate{certificate}, ServerName: host}, nil
}

func refreshRouter(state *desktopState, window fyne.Window) {
	if state.client == nil {
		dialog.ShowInformation("No session", "Connect to a NetRouter OS device first.", window)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	var status protocol.SystemStatus
	if err := state.client.Call(ctx, protocol.GetSystemStatus, map[string]string{}, &status); err != nil {
		dialog.ShowError(err, window)
		return
	}
	values := []string{status.Identity, status.Architecture, status.Kernel, fmt.Sprintf("%ds", status.Uptime), fmt.Sprintf("%d MiB free", status.MemoryFree/1024/1024), fmt.Sprintf("%.2f", status.Load1), status.DefaultRoute}
	for index, value := range values {
		state.systemRows[index].SetText(value)
	}
	var interfaces []protocol.Interface
	if err := state.client.Call(ctx, protocol.ListInterfaces, map[string]string{}, &interfaces); err != nil {
		dialog.ShowError(err, window)
		return
	}
	state.interfaces = interfaces
	state.table.Refresh()
}
