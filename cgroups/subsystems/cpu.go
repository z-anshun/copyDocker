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
 @Date: Creat in 21:28 2022/3/16
 @Description: 对 cpu 资源的限制
*/

type CpuSubsystem struct{}

func (s *CpuSubsystem) Set(cgroupPath string, res *ResourceConfig) error {
	if subsysCgroupPath, err := GetCgroupPath(s.Name(), cgroupPath, true); err != nil {
		return err
	} else {
		if res.CpuShare != "" {
			// 将对应的限制信息写入文件
			if err := ioutil.WriteFile(path.Join(subsysCgroupPath, "cpu.share"), []byte(res.CpuShare), 0644); err != nil {
				return fmt.Errorf("set cgroup cpu share fail %v", err)
			}
		}
		return nil
	}
}

// Apply 将该进程加入 subsystem
func (s *CpuSubsystem) Apply(cgroupPath string, pid int) error {
	subsysCgroupPath, err := GetCgroupPath(s.Name(), cgroupPath, false)
	if err != nil {
		return err
	}
	if err:=ioutil.WriteFile(path.Join(subsysCgroupPath,"tasks"),[]byte(strconv.Itoa(pid)),0644);err!=nil{
		return fmt.Errorf("set cgroup proc fail %v",err)
	}
	return nil
}

// Remove 移除对应的文件
func (s *CpuSubsystem) Remove(cgroupPath string) error {
	subsysCgroupPath, err := GetCgroupPath(s.Name(), cgroupPath, false)
	if err != nil {
		return err
	}
	return os.Remove(subsysCgroupPath)
}

// Name 返回名称
func (s *CpuSubsystem) Name() string {
	return "cpu"
}
