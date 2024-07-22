package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	DependenciesNotInstalledError = errors.New("Required dependencies are not installed")

	_externalExections = []string{
		"ffmpeg",
		"xrandr",
	}
)

func serve(cmd *cobra.Command, args []string) error {

	// Set the display environment variable
	os.Setenv("DISPLAY", ":0")

	if err := checkDependencies(); err != nil {
		panic(err)
	}

	verbose, _ := cmd.Flags().GetBool("verbose")
	if verbose {
		fmt.Println("Verbose mode enabled")
	}

	// samba
	sambaHost := viper.GetString("samba_host")
	sambaPassword := viper.GetString("samba_password")
	sambaUsername := viper.GetString("samba_username")
	sambaShare := viper.GetString("samba_share")
	folder := viper.GetString("folder")
	samba := NewSamba(sambaHost, sambaShare, folder)
	if sambaUsername != "" && sambaPassword != "" {
		samba.WithCredentials(sambaUsername, sambaPassword)
	}
	desktopUsername := viper.GetString("desktop_username")

	interval := viper.GetInt("interval")
	if interval < 1 {
		return errors.New("Interval must be greater than 0")
	}
	timer := time.NewTicker(time.Duration(interval) * time.Second)

	go func() {
		fmt.Println("Starting the application")

		if err := samba.Start(); err != nil {
			panic(err)
		}

		fmt.Println("Samba started")

		for range timer.C {
			if !isDesktopActive(desktopUsername) {
				continue
			}
			reader, err := takeScrrenshot()
			if err != nil {
				fmt.Println("Error taking screenshot:", err)
				continue
			}
			if err := samba.UploadScreenshot(reader); err != nil {
				fmt.Println("Error uploading screenshot:", err)
			}
			reader.Close()
		}
	}()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan

	fmt.Println("Received an interrupt, stopping the application")
	_ = samba.Stop()
	return nil
}

// Check if the required dependencies are installed
func checkDependencies() error {
	for _, command := range _externalExections {
		_, err := exec.LookPath(command)
		if err != nil {
			return errors.Join(DependenciesNotInstalledError, err)
		}
	}
	return nil
}

func isDesktopActive(desktopUsername string) bool {
	// Check if the desktop is active
	// loginctl list-sessions
	// SESSION  UID USER SEAT  TTY
	//       1 1000 u    seat0 tty2
	//    2728 1000 u          pts/17
	//    2729 1000 u
	//    2730 1000 u
	// loginctl show-session 1
	// Id=1
	// User=1000
	// Name=u
	// Timestamp=Mon 2024-07-08 09:44:17 CST
	// TimestampMonotonic=44836576
	// VTNr=2
	// Seat=seat0
	// TTY=tty2
	// Remote=no
	// Service=gdm-autologin
	// Scope=session-1.scope
	// Leader=6931
	// Audit=1
	// Type=x11
	// Class=user
	// Active=yes
	// State=active
	// IdleHint=no
	// IdleSinceHint=1721631957765280
	// IdleSinceHintMonotonic=1228944926373
	// LockedHint=no

	isUnlockedLocal := func(sessionID int) bool {
		cmd := exec.Command("loginctl", "show-session", fmt.Sprintf("%d", sessionID))
		output, err := cmd.Output()
		if err != nil {
			fmt.Printf("Error getting session info: %v\n", err)
		}

		remote, locked := false, false

		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			fields := strings.Split(line, "=")
			if len(fields) == 2 && fields[0] == "Remote" {
				remote = fields[1] == "no\n"
			} else if len(fields) == 2 && fields[0] == "LockedHint" {
				locked = fields[1] == "no\n"
			}
		}
		return !remote && !locked
	}

	// Parse the output to check if any session is active
	cmd := exec.Command("loginctl", "list-sessions")
	output, err := cmd.Output()
	if err != nil {
		panic(err)
	}
	lines := strings.Split(string(output), "\n")
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) >= 5 && fields[2] == desktopUsername {
			if sessionID, err := strconv.ParseInt(fields[0], 10, 64); err == nil {
				// fmt.Printf("Checking session %d\n", sessionID)
				if isUnlockedLocal(int(sessionID)) {
					return true
				}
			}
		}
	}

	return false
}

func takeScrrenshot() (io.ReadCloser, error) {
	// Take a screenshot of the current screen
	// xrandr
	// Screen 0: minimum 8 x 8, current 4480 x 1440, maximum 32767 x 32767
	// DVI-D-0 disconnected (normal left inverted right x axis y axis)
	// HDMI-0 connected 1920x1080+0+0 (normal left inverted right x axis y axis) 477mm x 268mm
	//    1920x1080     60.00*+  59.94    50.00    60.05    60.00    50.04
	//    1680x1050     59.95
	//    1600x900      60.00
	//    1280x1024     60.02
	//    1280x800      59.81
	//    1280x720      60.00    59.94    50.00
	//    1024x768      60.00
	//    800x600       60.32
	//    720x576       50.00
	//    720x480       59.94
	//    640x480       59.94    59.93
	// DP-0 connected primary 2560x1440+1920+0 (normal left inverted right x axis y axis) 597mm x 336mm
	//    2560x1440     59.95*+  74.97
	//    1920x1200     59.88
	//    1920x1080     60.00    59.94    50.00
	//    1680x1050     59.95
	//    1440x900      59.89
	//    1280x1024     75.02    60.02
	//    1280x960      60.00
	//    1280x720      60.00    59.94    50.00
	//    1152x864      75.00
	//    1024x768      75.03    70.07    60.00
	//    800x600       75.00    72.19    60.32    56.25
	//    720x576       50.00
	//    720x480       59.94
	//    640x480       75.00    72.81    59.94    59.93
	// DP-1 disconnected (normal left inverted right x axis y axis)

	// parse the output to get the screen resolution
	cmd := exec.Command("xrandr")
	output, err := cmd.Output()
	if err != nil {
		panic(err)
	}

	screenWidth, screenHeight := 0, 0
	lines := strings.Split(string(output), "\n")
	re := regexp.MustCompile(`current (\d+) x (\d+)`)
	for _, line := range lines {
		// Screen 0: minimum 8 x 8, current 4480 x 1440, maximum 32767 x 32767
		matches := re.FindStringSubmatch(line)
		if len(matches) == 3 {
			screenWidth, _ = strconv.Atoi(matches[1])
			screenHeight, _ = strconv.Atoi(matches[2])
			break
		}
	}

	// ffmpeg -f x11grab -video_size 3840x1080 -i :0.0 -vframes 1 -f image2pipe -vcodec png -
	cmd = exec.Command(
		"ffmpeg",
		"-f", "x11grab",
		"-video_size",
		fmt.Sprintf("%dx%d", screenWidth, screenHeight),
		"-i", ":0.0",
		"-vframes", "1",
		"-f", "image2pipe", "-vcodec", "png", "-",
	)
	cmd.Stderr = nil
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	err = cmd.Start()
	if err != nil {
		return nil, err
	}
	go func() {
		err := cmd.Wait()
		if err != nil {
			fmt.Printf("Error taking screenshot: %v\n", err)
		}
	}()
	return stdout, nil
}
