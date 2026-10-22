package notifier

import (
	"os/exec"
)

func Notify(appName, title, message, icon string) error {
	cmd := exec.Command(
		"notify-send",
		"--app-name", appName,
		"--icon", icon,
		title,
		message,
	)
	return cmd.Run()
}
