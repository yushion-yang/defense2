// chain.go — 链网络系统（Union-Find 并查集）。
//
// 相邻塔（距离 <= ChainDistance）自动形成链组，组内塔获得战力临时加成。
// 链越大加成越高：bonus = 组内塔数 × ChainStrengthPerTower。
//
// 算法选择：Union-Find（路径压缩 + 按秩合并），复杂度近似 O(n²)。
// n 是塔总数（最多 ~60），O(n²) 完全可接受，且实现简洁。
//
// 性能设计：
//   - 栈数组缓冲区（[64]int）替代堆分配——塔池上限 64，零 GC 压力
//   - 每帧由 Pipeline Step 4 调用 RebuildChainNetwork，自动更新所有塔的链加成
//   - 加成写入 Strength.Temp["chain"]，波结束时被 ClearTransient 统一清除
package strength

import (
	"math"

	"defense2/internal/config"
)

// ChainDistance 返回链连接最大距离（像素，从 balance.json 实时读取）。
func ChainDistance() float64 { return config.GlobalBalance().Chain.Distance }

// ChainStrengthPerTower 返回每个链组成员贡献的战力值（从 balance.json 实时读取）。
func ChainStrengthPerTower() float64 { return config.GlobalBalance().Chain.StrengthPerTower }

// ChainTower 链网络的输入数据。
// 使用独立 struct 而非直接引用 tower.Tower，避免 strength 包反向依赖 tower 包。
// 调用方（warden system）负责从 Tower 提取字段构建 ChainTower 切片。
type ChainTower struct {
	Index    int           // 塔在列表中的索引（用于结果映射的 key）
	X, Y     float64       // 塔的中心坐标（逻辑像素，用于距离计算）
	Strength *StrengthData // 塔的战力数据（可为 nil，nil 时跳过加成写入）
}

// ChainInfo 塔的链网络信息。
type ChainInfo struct {
	GroupID   int     // 所属链组ID
	GroupSize int     // 链组大小
	Bonus     float64 // 链加成值
}

// RebuildChainNetwork 重建链网络，返回塔索引 → ChainInfo。
//
// 算法步骤：
//  1. 初始化 Union-Find：每座塔是独立集合
//  2. O(n²) 双重循环检查每对塔的距离，距离 <= ChainDistance 则合并
//  3. 统计每个根节点的组大小
//  4. 2+ 成员的组发放链加成：bonus = groupSize × ChainStrengthPerTower
//  5. 自动写入 SetTemp("chain", bonus)，后续 RecalcStats 会读取
//
// 栈数组缓冲区避免每帧堆分配（塔池最大 64）。
var (
	chainParentBuf [64]int // Union-Find 父节点
	chainRankBuf   [64]int // Union-Find 秩（按秩合并用）
	chainGroupBuf  [64]int // 组大小（按根节点索引）
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
// 按秩合并保证树高 O(log n)，配合路径压缩达到近 O(1) 单次操作。
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
