package gamemode

import (
	"sync"

	defense2 "defense2"
	"defense2/internal/config"
)

func init() {
	config.SetDataFS(&defense2.DataFS)
	// dataFS 就绪后重置缓存，确保延迟加载能正确读取 gamemodes.json
	ResetModeConfigCache()
	registryOnce = sync.Once{}
	registry = map[string]Mode{}
}
