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
 @Date: Creat in 21:45 2022/3/16
 @Description: cpu核心数的限制
*/

type CpusetSubSystem struct{}

// Set 初始化 hierarchy，对 cpu 核心数的限制
func (s *CpusetSubSystem) Set(cgroupPath string, res *ResourceConfig) error {
	subsysCgroupPath, err := GetCgroupPath(s.Name(), cgroupPath, true)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(path.Join(subsysCgroupPath, "cpuset.cpus"), []byte(res.CpuSet), 0644); err != nil {
		return fmt.Errorf("set cgroup cpuset fail %v", err)
	}
	return nil

}

func (s *CpusetSubSystem) Apply(cgroupPath string, pid int) error {
	subsysCgroupPath, err := GetCgroupPath(s.Name(), cgroupPath, false)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(path.Join(subsysCgroupPath, "tasks"), []byte(strconv.Itoa(pid)), 0644)
	if err != nil {
		return fmt.Errorf("set cgroup proc fail %v", err)
	}
	return nil
}

// Remove 移除对应的文件
func (s *CpusetSubSystem) Remove(cgroupPath string) error {
	subsysCgroupPath, err := GetCgroupPath(s.Name(), cgroupPath, false)
	if err != nil {
		return err
	}
	return os.Remove(subsysCgroupPath)
}

// Name 返回名称
func (s *CpusetSubSystem) Name() string {
	return "cpuset"
}
