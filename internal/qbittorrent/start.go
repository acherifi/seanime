package qbittorrent

import (
	"errors"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

func (c *Client) getExecutableName() string {
	switch runtime.GOOS {
	case "windows":
		return "qbittorrent.exe"
	default:
		return "qbittorrent"
	}
}

func (c *Client) getExecutablePath() string {

	if len(c.Path) > 0 {
		return c.Path
	}

	switch runtime.GOOS {
	case "windows":
		return "C:/Program Files/qBittorrent/qbittorrent.exe"
	case "linux":
		return "/usr/bin/qbittorrent" // Default path for Client on most Linux distributions
	case "darwin":
		return "/Applications/Client.app/Contents/MacOS/qBittorrent" // Default path for Client on macOS
	default:
		return "C:/Program Files/qBittorrent/qbittorrent.exe"
	}
}

func (c *Client) isRunning(executable string) bool {
	cmd := exec.Command("tasklist")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	return strings.Contains(string(output), executable)
}

func (c *Client) Start() error {
	name := c.getExecutableName()
	exe := c.getExecutablePath()
	if c.isRunning(name) {
		return nil
	}

	cmd := exec.Command(exe)
	err := cmd.Start()
	if err != nil {
		return errors.New("failed to start qBittorrent")
	}

	time.Sleep(1 * time.Second)

	return nil
}

func (c *Client) CheckStart() bool {
	if c == nil {
		return false
	}

	_, err := c.Application.GetAppVersion()
	if err == nil {
		return true
	}

	err = c.Start()
	timeout := time.After(30 * time.Second)
	ticker := time.Tick(1 * time.Second)
	for {
		select {
		case <-ticker:
			_, err = c.Application.GetAppVersion()
			if err == nil {
				return true
			}
		case <-timeout:
			return false
		}
	}
}
