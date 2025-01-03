package appie

import (
	"runtime"

	"fyne.io/fyne/v2"
)

// AppData is an interface for accessing information about application icons
type AppData interface {
	Name() string                               // Name is the name of the app usually
	Run([]string) error                         // Run is the command to run the app, passing any environment variables to be set
	RunWithParameters([]string, []string) error // RunWithParameters is the command to run the app, passing command line parameters and setting any specified environment variables

	Categories() []string                      // Categories is a list of categories that the app fits in (platform specific)
	Hidden() bool                              // Hidden specifies whether instances of this app should be hidden
	Icon(theme string, size int) fyne.Resource // Icon returns an icon for the app in the requested theme and size
	MimeTypes() []string                       // MimeTypes returns a list of mimetypes that this application can handle

	Source() *AppSource // Source will return the location of the app source code from metadata, if known
}

// AppSource represents the source code information of an application
type AppSource struct {
	Repo, Dir string
}

// Provider describes a type that can locate icons and applications for the current system
type Provider interface {
	AvailableApps() []AppData
	AvailableThemes() []string
	FindAppFromName(appName string) AppData
	FindAppsMatching(pattern string) []AppData
	DefaultApps() []AppData
	CategorizedApps() map[string][]AppData
}

// SystemProvider returns an application provider for the current system.
// for macOS systems it will be a macOSProvider, for Linux/Unix it will be an FDOProvider.
func SystemProvider() Provider {
	switch runtime.GOOS {
	case "darwin":
		return NewMacOSProvider()
	case "linux", "freebsd", "openbsd", "netbsd", "dragonfly":
		return NewFDOProvider()
	}

	return nil
}
