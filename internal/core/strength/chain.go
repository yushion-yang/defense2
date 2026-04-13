// chain.go — 链网络系统（Union-Find）。
// 相邻塔形成链组，组内塔获得战力加成。
package strength

import (
	"math"

	"defense2/internal/config"
)

// ChainDistance 返回链连接最大距离（像素，从 balance.json 实时读取）。
func ChainDistance() float64 { return config.GlobalBalance().Chain.Distance }

// ChainStrengthPerTower 返回每个链组成员贡献的战力值（从 balance.json 实时读取）。
func ChainStrengthPerTower() float64 { return config.GlobalBalance().Chain.StrengthPerTower }

// ChainTower 链网络输入（避免直接依赖 tower 包）。
type ChainTower struct {
	Index    int           // 塔在列表中的索引
	X, Y     float64       // 塔的位置（像素坐标）
	Strength *StrengthData // 塔的战力数据（可为 nil）
}

// ChainInfo 塔的链网络信息。
type ChainInfo struct {
	GroupID   int     // 所属链组ID
	GroupSize int     // 链组大小
	Bonus     float64 // 链加成值
}

// RebuildChainNetwork 重建链网络，返回塔索引 → ChainInfo。
// O(n²) 距离检查 + Union-Find（路径压缩 + 按秩合并）。
// 对 2+ 成员的链组，bonus = groupSize * ChainStrengthPerTower。
// 如果塔有 StrengthData，自动调用 SetTemp("chain", bonus)。
// 栈数组缓冲区，避免每帧堆分配（塔池最大 64）。
var (
	chainParentBuf [64]int
	chainRankBuf   [64]int
	chainGroupBuf  [64]int // groupSize indexed by tower index (after path compression, root ∈ [0,n))
)

func RebuildChainNetwork(towers []ChainTower) map[int]ChainInfo {
	n := len(towers)
	if n == 0 {
		return nil
	}

	// 初始化 Union-Find（栈数组，零堆分配）
	parent := chainParentBuf[:n]
	rank := chainRankBuf[:n]
	for i := 0; i < n; i++ {
		parent[i] = i
		rank[i] = 0
	}

	// O(n²) 距离检查，距离 <= ChainDistance 的塔合并
	chainDist := ChainDistance()
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			dx := towers[i].X - towers[j].X
			dy := towers[i].Y - towers[j].Y
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist <= chainDist {
				UFUnion(parent, rank, i, j)
			}
		}
	}

	// 统计每个链组的大小（栈数组替代 map）
	gs := chainGroupBuf[:n]
	for i := range gs {
		gs[i] = 0
	}
	for i := 0; i < n; i++ {
		root := UFFind(parent, i)
		gs[root]++
	}

	// 构建结果
	result := make(map[int]ChainInfo, n)
	perTower := ChainStrengthPerTower()
	for i := 0; i < n; i++ {
		root := UFFind(parent, i)
		size := gs[root]

		var bonus float64
		if size >= 2 {
			bonus = float64(size) * perTower
		}

		result[towers[i].Index] = ChainInfo{
			GroupID:   root,
			GroupSize: size,
			Bonus:     bonus,
		}

		// 自动设置临时战力加成
		if towers[i].Strength != nil && bonus > 0 {
			towers[i].Strength.SetTemp("chain", bonus)
		}
	}

	return result
}

// UFFind 查找根节点（带路径压缩）。
func UFFind(parent []int, x int) int {
	if parent[x] != x {
		parent[x] = UFFind(parent, parent[x])
	}
	return parent[x]
}

// UFUnion 合并两个集合（按秩合并）。
// rank 可为 nil，此时退化为简单合并（无秩优化）。
func UFUnion(parent, rank []int, x, y int) {
	rx := UFFind(parent, x)
	ry := UFFind(parent, y)
	if rx == ry {
		return
	}
	if rank == nil {
		parent[rx] = ry
		return
	}
	if rank[rx] < rank[ry] {
		parent[rx] = ry
	} else if rank[rx] > rank[ry] {
		parent[ry] = rx
	} else {
		parent[ry] = rx
		rank[rx]++
	}
}
