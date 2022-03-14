package subsystems

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
)

/*
 @Author: as
 @Date: Creat in 16:11 2022/3/14
 @Description: 内存限制
*/

type MemorySubsystem struct{}

// Set 设置 cgroup 的内存资源限制
func (s *MemorySubsystem) Set(cgroupPath string, res *ResourceConfig) error {
	// 获取 subsystem 在虚拟文件系统中的路径
	if subsysCgroupPtah, err := GetCgroupPath(s.Name(), cgroupPath, false); err == nil {
		if res.MemoryLimit != "" {
			// 设置内存限制，即将限制写入到 cgroup 对应目录的 memory.limit_in_bytes 文件中
			if err := ioutil.WriteFile(path.Join(subsysCgroupPtah, "memory.limit_in_bytes"),
				[]byte(res.MemoryLimit), 0644); err != nil {
				return fmt.Errorf("set cgroup memory fail %v", err)
			}
		}
		return nil
	} else {
		return err
	}
}

// Remove 删除对应节点的限制
func (s *MemorySubsystem) Remove(cgroupPath string) error {
	subsysCgroupPath, err := GetCgroupPath(s.Name(), cgroupPath, false)
	if err != nil {
		return err
	}
	return os.Remove(subsysCgroupPath)
}

// Apply 使进程加入某个 cgroup
func (s *MemorySubsystem) Apply(cgroupPath string, pid int) error {
	subsysCgroupPtah, err := GetCgroupPath(s.Name(), cgroupPath, false)
	if err != nil {
		return fmt.Errorf("get cgroup %s path error:", cgroupPath, err)
	}
	// 写入 tasks 即可
	if err := ioutil.WriteFile(path.Join(subsysCgroupPtah, "tasks"),
		[]byte(strconv.Itoa(pid)), 0644); err != nil {
		fmt.Errorf("set cgroup proc fail %v", err)
		return err
	}
	return nil
}

// Name 返回对应名称
func (s *MemorySubsystem) Name() string {
	return "memory"
}
