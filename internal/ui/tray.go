package ui

import (
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/driver/desktop"
	appassets "github.com/tidyyy/assets"
)

type TrayLifecycleHooks struct {
	IsDaemonRunning func() bool
	IsPaused        func() bool
	SetPaused       func(bool) error
	RestartDaemon   func() error
	Quit            func() error
}

// RunTrayRuntime starts the desktop tray app and keeps the daemon alive when settings window closes.
func RunTrayRuntime(logger *slog.Logger, settingsHooks *SettingsLifecycleHooks, trayHooks *TrayLifecycleHooks) error {
	if logger == nil {
		logger = slog.Default()
	}

	a := app.NewWithID("com.tidyyy.alpha")
	a.Settings().SetTheme(newSilentPrecisionTheme())
	a.SetIcon(appassets.AppIcon)

	w, err := NewSettingsWindow(a, logger, settingsHooks)
	if err != nil {
		return err
	}

	w.SetCloseIntercept(func() {
		w.Hide()
	})

	if da, ok := a.(desktop.App); ok {
		da.SetSystemTrayIcon(appassets.AppIcon)

		var refreshTrayMenu func()
		refreshTrayMenu = func() {
			pauseLabel := "Pause Daemon"
			if trayHooks != nil && trayHooks.IsPaused != nil && trayHooks.IsPaused() {
				pauseLabel = "Resume Daemon"
			}

			statusLabel := "Daemon: Stopped"
			if trayHooks != nil && trayHooks.IsDaemonRunning != nil && trayHooks.IsDaemonRunning() {
				if trayHooks.IsPaused != nil && trayHooks.IsPaused() {
					statusLabel = "Daemon: Paused"
				} else {
					statusLabel = "Daemon: Running"
				}
			}

			statusItem := fyne.NewMenuItem(statusLabel, nil)
			statusItem.Disabled = true

			pauseResumeItem := fyne.NewMenuItem(pauseLabel, func() {
				if trayHooks == nil || trayHooks.SetPaused == nil || trayHooks.IsPaused == nil {
					return
				}
				nextPaused := !trayHooks.IsPaused()
				if err := trayHooks.SetPaused(nextPaused); err != nil {
					logger.Error("failed to toggle pause", "err", err)
				}
				refreshTrayMenu()
			})

			restartItem := fyne.NewMenuItem("Restart Daemon", func() {
				if trayHooks == nil || trayHooks.RestartDaemon == nil {
					return
				}
				if err := trayHooks.RestartDaemon(); err != nil {
					logger.Error("daemon restart failed", "err", err)
				}
				refreshTrayMenu()
			})

			openSettingsItem := fyne.NewMenuItem("Open Settings", func() {
				w.Show()
				w.RequestFocus()
			})

			quitItem := fyne.NewMenuItem("Quit Tidyyy", func() {
				if trayHooks != nil && trayHooks.Quit != nil {
					if err := trayHooks.Quit(); err != nil {
						logger.Error("quit hook failed", "err", err)
					}
				}
				a.Quit()
			})

			da.SetSystemTrayMenu(fyne.NewMenu(
				"Tidyyy",
				statusItem,
				fyne.NewMenuItemSeparator(),
				pauseResumeItem,
				restartItem,
				openSettingsItem,
				fyne.NewMenuItemSeparator(),
				quitItem,
			))
		}

		refreshTrayMenu()
	} else {
		logger.Warn("system tray not supported by current fyne driver")
	}

	w.Show()
	a.Run()
	return nil
}

func ValidateTrayHooks(hooks *TrayLifecycleHooks) error {
	if hooks == nil {
		return fmt.Errorf("tray hooks are nil")
	}
	if hooks.Quit == nil {
		return fmt.Errorf("tray quit hook is required")
	}
	return nil
}
