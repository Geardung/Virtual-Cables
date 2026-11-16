//go:build windows

package main

import (
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"
)

// portConflictHint explains why the preferred USB/IP address could not be
// opened and names the process that is holding the port, when it can be found.
func portConflictHint(address string) string {
	port := listenPort(address)
	message := fmt.Sprintf("USB/IP port %d cannot be opened.", port)
	if owner := listeningProcess(port); owner != "" {
		message += fmt.Sprintf(" It is held by %s.", owner)
	} else {
		message += " It is held by another process."
	}
	message += " This is commonly the \"USBIP Device Host\" service from usbipd-win (usbipd.exe)," +
		" which occupies the same default port as Virtual Cables. Stop it with 'net stop usbipd'" +
		" if Virtual Cables should use this port. Virtual Cables will try another port."
	return message
}

func listeningProcess(port int) string {
	out, err := exec.Command("netstat", "-ano").CombinedOutput()
	if err != nil {
		return ""
	}
	pid := ""
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) < 5 || !strings.EqualFold(fields[0], "TCP") || !strings.EqualFold(fields[3], "LISTENING") {
			continue
		}
		_, portPart, splitErr := net.SplitHostPort(fields[1])
		if splitErr != nil || portPart != strconv.Itoa(port) {
			continue
		}
		pid = fields[4]
		break
	}
	if pid == "" {
		return ""
	}
	if out, err := exec.Command("tasklist", "/FI", "PID eq "+pid, "/FO", "CSV", "/NH").CombinedOutput(); err == nil {
		line := strings.TrimSpace(string(out))
		if line != "" && !strings.HasPrefix(line, "INFO") {
			if strings.HasPrefix(line, "\"") {
				if end := strings.Index(line[1:], "\""); end >= 0 {
					return fmt.Sprintf("%s (PID %s)", line[1:end+1], pid)
				}
			}
		}
	}
	return "PID " + pid
}
