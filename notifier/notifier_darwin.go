package notifier

func Notify(appName, title, message, icon string) error {
	notification := gosxnotifier.NewNotification(message)
	notification.Title = appName
	notification.Subtitle = title
	notification.AppIcon = icon
	notification.Sender = "io.hoppscotch.proxy"
	_ = notification.Push()
	return nil
}
