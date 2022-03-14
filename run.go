package main

import (
	"copyDocker/cgroups/subsystems"
	"copyDocker/container"
	"github.com/sirupsen/logrus"
	"os"
)

/*
 @Author: as
 @Date: Creat in 12:01 2022/3/14
 @Description: copyDocker
*/

// Run Start 方法前的调用，即init的实现。首先 clone 一个 namespace 隔离进程
// 然后，在子进程中，调用/proc/self/exe(即自己)，发送init参数，就是实现了init初始化
func Run(tty bool, comArray []string,res *subsystems.ResourceConfig) {
	//TODO: 增加WritePipe
	parent:=container.NewParentProcess(tty)
	if err:=parent.Start();err!=nil{
		logrus.Error(err)
	}
	parent.Wait()
	os.Exit(-1)
}

