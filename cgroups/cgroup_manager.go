package cgroups

import (
	"copyDocker/cgroups/subsystems"
	"github.com/sirupsen/logrus"
)

/*
 @Author: as
 @Date: Creat in 23:16 2022/3/14
 @Description: 把不同的 subsystem 中的 cgroup 进行管理
*/

type CgroupManager struct {
	// cgroup 的路径，相对于 root cgroup 目录的路径
	Path string
	// 资源配置
	Resource *subsystems.ResourceConfig
}

func NewCgroupManager(path string) *CgroupManager {
	return &CgroupManager{Path: path}
}

// Apply 将进程 PID 加入到每个 cgroup
func (c *CgroupManager) Apply(pid int) error {
	for _, subSysIns := range subsystems.SubsystemsIns {
		subSysIns.Apply(c.Path, pid)
	}
	return nil
}

// Set 设置对应 subsystem 挂载中 cgroup 资源限制
func (c *CgroupManager) Set(config *subsystems.ResourceConfig) error {
	for _, subSysIns := range subsystems.SubsystemsIns {
		subSysIns.Set(c.Path, config)
	}
	return nil
}

// Destroy 释放各 subsystem 挂载中的cgroup
func (c *CgroupManager) Destroy() error {
	for _,subSysIns:=range subsystems.SubsystemsIns{
		if err:=subSysIns.Remove(c.Path);err!=nil{
			logrus.Warnf("remove cgroup fail %v",err)
		}
	}
	return nil
}
