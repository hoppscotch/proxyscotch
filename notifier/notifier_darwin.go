package notifier

import (
	"github.com/deckarep/gosx-notifier"
)

func Notify(appName string, title string, message string, icon string) error {
	notification := gosxnotifier.NewNotification(message);
	notification.Title = appName;
	notification.Subtitle = title;
	notification.AppIcon = icon;
	notification.Sender = "io.postwoman.proxy";
	_ = notification.Push();
	return nil;
}