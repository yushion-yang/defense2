// main.go — 游戏入口（桌面 + WASM 通用）。
// 初始化嵌入式文件系统，配置窗口参数（1200x540 可缩放），启动 Ebitengine 游戏循环。
package main

import (
	"fmt"
	"log"
	"runtime"

	"github.com/hajimehoshi/ebiten/v2"

	defense2 "defense2"
	"defense2/internal/config"
	"defense2/internal/scene"
)

func main() {
	// WASM 环境下捕获 panic，输出到 console（否则只显示 "Go program has already exited"）
	if runtime.GOOS == "js" {
		defer func() {
			if r := recover(); r != nil {
				buf := make([]byte, 4096)
				n := runtime.Stack(buf, false)
				log.Printf("FATAL PANIC: %v\n%s", r, buf[:n])
				fmt.Printf("FATAL PANIC: %v\n%s", r, buf[:n])
			}
		}()
	}

	// 注入嵌入式文件系统，供配置和资源加载器使用
	config.SetDataFS(&defense2.DataFS)
	config.SetAssetFS(&defense2.AssetFS)

	ebiten.SetWindowSize(1200, 540)
	ebiten.SetWindowTitle("Tower Defense")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	g := scene.NewGame()
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
