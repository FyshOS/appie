package appie

import (
	"bytes"
	_ "image/jpeg" // support JPEG images
	"image/png"    // PNG support is required as we use it directly
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jackmordaunt/icns"
	"howett.net/plist"

	"fyne.io/fyne/v2"
)

type macOSAppBundle struct {
	DisplayName string `plist:"CFBundleDisplayName"`
	// TODO alternateNames []string
	Executable string `plist:"CFBundleExecutable"`

	categories []string
	runPath    string
	IconFile   string `plist:"CFBundleIconFile"`
	iconPath   string

	iconCache fyne.Resource
}

func (m *macOSAppBundle) Name() string {
	return m.DisplayName
}

func (m *macOSAppBundle) Categories() []string {
	return m.categories
}

func (m *macOSAppBundle) Hidden() bool {
	return false
}

func (m *macOSAppBundle) Icon(_ string, _ int) fyne.Resource {
	if m.iconCache != nil {
		return m.iconCache
	}

	src, err := os.Open(m.iconPath)
	if err != nil {
		fyne.LogError("Failed to read icon data for "+m.iconPath, err)
		return nil
	}

	icon, err := icns.Decode(src)
	if err != nil {
		fyne.LogError("Failed to parse icon data for "+m.iconPath, err)
		return nil
	}

	var data bytes.Buffer
	err = png.Encode(&data, icon)
	if err != nil {
		fyne.LogError("Failed to encode icon data for "+m.iconPath, err)
		return nil
	}

	iconName := filepath.Base(m.iconPath)
	m.iconCache = fyne.NewStaticResource(strings.Replace(iconName, ".icns", ".png", 1), data.Bytes())
	return m.iconCache
}

func (m *macOSAppBundle) MimeTypes() []string {
	// TODO actually find out what mime types are associated with this app
	// Possibly through parsing UTIs etc, could be mdls command
	// might need to explore _UTCopyDeclaredTypeIdentifiers private API
	// then per-app we can iterate the associated UTIs and access tags
	// using key "public.mime-type" to get a NSArray<NSString *> of mime types
	return []string{}
}

func (m *macOSAppBundle) Run(env []string) error {
	return m.RunWithParameters([]string{}, env)
}

func (m *macOSAppBundle) RunWithParameters(params, _ []string) error {
	// in macOS test mode we ignore the wm env flags
	if len(params) == 0 {
		return exec.Command("open", "-a", m.runPath).Start()
	}

	return exec.Command("open", "-a", m.runPath, params[0]).Start()
}

func (m *macOSAppBundle) Source() *AppSource {
	return nil
}

func loadAppBundle(name, path, category string) AppData {
	buf, err := os.Open(filepath.Join(path, "Contents", "Info.plist"))
	if err != nil {
		fyne.LogError("Unable to read application plist", err)
		return nil
	}

	data := macOSAppBundle{DisplayName: name, categories: []string{category}}
	decoder := plist.NewDecoder(buf)
	err = decoder.Decode(&data)
	if err != nil {
		fyne.LogError("Unable to parse application plist", err)
		return nil
	}
	data.runPath = filepath.Join(path, "Contents", "MacOS", data.Executable)

	data.iconPath = filepath.Join(path, "Contents", "Resources", data.IconFile)
	pos := strings.Index(data.iconPath, ".icns")
	if pos == -1 {
		data.iconPath = data.iconPath + ".icns"
	}
	return &data
}

type macOSAppProvider struct {
	rootDirs []string
	cache    *appCache
}

func (m *macOSAppProvider) forEachApplication(f func(name, path, category string) bool) {
	for _, root := range m.rootDirs {
		category := filepath.Base(root)
		files, err := os.ReadDir(root)
		if err != nil {
			fyne.LogError("Could not read applications directory "+root, err)
			return
		}
		for _, file := range files {
			if !file.IsDir() || !strings.HasSuffix(file.Name(), ".app") {
				continue // skip non-app bundles
			}
			appDir := filepath.Join(root, file.Name())
			if f(file.Name()[0:len(file.Name())-4], appDir, category) {
				break
			}
		}
	}
}

func (m *macOSAppProvider) AvailableApps() []AppData {
	var icons []AppData
	m.forEachApplication(func(name, path, category string) bool {
		app := loadAppBundle(name, path, category)
		if app != nil {
			icons = append(icons, app)
		}
		return false
	})
	return icons
}

func (m *macOSAppProvider) AvailableThemes() []string {
	//I'm not sure this is relevant on Mac OSX
	return []string{}
}

func (m *macOSAppProvider) FindAppFromName(appName string) AppData {
	var icon AppData
	m.cache.forEachCachedApplication(func(name string, app AppData) bool {
		if name == appName {
			icon = app
			return true
		}

		return false
	})

	return icon
}

func (m *macOSAppProvider) DefaultApps() []AppData {
	var apps []AppData

	apps = appendAppIfExists(apps, findOneAppFromNames(m, "Terminal", "iTerm"))
	apps = appendAppIfExists(apps, findOneAppFromNames(m, "Google Chrome", "Firefox", "Safari"))
	apps = appendAppIfExists(apps, findOneAppFromNames(m, "Spark", "AirMail", "Mail"))
	apps = appendAppIfExists(apps, m.FindAppFromName("Photos"))
	apps = appendAppIfExists(apps, m.FindAppFromName("System Preferences"))

	return apps
}

func (m *macOSAppProvider) FindAppsMatching(pattern string) []AppData {
	var icons []AppData
	m.cache.forEachCachedApplication(func(name string, app AppData) bool {
		if !strings.Contains(strings.ToLower(name), strings.ToLower(pattern)) {
			return false
		}

		icons = append(icons, app)
		return false
	})

	return icons
}

func (m *macOSAppProvider) CategorizedApps() map[string][]AppData {
	var allApps, allUtils []AppData
	m.cache.forEachCachedApplication(func(_ string, app AppData) bool {
		if app.Categories()[0] == "Applications" {
			allApps = append(allApps, app)
		} else {
			allUtils = append(allUtils, app)
		}
		return false
	})

	return map[string][]AppData{
		"Applications": allApps,
		"Utilities":    allUtils,
	}
}

// NewMacOSProvider creates an instance of a Provider that can find and decode macOS apps
func NewMacOSProvider() Provider {
	source := &macOSAppProvider{rootDirs: []string{"/Applications", "/Applications/Utilities",
		"/System/Applications", "/System/Applications/Utilities"}}
	source.cache = newAppCache(source)
	return source
}
