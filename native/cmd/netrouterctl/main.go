// netrouterctl is a local diagnostic client for the Unix management socket.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/ahmedha43/netrouter-manager/native/internal/protocol"
)

func main() {
	var socket, command string
	flag.StringVar(&socket, "socket", "/run/netrouterd.sock", "NetRouter Unix socket")
	flag.StringVar(&command, "command", "status", "status or interfaces")
	flag.Parse()
	method := protocol.GetSystemStatus
	if command == "interfaces" {
		method = protocol.ListInterfaces
	}
	connection, err := net.Dial("unix", socket)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer connection.Close()
	request := protocol.Request{Version: protocol.Version, ID: "ctl-1", Method: method}
	if err := json.NewEncoder(connection).Encode(request); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var response protocol.Response
	if err := json.NewDecoder(connection).Decode(&response); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if response.Error != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", response.Error.Code, response.Error.Message)
		os.Exit(1)
	}
	var pretty any
	_ = json.Unmarshal(response.Result, &pretty)
	output, _ := json.MarshalIndent(pretty, "", "  ")
	fmt.Println(string(output))
}
