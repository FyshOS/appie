package appie

import (
	"bufio"
	"io/fs"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"

	_ "github.com/fyne-io/image/xpm" // load XPM icons to supported image format
)

var (
	iconExtensions      = []string{".png", ".svg", ".xpm"}
	fallbackCategory    = "Other"
	supportedCategories = []string{
		"AudioVideo", "Development", "Education", "Game", "Graphics", "Network",
		"Office", "Science", "Settings", "System", "Utility"}
)

// fdoApplicationData is a structure that contains information about .desktop files
type fdoApplicationData struct {
	name     string // Application name
	iconName string // Icon name
	iconPath string // Icon path
	exec     string // Command to execute application

	categories, mime []string
	hide             bool
	iconCache        fyne.Resource

	source *AppSource
}

// Name returns the name associated with an fdo app
func (data *fdoApplicationData) Name() string {
	return data.name
}

// Categories returns a list of the categories this icon has configured
func (data *fdoApplicationData) Categories() []string {
	return data.categories
}

func (data *fdoApplicationData) Hidden() bool {
	return data.hide
}

// IconName returns the name of the icon that an fdo app wishes to use
func (data *fdoApplicationData) IconName() string {
	return data.iconName
}

// Icon returns the path of the icon that an fdo app wishes to use
func (data *fdoApplicationData) Icon(theme string, size int) fyne.Resource {
	if data.iconCache != nil {
		return data.iconCache
	}

	path := data.iconPath
	if path == "" {
		path = FdoLookupIconPath(theme, size, data.iconName)
		if path == "" {
			return nil
		}
	}

	data.iconCache = loadIcon(path)
	return data.iconCache
}

func (data *fdoApplicationData) MimeTypes() []string {
	return data.mime
}

func (data *fdoApplicationData) Source() *AppSource {
	return data.source
}

// extractArgs sanitises argument parameters from an Exec configuration
func extractArgs(args, params []string) []string {
	paramIndex := 0
	var ret []string
	for _, arg := range args {
		if len(arg) >= 2 && arg[0] == '%' {
			switch arg[1] {
			case 'u', 'f': // url, file
				if paramIndex < len(params) {
					ret = append(ret, params[paramIndex])
					paramIndex++
				}
			case 'U', 'F': // url list, file list
				if paramIndex < len(params) {
					ret = append(ret, params[paramIndex:]...)
					paramIndex = len(params)
				}
			}
			continue
		}
		ret = append(ret, arg)
	}

	return ret
}

// Run executes the command for this fdo app.
// It also passes in the specified environment variables
func (data *fdoApplicationData) Run(env []string) error {
	return data.RunWithParameters([]string{}, env)
}

// RunWithParameters executes the command for this fdo app.
// It passes any parameters specified and sets up the listed environment.
func (data *fdoApplicationData) RunWithParameters(params, env []string) error {
	vars := os.Environ()
	vars = append(vars, env...)

	commands := strings.Split(data.exec, " ")
	command := commands[0]
	if command[0] == '"' {
		command = command[1 : len(command)-1]
	}

	cmd := exec.Command(command)
	if len(commands) > 1 {
		cmd.Args = extractArgs(commands, params) // Args[0] should be binary path
	}

	cmd.Env = vars
	return cmd.Start()
}

func (data *fdoApplicationData) mainCategory() string {
	if len(data.Categories()) == 0 {
		return fallbackCategory
	}

	// check that one of the app categories is supported
	for _, cat := range data.Categories() {
		for _, each := range supportedCategories {
			if each == cat {
				return cat
			}
		}
	}

	return fallbackCategory
}

func loadIcon(path string) fyne.Resource {
	data, err := os.ReadFile(path)
	if err != nil {
		fyne.LogError("Failed to load image", err)
		return nil
	}

	return fyne.NewStaticResource(filepath.Base(path), data)
}

// fdoLookupXdgDataDirs returns a string slice of all XDG_DATA_DIRS
func fdoLookupXdgDataDirs() []string {
	dataLocation := os.Getenv("XDG_DATA_DIRS")
	locationLookup := strings.Split(dataLocation, ":")
	if len(locationLookup) == 0 || (len(locationLookup) == 1 && locationLookup[0] == "") {
		var fallbackLocations []string
		homeDir, err := os.UserHomeDir()
		if err == nil {
			fallbackLocations = append(fallbackLocations, filepath.Join(homeDir, ".local/share"))
		}
		fallbackLocations = append(fallbackLocations, "/usr/local/share")
		fallbackLocations = append(fallbackLocations, "/usr/share")
		return fallbackLocations
	}
	return locationLookup
}

func fdoForEachApplicationFile(f func(data AppData) bool) {
	locationLookup := fdoLookupXdgDataDirs()
	for _, dataDir := range locationLookup {
		testLocation := filepath.Join(dataDir, "applications")
		files, err := os.ReadDir(testLocation)
		if err != nil {
			continue
		}
		for _, file := range files {
			if strings.HasPrefix(file.Name(), ".") || file.IsDir() || !strings.HasSuffix(file.Name(), ".desktop") {
				continue
			}

			icon := newFdoIconData(filepath.Join(testLocation, file.Name()))
			if icon == nil {
				continue
			}

			if f(icon) {
				return
			}
		}
	}
}

// lookupApplicationByMetadata looks up an application by comparing the requested name to the contents of .desktop files
func (f *fdoIconProvider) lookupApplicationByMetadata(appName string) AppData {
	var returnIcon AppData
	f.cache.forEachCachedApplication(func(_ string, icon AppData) bool {
		if icon.(*fdoApplicationData).name == appName || icon.(*fdoApplicationData).exec == appName {
			returnIcon = icon
			return true
		}
		return false
	})
	return returnIcon
}

// lookupApplication looks up an application by name and returns an fdoApplicationData struct
func (f *fdoIconProvider) lookupApplication(appName string) AppData {
	if appName == "" {
		return nil
	}

	var found AppData
	f.cache.forEachCachedApplication(func(name string, icon AppData) bool {
		if name == appName {
			found = icon
			return true
		}

		return false
	})
	if found != nil {
		return found
	}

	//If no match was found checking by filenames, check by file contents
	return f.lookupApplicationByMetadata(appName)
}

func fdoClosestSizeIcon(files []fs.DirEntry, iconSize int, format string, baseDir string, joiner string, iconName string) string {
	var sizes []int
	for _, f := range files {
		if format == "32x32" {
			size, err := strconv.Atoi(strings.Split(f.Name(), "x")[0])
			if err != nil {
				continue
			}
			sizes = append(sizes, size)
		} else {
			size, err := strconv.Atoi(f.Name())
			if err != nil {
				continue
			}
			sizes = append(sizes, size)
		}
	}

	if len(sizes) == 0 {
		return ""
	}

	var closestMatch string
	var difference int
	for _, size := range sizes {
		sizeDir := strconv.Itoa(size)
		if format == "32x32" {
			sizeDir = sizeDir + "x" + sizeDir
		}
		matchDir := ""
		if joiner != "" {
			matchDir = filepath.Join(baseDir, sizeDir, joiner)
		} else {
			matchDir = filepath.Join(baseDir, sizeDir)
		}
		diff := int(math.Abs(float64(iconSize - size)))
		testIcon := ""
		for _, extension := range iconExtensions {
			testIcon = filepath.Join(matchDir, iconName+extension)
			if _, err := os.Stat(testIcon); os.IsNotExist(err) {
				testIcon = ""
			} else {
				break
			}
		}
		if closestMatch == "" && testIcon != "" {
			closestMatch = testIcon
			difference = diff
			continue
		}
		if diff < difference && testIcon != "" {
			closestMatch = testIcon
			difference = diff
		}
	}
	return closestMatch
}

func lookupAnyIconSizeInThemeDir(dir string, joiner string, iconName string, iconSize int) string {
	files, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	// Example is /usr/share/icons/icon_theme/<size>/<joiner>/xterm.png
	closestMatch := fdoClosestSizeIcon(files, iconSize, "32x32", dir, joiner, iconName)
	if closestMatch == "" {
		closestMatch = fdoClosestSizeIcon(files, iconSize, "32", dir, joiner, iconName)
	}
	if closestMatch != "" {
		return closestMatch
	}

	directory := filepath.Join(dir, joiner)
	files, err = os.ReadDir(directory)
	if err != nil {
		return ""
	}

	// Example is /usr/share/icons/icon_theme/<joiner>/<size>/xterm.png
	closestMatch = fdoClosestSizeIcon(files, iconSize, "32x32", directory, "", iconName)
	if closestMatch == "" {
		closestMatch = fdoClosestSizeIcon(files, iconSize, "32", directory, "", iconName)
	}
	return closestMatch
}

// FdoLookupIconPathInTheme searches icon locations to find a match using a provided theme directory.
// The dir path is the root of the theme, parentDir is used to look up inherited themes
func FdoLookupIconPathInTheme(iconSize string, dir string, parentDir string, iconName string) string {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return ""
	}
	for _, extension := range iconExtensions {
		// Example is /usr/share/icons/icon_theme/32/apps/xterm.png
		testIcon := filepath.Join(dir, iconSize, "apps", iconName+extension)
		if _, err := os.Stat(testIcon); err == nil {
			return testIcon
		}
		// Example is /usr/share/icons/icon_theme/32x32/apps/xterm.png
		testIcon = filepath.Join(dir, iconSize+"x"+iconSize, "apps", iconName+extension)
		if _, err := os.Stat(testIcon); err == nil {
			return testIcon
		}

		// Example is /usr/share/icons/icon_theme/apps/32/xterm.png
		testIcon = filepath.Join(dir, "apps", iconSize, iconName+extension)
		if _, err := os.Stat(testIcon); err == nil {
			return testIcon
		}
		// Example is /usr/share/icons/icon_theme/apps/32x32/xterm.png
		testIcon = filepath.Join(dir, "apps", iconSize+"x"+iconSize, iconName+extension)
		if _, err := os.Stat(testIcon); err == nil {
			return testIcon
		}
	}
	// If the requested icon wasn't found in the specific size or scalable dirs
	// of the theme try all sizes within theme and all icon dirs besides apps
	var subIconDirs = []string{"apps", "actions", "devices", "emblems", "legacy", "mimetypes", "places", "status"}
	iconSizeInt, err := strconv.Atoi(iconSize)
	if err != nil {
		iconSizeInt = 32
	}
	for _, joiner := range subIconDirs {
		testIcon := lookupAnyIconSizeInThemeDir(dir, joiner, iconName, iconSizeInt)
		if testIcon != "" {
			return testIcon
		}
	}

	//Lets check inherited themes for the match
	indexTheme := filepath.Join(dir, "index.theme")
	if _, err := os.Stat(indexTheme); !os.IsNotExist(err) {
		file, err := os.Open(indexTheme)
		if err == nil {
			defer file.Close()

			var inheritedThemes []string
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "Inherits=") {
					inherits := strings.SplitAfter(line, "=")
					inheritedThemes = strings.SplitN(inherits[1], ",", -1)
					break
				}
			}
			if len(inheritedThemes) > 0 {
				for _, theme := range inheritedThemes {
					childDir := filepath.Join(parentDir, "icons", theme)
					iconPath := FdoLookupIconPathInTheme(iconSize, childDir, parentDir, iconName)
					if iconPath != "" {
						return iconPath
					}
				}
			}
		}
	}

	// Try this as a last resort
	for _, extension := range iconExtensions {
		// Example is /usr/share/icons/icon_theme/scalable/apps/xterm.png
		testIcon := filepath.Join(dir, "scalable", "apps", iconName+extension)
		if _, err := os.Stat(testIcon); err == nil {
			return testIcon
		}
		// Example is /usr/share/icons/icon_theme/apps/scalable/xterm.png
		testIcon = filepath.Join(dir, "apps", "scalable", iconName+extension)
		if _, err := os.Stat(testIcon); err == nil {
			return testIcon
		}
	}
	return ""
}

// FdoLookupIconPath will take the name of an icon and find a matching image file
func FdoLookupIconPath(theme string, size int, iconName string) string {
	locationLookup := fdoLookupXdgDataDirs()
	iconTheme := theme
	iconSize := strconv.Itoa(size)
	for _, dataDir := range locationLookup {
		//Example is /usr/share/icons/icon_theme
		dir := filepath.Join(dataDir, "icons", iconTheme)
		iconPath := FdoLookupIconPathInTheme(iconSize, dir, dataDir, iconName)
		if iconPath != "" {
			return iconPath
		}
	}
	for _, dataDir := range locationLookup {
		//Hicolor is the default fallback theme - Example /usr/share/icons/icon_theme/hicolor
		dir := filepath.Join(dataDir, "icons", "hicolor")
		iconPath := FdoLookupIconPathInTheme(iconSize, dir, dataDir, iconName)
		if iconPath != "" {
			return iconPath
		}
	}
	for _, dataDir := range locationLookup {
		//Icons may be in the pixmaps directory - test before we do our final fallback
		for _, extension := range iconExtensions {
			iconPath := filepath.Join(dataDir, "pixmaps", iconName+extension)
			if _, err := os.Stat(iconPath); err == nil {
				return iconPath
			}
		}
	}
	//No Icon Was Found
	return ""
}

func fdoLookupAvailableThemes() []string {
	var themes []string
	locationLookup := fdoLookupXdgDataDirs()
	for _, dataDir := range locationLookup {
		files, err := os.ReadDir(filepath.Join(dataDir, "icons"))
		if err != nil {
			continue
		}
		//Enter icon theme
		for _, f := range files {
			if strings.HasPrefix(f.Name(), ".") || !f.IsDir() {
				continue
			}
			//Example is /usr/share/icons/gnome
			themes = append(themes, f.Name())
		}
	}
	return themes
}

// newFdoIconData creates and returns a struct that contains needed fields from a .desktop file
func newFdoIconData(desktopPath string) AppData {
	file, err := os.Open(desktopPath)
	if err != nil {
		fyne.LogError("Could not open file", err)
		return nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	fdoApp := fdoApplicationData{name: "", iconName: "", iconPath: "", exec: ""}
	var currentSection string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "[") {
			currentSection = line
		}
		switch currentSection {
		case "[X-Fyne Source]":
			if fdoApp.source == nil {
				fdoApp.source = &AppSource{}
			}
			if strings.HasPrefix(line, "Repo=") {
				name := strings.SplitAfter(line, "=")
				fdoApp.source.Repo = name[1]
			}
			if strings.HasPrefix(line, "Dir=") {
				name := strings.SplitAfter(line, "=")
				fdoApp.source.Dir = name[1]
			}
		case "[Desktop Entry]": // fall through to code below
		default:
			continue
		}
		if strings.HasPrefix(line, "Name=") {
			name := strings.SplitAfter(line, "=")
			fdoApp.name = name[1]
		} else if strings.HasPrefix(line, "Icon=") {
			icon := strings.SplitAfter(line, "=")
			fdoApp.iconName = icon[1]
			if _, err := os.Stat(icon[1]); err == nil {
				fdoApp.iconPath = icon[1]
			}
		} else if strings.HasPrefix(line, "Exec=") {
			exec := strings.SplitAfter(line, "=")
			fdoApp.exec = exec[1]
		} else if strings.HasPrefix(line, "Categories=") {
			cats := strings.SplitAfter(line, "=")
			fdoApp.categories = strings.Split(cats[1], ";")
		} else if strings.HasPrefix(line, "MimeType=") {
			mimes := strings.SplitAfter(line, "=")
			fdoApp.mime = strings.Split(mimes[1], ";")
		} else if strings.HasPrefix(line, "NoDisplay=") {
			val := strings.Split(line, "=")
			if strings.TrimSpace(val[1]) == "true" {
				fdoApp.hide = true
			}
		}
	}
	if err := scanner.Err(); err != nil {
		fyne.LogError("Could not read file", err)
		return nil
	}
	return &fdoApp
}

type fdoIconProvider struct {
	cache *appCache
}

// AvailableApps returns all of the available applications in a AppData slice
func (f *fdoIconProvider) AvailableApps() []AppData {
	var icons []AppData
	fdoForEachApplicationFile(func(icon AppData) bool {
		if icon == nil {
			return false
		}
		icons = append(icons, icon)
		return false
	})
	return icons
}

// AvailableThemes returns all available icon themes in a string slice
func (f *fdoIconProvider) AvailableThemes() []string {
	return fdoLookupAvailableThemes()
}

// FindAppFromName matches an icon name to a location and returns an AppData interface
func (f *fdoIconProvider) FindAppFromName(appName string) AppData {
	return f.lookupApplication(appName)
}

// FindAppsMatching returns a list of icons that match a partial name of an app and returns an AppData slice
func (f *fdoIconProvider) FindAppsMatching(appName string) []AppData {
	var icons []AppData
	f.cache.forEachCachedApplication(func(_ string, icon AppData) bool {
		if icon == nil {
			return false
		}
		if strings.Contains(strings.ToLower(icon.(*fdoApplicationData).name), strings.ToLower(appName)) ||
			strings.Contains(strings.ToLower(icon.(*fdoApplicationData).exec), strings.ToLower(appName)) {
			icons = append(icons, icon)
		}

		return false
	})

	return icons
}

func findOneAppFromNames(f Provider, names ...string) AppData {
	for _, name := range names {
		app := f.FindAppFromName(name)
		if app != nil {
			return app
		}
	}

	return nil
}

func appendAppIfExists(apps []AppData, app AppData) []AppData {
	if app == nil {
		return apps
	}

	return append(apps, app)
}

func (f *fdoIconProvider) DefaultApps() []AppData {
	var apps []AppData

	apps = appendAppIfExists(apps, findOneAppFromNames(f, "fyneterm", "xfce4-terminal", "gnome-terminal", "org.kde.konsole", "xterm"))
	apps = appendAppIfExists(apps, findOneAppFromNames(f, "chromium", "google-chrome", "firefox"))
	apps = appendAppIfExists(apps, findOneAppFromNames(f, "sylpheed", "thunderbird", "evolution"))
	apps = appendAppIfExists(apps, f.FindAppFromName("gimp"))

	return apps
}

func (f *fdoIconProvider) CategorizedApps() map[string][]AppData {
	cats := map[string][]AppData{}

	f.cache.forEachCachedApplication(func(_ string, app AppData) bool {
		cat := app.(*fdoApplicationData).mainCategory()
		var list []AppData
		if c, ok := cats[cat]; ok {
			list = c
		}
		list = append(list, app)
		cats[cat] = list
		return false
	})
	return cats
}

// NewFDOProvider returns a new application provider following the FreeDesktop.org specifications
func NewFDOProvider() Provider {
	source := &fdoIconProvider{}
	source.cache = newAppCache(source)
	return source
}
