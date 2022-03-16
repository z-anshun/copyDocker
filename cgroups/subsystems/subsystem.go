package subsystems

/*
 @Author: as
 @Date: Creat in 15:49 2022/3/14
 @Description: subsystems 相应的数据结构
*/

// ResourceConfig 资源传递的限制
type ResourceConfig struct {
	MemoryLimit string // 内存限制
	CpuShare    string // CPU 时间片的权重
	CpuSet      string // CPU 核心数
}

// Subsystem 接口，对其资源限制方法的规范
// cgroup 在 hierarchy 的路径，就是其虚拟文件系统中的虚拟路径
// 因此，这里的 cgroup 抽象成了 path
type Subsystem interface {
	Name() string                               // 返回 subsystem 的名字，如 CPU memory
	Set(path string, res *ResourceConfig) error // 设置对应 cgroup 节点的资源限制
	Apply(path string, pid int) error           // 将进程添加至 cgroup 中
	Remove(path string) error                   // 移除某个 cgroup
}

// SubsystemsIns 通过不同的 subsystem 初始化实例创建资源限制处理链数组
var (
	SubsystemsIns = []Subsystem{
		&CpusetSubSystem{},
		&MemorySubsystem{},
		&CpuSubsystem{},
	}
)
