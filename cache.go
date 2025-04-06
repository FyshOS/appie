package appie

type appCache struct {
	source  Provider
	appList []AppData
}

func (c *appCache) clearCache() {
	c.appList = nil
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

func newAppCache(c Provider) *appCache {
	return &appCache{source: c}
}
