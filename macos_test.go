package appie

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadAppBundle(t *testing.T) {
	path := "testdata/Test.app"
	app := loadAppBundle("Test", path, "Applications")

	assert.NotNil(t, app)
	assert.Equal(t, "Test", app.Name())
	assert.NotNil(t, app.Icon("", 0))
}

func TestMacOSAppProvider_FindAppFromName(t *testing.T) {
	provider := NewMacOSProvider()
	provider.(*macOSAppProvider).rootDirs = []string{"testdata"}

	app := provider.FindAppFromName("Test")
	assert.NotNil(t, app)
	assert.Equal(t, "Test", app.Name())
}
