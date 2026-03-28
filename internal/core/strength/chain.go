// chain.go — 链网络系统（Union-Find）。
// 相邻塔形成链组，组内塔获得战力加成。
package strength

import "math"

const (
	// ChainDistance 链连接最大距离（像素）。
	ChainDistance = 150.0
	// ChainStrengthPerTower 每个链组成员贡献的战力值。
	ChainStrengthPerTower = 10.0
)

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
func RebuildChainNetwork(towers []ChainTower) map[int]ChainInfo {
	n := len(towers)
	if n == 0 {
		return make(map[int]ChainInfo)
	}

	// 初始化 Union-Find
	parent := make([]int, n)
	rank := make([]int, n)
	for i := 0; i < n; i++ {
		parent[i] = i
	}

	// O(n²) 距离检查，距离 <= ChainDistance 的塔合并
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			dx := towers[i].X - towers[j].X
			dy := towers[i].Y - towers[j].Y
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist <= ChainDistance {
				ufUnion(parent, rank, i, j)
			}
		}
	}

	// 统计每个链组的大小
	groupSize := make(map[int]int)
	for i := 0; i < n; i++ {
		root := ufFind(parent, i)
		groupSize[root]++
	}

	// 构建结果
	result := make(map[int]ChainInfo, n)
	for i := 0; i < n; i++ {
		root := ufFind(parent, i)
		size := groupSize[root]

		var bonus float64
		if size >= 2 {
			bonus = float64(size) * ChainStrengthPerTower
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

// ufFind 查找根节点（带路径压缩）。
func ufFind(parent []int, x int) int {
	if parent[x] != x {
		parent[x] = ufFind(parent, parent[x])
	}
	return parent[x]
}

// ufUnion 合并两个集合（按秩合并）。
func ufUnion(parent, rank []int, x, y int) {
	rx := ufFind(parent, x)
	ry := ufFind(parent, y)
	if rx == ry {
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
