package main

import (
    "github.com/atotto/clipboard"
    "github.com/getlantern/systray"
    "github.com/martinlindhe/inputbox"
    "github.com/pkg/browser"

    "postwoman.io/proxy/icons"
    "postwoman.io/proxy/libproxy"
    "postwoman.io/proxy/notifier"
);

var mStatus *systray.MenuItem;
var mCopyAccessToken *systray.MenuItem;

func main() {
    systray.Run(onReady, onExit);
}

func onReady() {
    systray.SetIcon(icon.Data);
    systray.SetTooltip("Postwoman Proxy v1.0.0 - created by NBTX");

    /** Set up menu items. **/

    // Status
    mStatus = systray.AddMenuItem("Starting...", "");
    mStatus.Disable();
    mCopyAccessToken = systray.AddMenuItem("Copy Access Token...", "");
    mCopyAccessToken.Disable();

    systray.AddSeparator();

    // Open Postwoman Interface
    mOpenPostwoman := systray.AddMenuItem("Open Postwoman", "");

    systray.AddSeparator();

    // View Help
    mViewHelp := systray.AddMenuItem("Help...", "");
    // Set Proxy Authentication Token
    mSetAccessToken := systray.AddMenuItem("Set Access Token...", "");
    // Check for Updates
    mUpdateCheck := systray.AddMenuItem("Check for Updates...", "");

    systray.AddSeparator();

    // Quit Proxy
    mQuit := systray.AddMenuItem("Quit Postwoman Proxy", "");

    /** Start proxy server. **/
    go runPostwomanProxy();

    /** Wait for menu input. **/
    for {
        select {
            case <- mOpenPostwoman.ClickedCh:
                _ = browser.OpenURL("https://postwoman.io/");

            case <- mCopyAccessToken.ClickedCh:
                _ = clipboard.WriteAll(libproxy.GetAccessToken());
                _ = notifier.Notify("Postwoman Proxy", "Proxy Access Token copied...", "The Proxy Access Token has been copied to your clipboard.", notifier.GetIcon());

            case <- mViewHelp.ClickedCh:
                _ = browser.OpenURL("https://github.com/NBTX/postwoman-proxy/wiki");

            case <- mSetAccessToken.ClickedCh:
                newAccessToken, success := inputbox.InputBox("Postwoman Proxy", "Please enter the new Proxy Access Token...\n(Leave this blank to disable access checks.)", "");
                if success {
                    libproxy.SetAccessToken(newAccessToken);

                    if(len(newAccessToken) == 0){
                        _ = notifier.Notify("Postwoman Proxy", "Proxy Access check disabled.", "**Anyone can access your proxy server!** The Proxy Access Token check has been disabled.", notifier.GetIcon());
                    }else{
                        _ = notifier.Notify("Postwoman Proxy", "Proxy Access Token updated...", "The Proxy Access Token has been updated.", notifier.GetIcon());
                    }
                }

            case <- mUpdateCheck.ClickedCh:
                // TODO: Add update check.
                _ = browser.OpenURL("https://github.com/NBTX/postwoman-proxy");

            case <- mQuit.ClickedCh:
                systray.Quit();
                return;
        }
    }
}

func onExit() {

}

func runPostwomanProxy() {
    libproxy.Initialize("postwoman", "127.0.0.1:9159", onProxyStateChange, true, nil);
}

func onProxyStateChange(status string, isListening bool){
    mStatus.SetTitle(status);

    if isListening {
        mCopyAccessToken.Enable();
    }
}