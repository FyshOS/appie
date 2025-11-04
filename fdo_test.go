package appie

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"fyne.io/fyne/v2"
)

var (
	iconTheme = "default_theme"
	iconSize  = 32
)

func exists(data AppData) bool {
	return data != nil && data.Icon(iconTheme, iconSize) != nil
}

func setTestEnv(t *testing.T) {
	workingDir, err := os.Getwd()
	if err != nil {
		fyne.LogError("Could not get current working directory", err)
		t.FailNow()
	}
	err = os.Setenv("XDG_DATA_DIRS", filepath.Join(workingDir, "testdata"))
	if err != nil {
		fyne.LogError("Could not set test environment variable", err)
		t.FailNow()
	}
}

// applications/app1.desktop and icons/default_theme/apps/32x32/app1.png
func TestFdoLookupDefaultTheme(t *testing.T) {
	setTestEnv(t)
	data := NewFDOProvider().(*fdoIconProvider).lookupApplication("app1")
	assert.Equal(t, true, exists(data))
}

// applications/com.fyne.app.desktop and icons/default_theme/apps/scalable/app2.svg
func TestFdoFileNameMisMatchAndScalable(t *testing.T) {
	setTestEnv(t)
	data := NewFDOProvider().(*fdoIconProvider).lookupApplication("app2")
	assert.Equal(t, true, exists(data))
}

// check the category from app1
func TestFdoIconCategory(t *testing.T) {
	setTestEnv(t)
	data := NewFDOProvider().(*fdoIconProvider).lookupApplication("app1")
	assert.Equal(t, 2, len(data.Categories()))
	assert.Equal(t, "App1", data.Categories()[0])

	// also check the first supported one
	assert.Equal(t, "Utility", data.(*fdoApplicationData).mainCategory())
}

// applications/app3.desktop and applications/app3.png
func TestFdoIconNameIsPath(t *testing.T) {
	setTestEnv(t)
	dataLocation := os.Getenv("XDG_DATA_DIRS")
	output := fmt.Sprintf("[Desktop Entry]\nName=App3\nExec=app3\nIcon=%s\n", filepath.Join(dataLocation, "icons", "app3.png"))
	err := os.WriteFile(filepath.Join(dataLocation, "applications", "app3.desktop"), []byte(output), 0o644)
	if err != nil {
		fyne.LogError("Could not create desktop for Icon Name path example", err)
		t.FailNow()
	}
	data := NewFDOProvider().(*fdoIconProvider).lookupApplication("app3")
	assert.Equal(t, true, exists(data))
}

// check NoDisplay from app4
func TestFdoIconHide(t *testing.T) {
	setTestEnv(t)
	data := NewFDOProvider().(*fdoIconProvider).lookupApplication("app4")
	assert.Equal(t, true, data.Hidden())
}

// applications/app4.desktop and pixmaps/app4.png
func TestFdoIconInPixmaps(t *testing.T) {
	setTestEnv(t)
	data := NewFDOProvider().(*fdoIconProvider).lookupApplication("app4")
	assert.Equal(t, true, exists(data))
}

// applications/app5.desktop and icons/hicolor/32x32/apps/app5.png
func TestFdoIconHicolorFallback(t *testing.T) {
	setTestEnv(t)
	data := NewFDOProvider().(*fdoIconProvider).lookupApplication("app5")
	assert.Equal(t, true, exists(data))
}

// applications/app6.desktop and icons/hicolor/scalable/apps/app6.svg
func TestFdoIconHicolorFallbackScalable(t *testing.T) {
	setTestEnv(t)
	data := NewFDOProvider().(*fdoIconProvider).lookupApplication("app6")
	assert.Equal(t, true, exists(data))
}

// applications/com.fyne.app.desktop
func TestFdoIconSource(t *testing.T) {
	setTestEnv(t)
	data := NewFDOProvider().(*fdoIconProvider).lookupApplication("app2")

	assert.NotNil(t, data.Source())
	assert.Equal(t, "https://example.com/repo", data.Source().Repo)
	assert.Equal(t, "cmd/dir", data.Source().Dir)
}

// applications/app7.desktop and icons/default_theme/apps/16x16/app7.png
func TestFdoLookupDefaultThemeDifferentSize(t *testing.T) {
	setTestEnv(t)
	data := NewFDOProvider().(*fdoIconProvider).lookupApplication("app7")
	assert.Equal(t, true, exists(data))
}

// applications/app8.desktop and icons/third_theme/apps/32/app8.png
func TestFdoLookupAnyThemeFallback(t *testing.T) {
	setTestEnv(t)
	data := NewFDOProvider().(*fdoIconProvider).lookupApplication("app8")
	assert.Equal(t, true, exists(data))
}

// applications/xterm.desktop and icons/third_theme/emblems/16x16/app9.png
func TestFdoLookupIconNotInApps(t *testing.T) {
	setTestEnv(t)
	data := NewFDOProvider().(*fdoIconProvider).lookupApplication("xterm")
	assert.Equal(t, true, exists(data))
}

func TestFdoLookupPartialMatches(t *testing.T) {
	setTestEnv(t)
	dataMatches := NewFDOProvider().FindAppsMatching("app")
	assert.Equal(t, true, len(dataMatches) > 1)
	for _, data := range dataMatches {
		assert.Equal(t, true, exists(data))
	}
}

func TestFdoIconProvider_findOneAppFromNames(t *testing.T) {
	setTestEnv(t)
	single := findOneAppFromNames(NewFDOProvider(), "missing", "app1", "xterm")
	assert.NotNil(t, single)
	assert.Equal(t, "App1", single.Name())
}

func TestFdoIconProvider_DefaultApps(t *testing.T) {
	setTestEnv(t)
	defaults := NewFDOProvider().DefaultApps()
	assert.True(t, len(defaults) > 0)
}

func TestFdoExtractArgs(t *testing.T) {
	params := []string{"-u", "thing", "%u"}

	extracted := extractArgs(params, []string{"https://example.com"})

	assert.Equal(t, []string{"-u", "thing", "https://example.com"}, extracted)
}
