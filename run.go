package main

import (
	"copyDocker/cgroups"
	"copyDocker/cgroups/subsystems"
	"copyDocker/container"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
)

/*
 @Author: as
 @Date: Creat in 12:01 2022/3/14
 @Description: copyDocker
*/

// Run Start 方法前的调用，即init的实现。首先 clone 一个 namespace 隔离进程
// 然后，在子进程中，调用/proc/self/exe(即自己)，发送init参数，就是实现了init初始化,
// 使用 pivot_root 将 root 目录切换 pivot new_root put_old
func Run(tty bool, comArray []string, volume string, res *subsystems.ResourceConfig,
	containerName, imageName string, envSlice []string) {
	// 保证容器名不为空
	if containerName == "" {
		containerName = randStringBytes(10)
	}

	parent, writePipe := container.NewParentProcess(tty, volume, containerName,
		imageName,envSlice)
	if parent == nil {
		logrus.Errorf("Create New Process error")
		return
	}
	if err := parent.Start(); err != nil {
		logrus.Error(err)
	}

	// 记录容器信息,并返回容器名
	containerName, err := recordContainerInfo(parent.Process.Pid, comArray, containerName, volume)
	if err != nil {
		logrus.Errorf("Record container info error: %v", err)
		return
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

	// 如果要交互，才进行等待
	// 也就是如果加了 -d，父级进程就会直接退出，子进程为孤儿进程，由 init 管理
	if tty {
		parent.Wait()
		delContainerInfo(containerName)
		container.DeleteWorkSpace(volume, containerName)
	}

}

// 记录容器的信息
func recordContainerInfo(containerPID int, commandArray []string, containerName, volume string) (string, error) {
	// 生成容器的随机ID
	id := randStringBytes(10)
	// 当前时间创的容器
	createTime := time.Now().Format("2006-01-02 15:04:05")
	command := strings.Join(commandArray, "")

	// 对应的信息实体
	containerInfo := &container.ContainerInfo{
		ID:          id,
		Pid:         strconv.Itoa(containerPID),
		Command:     command,
		CreatedTime: createTime,
		Status:      container.RUNNING,
		Name:        containerName,
		Volume:      volume,
	}

	// json 序列化
	jsonByte, err := json.Marshal(containerInfo)
	if err != nil {
		logrus.Errorf("Record container error %v", err)
		return "", err
	}
	jsonStr := string(jsonByte)

	// 存储容器信息的路径
	// /var/run/copyDocker/${}
	dirUrl := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	// 如果路径不存在
	if err := os.MkdirAll(dirUrl, 0622); err != nil {
		logrus.Errorf("Mkdir error %s , %v", dirUrl, err)
		return "", err
	}

	// /var/run/copyDocker/${}/config.json
	fileName := dirUrl + "/" + container.ConfigName
	// 创建最终的配置文件 config.json
	file, err := os.Create(fileName)
	defer file.Close()
	if err != nil {
		logrus.Errorf("Create file %s error %v.", fileName, err)
		return "", err
	}
	// 将 json 化之后的数据写入文件中
	if _, err := file.WriteString(jsonStr); err != nil {
		logrus.Errorf("File Write string error %v", err)
		return "", err
	}

	return containerName, err
}

// 删除当前容器信息
func delContainerInfo(containerName string) {
	// /var/run/copyDocker/${}
	dirURL := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	if err := os.RemoveAll(dirURL); err != nil {
		logrus.Errorf("Remove dir %s error %v", dirURL, err)
	}
}

func sendInitCommand(cmdArray []string, writePipe *os.File) {
	command := strings.Join(cmdArray, " ")
	logrus.Infof("command all is %s", command)
	writePipe.WriteString(command)
	writePipe.Close()
}

// 随机一个 container ID
func randStringBytes(n int) string {
	letterBytes := "1234567890"
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
