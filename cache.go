package appie

type appCache struct {
	source  ApplicationProvider
	appList []AppData
}

func (c *appCache) forEachCachedApplication(f func(string, AppData) bool) {
	if c.appList == nil {
		c.appList = c.source.AvailableApps()
	}

	for _, a := range c.appList {
		if f(a.Name(), a) {
			return
		}
	}
}

func newAppCache(c ApplicationProvider) *appCache {
	return &appCache{source: c}
}
