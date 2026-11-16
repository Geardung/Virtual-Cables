//go:build windows

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var usbipPortPattern = regexp.MustCompile(`(?i)^Port\s+(\d+):`)

func findUSBIPExecutable() string {
	if path, err := exec.LookPath("usbip.exe"); err == nil {
		return path
	}
	candidates := []string{
		filepath.Join(os.Getenv("ProgramFiles"), "USBip", "usbip.exe"),
		filepath.Join(os.Getenv("ProgramFiles"), "usbip-win2", "usbip.exe"),
		filepath.Join(os.Getenv("ProgramFiles"), "usbip", "usbip.exe"),
	}
	if programFilesX86 := os.Getenv("ProgramFiles(x86)"); programFilesX86 != "" {
		candidates = append(candidates,
			filepath.Join(programFilesX86, "USBip", "usbip.exe"),
			filepath.Join(programFilesX86, "usbip-win2", "usbip.exe"),
		)
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return ""
}

func runUSBIP(executable string, logger *log.Logger, arguments ...string) (string, error) {
	logger.Printf("usbip.exe %s", strings.Join(arguments, " "))
	command := exec.Command(executable, arguments...)
	hideCommandWindow(command)
	output, err := command.CombinedOutput()
	text := strings.TrimSpace(string(output))
	if text != "" {
		logger.Printf("usbip: %s", strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", " | "), "\n", " | "))
	}
	if err != nil {
		return text, fmt.Errorf("usbip %s: %w", strings.Join(arguments, " "), err)
	}
	return text, nil
}

// usbipArgs puts --tcp-port before the subcommand when the server is off the
// default port; 3240 is left implicit so older usbip.exe builds still work.
func usbipArgs(port int, arguments ...string) []string {
	if port > 0 && port != 3240 {
		return append([]string{"--tcp-port", strconv.Itoa(port)}, arguments...)
	}
	return arguments
}

func hideCommandWindow(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
}

func detachOwnUSBIPPorts(executable string, logger *log.Logger, port int) {
	output, err := runUSBIP(executable, logger, "port")
	if err != nil {
		logger.Printf("Could not inspect existing USB/IP ports: %v", err)
		return
	}
	markers := []string{fmt.Sprintf("usbip://127.0.0.1:%d/", port)}
	if port != 3240 {
		// Also clean up attachments left behind by previous default-port runs.
		markers = append(markers, "usbip://127.0.0.1:3240/")
	}
	currentPort := ""
	var ownPorts []string
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if match := usbipPortPattern.FindStringSubmatch(trimmed); len(match) == 2 {
			currentPort = match[1]
			continue
		}
		if currentPort != "" && containsAny(trimmed, markers) {
			ownPorts = append(ownPorts, currentPort)
			currentPort = ""
		}
	}
	for _, port := range ownPorts {
		if _, err := runUSBIP(executable, logger, "detach", "-p", port); err != nil {
			logger.Printf("Could not detach previous Virtual Cables port %s: %v", port, err)
		}
	}
}

func containsAny(text string, substrings []string) bool {
	for _, sub := range substrings {
		if strings.Contains(text, sub) {
			return true
		}
	}
	return false
}

func runAttachHelper(count, port int, logger *log.Logger) error {
	if count < 1 || count > 32 {
		return fmt.Errorf("invalid cable count %d", count)
	}
	if port < 1 || port > 65535 {
		return fmt.Errorf("invalid USB/IP TCP port %d", port)
	}
	executable := findUSBIPExecutable()
	if executable == "" {
		return fmt.Errorf("usbip.exe was not found; install the driver from the main window")
	}
	logger.Printf("Elevated Windows device synchronization started for %d cable(s) on TCP port %d", count, port)
	logger.Printf("usbip.exe: %s", executable)

	// The server can still be completing its restart when the elevated helper
	// starts. Retry only the local device-list request for a short bounded time.
	var listErr error
	for attempt := 1; attempt <= 20; attempt++ {
		if _, listErr = runUSBIP(executable, logger, usbipArgs(port, "list", "-r", "127.0.0.1")...); listErr == nil {
			break
		}
		time.Sleep(150 * time.Millisecond)
	}
	if listErr != nil {
		return fmt.Errorf("local USB/IP server is not ready: %w", listErr)
	}

	detachOwnUSBIPPorts(executable, logger, port)
	for number := 1; number <= count; number++ {
		busID := "1-" + strconv.Itoa(number)
		if _, err := runUSBIP(executable, logger, usbipArgs(port, "attach", "-r", "127.0.0.1", "-b", busID)...); err != nil {
			return fmt.Errorf("attach %s failed: %w", busID, err)
		}
	}
	logger.Printf("Windows device synchronization completed for %d cable(s)", count)
	return nil
}
