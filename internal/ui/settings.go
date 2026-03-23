package ui

import (
	"fmt"
	"image/color"
	"log/slog"
	"path/filepath"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	appassets "github.com/tidyyy/assets"
	"github.com/tidyyy/internal/config"
)

func ShowSettingsWindow(logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	a := app.NewWithID("com.tidyyy.alpha")
	a.Settings().SetTheme(newSilentPrecisionTheme())
	a.SetIcon(appassets.AppIcon)
	w := a.NewWindow("Tidyyy Configuration")
	w.SetIcon(appassets.AppIcon)
	w.Resize(fyne.NewSize(1060, 720))

	status := widget.NewLabel("Adjust settings, then save to apply on next daemon restart.")
	status.Wrapping = fyne.TextWrapWord
	status.TextStyle = fyne.TextStyle{Italic: true}

	watchDirs := widget.NewMultiLineEntry()
	watchDirs.SetPlaceHolder("One folder per line")
	watchDirs.SetText(strings.Join(cfg.WatchDirs, "\n"))
	watchDirs.SetMinRowsVisible(8)

	addFolderBtn := widget.NewButton("Add Folder", func() {
		dialog.NewFolderOpen(func(uri fyne.ListableURI, openErr error) {
			if openErr != nil {
				status.SetText("Folder picker error: " + openErr.Error())
				return
			}
			if uri == nil {
				return
			}

			selected := uri.Path()
			if selected == "" {
				return
			}

			current := parseWatchDirsInput(watchDirs.Text)
			current = append(current, selected)
			watchDirs.SetText(strings.Join(uniquePaths(current), "\n"))
		}, w).Show()
	})

	cloudEnabled := widget.NewCheck("Enable cloud naming (opt-in)", nil)
	cloudEnabled.SetChecked(cfg.CloudEnabled)

	cloudAPIKey := widget.NewPasswordEntry()
	cloudAPIKey.SetText(cfg.CloudAPIKey)
	cloudAPIKey.SetPlaceHolder("API key (from env: CLOUD_API_KEY)")

	maxNameWords := widget.NewSelect([]string{"2", "3", "4", "5"}, nil)
	maxNameWords.SetSelected(strconv.Itoa(cfg.MaxNameWords))
	if maxNameWords.Selected == "" {
		maxNameWords.SetSelected("5")
	}
	maxNameHint := widget.NewLabel("Controls slug length for generated file names.")
	maxNameHint.Wrapping = fyne.TextWrapWord

	saveSettings := func() bool {
		dirs := parseWatchDirsInput(watchDirs.Text)
		if len(dirs) == 0 {
			status.SetText("Add at least one folder to watch.")
			return false
		}

		words, convErr := strconv.Atoi(maxNameWords.Selected)
		if convErr != nil {
			status.SetText("Max name words must be a number.")
			return false
		}

		next := config.AppConfig{
			WatchDirs:    dirs,
			CloudEnabled: cloudEnabled.Checked,
			CloudAPIKey:  strings.TrimSpace(cloudAPIKey.Text),
			MaxNameWords: words,
		}

		if err := config.Save(next); err != nil {
			status.SetText("Save failed: " + err.Error())
			return false
		}

		logger.Info("settings saved", "watch_dirs", len(dirs), "cloud_enabled", next.CloudEnabled, "max_name_words", words)
		status.SetText("Saved to macOS config directory. Restart Tidyyy daemon to apply.")
		return true
	}

	saveButton := widget.NewButton("Save", func() {
		saveSettings()
	})
	saveButton.Importance = widget.MediumImportance

	saveCloseButton := widget.NewButton("Save and Close", func() {
		if saveSettings() {
			w.Close()
		}
	})
	saveCloseButton.Importance = widget.HighImportance

	watchHeading := widget.NewLabelWithStyle("WATCHED FOLDERS", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	watchHeading.TextStyle = fyne.TextStyle{Bold: true}
	watchSub := widget.NewLabel("List each folder on its own line.")
	watchSub.Wrapping = fyne.TextWrapWord

	watchSection := tonalPanel(surfaceContainerLowest, container.NewVBox(
		container.NewHBox(
			container.NewVBox(watchHeading, watchSub),
			layout.NewSpacer(),
			addFolderBtn,
		),
		watchDirs,
	))

	cloudHeading := widget.NewLabelWithStyle("AI AND ANALYSIS", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	cloudDescription := widget.NewLabel("Cloud base URL and model are configured with environment variables.")
	cloudDescription.Wrapping = fyne.TextWrapWord

	cloudSection := tonalPanel(surfaceContainerLowest, container.NewVBox(
		cloudHeading,
		cloudDescription,
		cloudEnabled,
		widget.NewLabel("Cloud API Key"),
		cloudAPIKey,
	))

	appHeading := widget.NewLabelWithStyle("APPLICATION", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	appDescription := widget.NewLabel("Set naming precision for generated slugs.")
	appDescription.Wrapping = fyne.TextWrapWord

	appSection := tonalPanel(surfaceContainerLowest, container.NewVBox(
		appHeading,
		appDescription,
		widget.NewLabel("Max Name Words"),
		maxNameWords,
		maxNameHint,
	))

	headerTitle := widget.NewRichTextFromMarkdown("# Configuration")
	headerSubtitle := widget.NewLabel("Fine-tune your local file automation engine. Settings are stored in your macOS user config directory.")
	headerSubtitle.Wrapping = fyne.TextWrapWord

	brandBlock := container.NewVBox(
		widget.NewLabelWithStyle("Tidyyy", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("THE SILENT PRECISIONIST"),
	)

	topBar := container.NewHBox(
		brandBlock,
		layout.NewSpacer(),
		saveCloseButton,
	)

	settingsGrid := container.NewGridWithColumns(2, cloudSection, appSection)

	mainContent := container.NewVBox(
		headerTitle,
		headerSubtitle,
		watchSection,
		settingsGrid,
		container.NewHBox(layout.NewSpacer(), saveButton),
		status,
	)

	sidebar := tonalPanel(surfaceContainerLow, container.NewVBox(
		widget.NewLabelWithStyle("UTILITY", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("v2.4.0"),
		layout.NewSpacer(),
		container.NewHBox(widget.NewIcon(theme.SettingsIcon()), widget.NewLabel("Settings")),
	))

	mainScroll := container.NewVScroll(mainContent)
	mainScroll.SetMinSize(fyne.NewSize(640, 520))

	contentRow := container.NewHBox(
		container.New(layout.NewGridWrapLayout(fyne.NewSize(220, 340)), sidebar),
		layout.NewSpacer(),
		tonalPanel(surfaceColor, mainScroll),
	)

	body := container.NewBorder(
		container.NewPadded(topBar),
		nil,
		nil,
		nil,
		container.NewPadded(contentRow),
	)

	root := tonalPanel(surfaceColor, body)
	w.SetContent(root)
	w.ShowAndRun()
	return nil
}

type silentPrecisionTheme struct {
	base fyne.Theme
}

func newSilentPrecisionTheme() fyne.Theme {
	return &silentPrecisionTheme{base: theme.DefaultTheme()}
}

func (t *silentPrecisionTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return surfaceColor
	case theme.ColorNameForeground:
		return onSurface
	case theme.ColorNamePrimary:
		return primaryColor
	case theme.ColorNameButton:
		return surfaceContainerHigh
	case theme.ColorNameHover:
		return surfaceContainer
	case theme.ColorNameInputBackground:
		return surfaceContainerLowest
	case theme.ColorNameInputBorder:
		return outlineVariantGhost
	case theme.ColorNameFocus:
		return primaryGhost
	case theme.ColorNameSelection:
		return primaryContainer
	case theme.ColorNamePlaceHolder:
		return onSurfaceVariant
	default:
		return t.base.Color(name, variant)
	}
}

func (t *silentPrecisionTheme) Font(style fyne.TextStyle) fyne.Resource {
	return t.base.Font(style)
}

func (t *silentPrecisionTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return t.base.Icon(name)
}

func (t *silentPrecisionTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNamePadding:
		return 14
	case theme.SizeNameText:
		return 13
	case theme.SizeNameInlineIcon:
		return 18
	default:
		return t.base.Size(name)
	}
}

func tonalPanel(bg color.Color, content fyne.CanvasObject) fyne.CanvasObject {
	rect := canvas.NewRectangle(bg)
	rect.CornerRadius = 8
	return container.NewStack(rect, container.NewPadded(content))
}

var (
	surfaceColor          = color.NRGBA{R: 0xF8, G: 0xF9, B: 0xFA, A: 0xFF}
	surfaceContainerLow   = color.NRGBA{R: 0xF1, G: 0xF4, B: 0xF6, A: 0xFF}
	surfaceContainer      = color.NRGBA{R: 0xEA, G: 0xEF, B: 0xF1, A: 0xFF}
	surfaceContainerHigh  = color.NRGBA{R: 0xE3, G: 0xE9, B: 0xEC, A: 0xFF}
	surfaceContainerLowest = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	onSurface            = color.NRGBA{R: 0x2B, G: 0x34, B: 0x37, A: 0xFF}
	onSurfaceVariant     = color.NRGBA{R: 0x58, G: 0x60, B: 0x64, A: 0xFF}
	primaryColor         = color.NRGBA{R: 0x44, G: 0x62, B: 0x73, A: 0xFF}
	primaryContainer     = color.NRGBA{R: 0xC7, G: 0xE7, B: 0xFA, A: 0xFF}
	outlineVariantGhost  = color.NRGBA{R: 0xAB, G: 0xB3, B: 0xB7, A: 0x26}
	primaryGhost         = color.NRGBA{R: 0x44, G: 0x62, B: 0x73, A: 0x66}
)

func parseWatchDirsInput(text string) []string {
	raw := strings.ReplaceAll(text, ",", "\n")
	parts := strings.Split(raw, "\n")
	dirs := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		abs, err := filepath.Abs(p)
		if err != nil {
			continue
		}
		dirs = append(dirs, abs)
	}
	return uniquePaths(dirs)
}

func uniquePaths(paths []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}
