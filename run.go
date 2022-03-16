package main

import (
	"copyDocker/cgroups"
	"copyDocker/cgroups/subsystems"
	"copyDocker/container"
	"github.com/sirupsen/logrus"
	"os"
	"strings"
)

/*
 @Author: as
 @Date: Creat in 12:01 2022/3/14
 @Description: copyDocker
*/

// Run Start 方法前的调用，即init的实现。首先 clone 一个 namespace 隔离进程
// 然后，在子进程中，调用/proc/self/exe(即自己)，发送init参数，就是实现了init初始化
func Run(tty bool, comArray []string, res *subsystems.ResourceConfig) {
	parent, writePipe := container.NewParentProcess(tty)
	if parent == nil {
		logrus.Errorf("Create New Process error")
		return
	}
	if err := parent.Start(); err != nil {
		logrus.Error(err)
	}
	// 创建 cgroup manager，通过 set 设置，apply加入实现资源限制
	cgroupManager := cgroups.NewCgroupManager("copyDocker-cgroup")
	defer cgroupManager.Destroy()
	// set 设置资源
	cgroupManager.Set(res)
	// 将容器进程加入到各个 subsystem 挂载对应的cgroup中
	cgroupManager.Apply(parent.Process.Pid)
	// 限制完后，开始初始化,并写入命令
	sendInitCommand(comArray, writePipe)
	parent.Wait()

}

func sendInitCommand(cmdArray []string, writePipe *os.File) {
	command := strings.Join(cmdArray, " ")
	logrus.Infof("command all is %s", command)
	writePipe.WriteString(command)
	writePipe.Close()
}
