package main

import (
	"copyDocker/container"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"strconv"
	"syscall"
)

/*
 @Author: as
 @Date: Creat in 16:36 2022/3/20
 @Description: docker stop 的实现
*/

// 1. 找到容器的PID
// 2. kill 容器，信号量为SIGTERM，保证正常退出
// 3. 更改 config 的状态，并重写
func stopContainer(containerName string) {
	pid, err := getContainerPidByName(containerName)
	if err != nil {
		logrus.Errorf("Get container pid by name error:%v", err)
		return
	}

	pidInt, err := strconv.Atoi(pid)
	if err != nil {
		logrus.Errorf("Conver pid from string to int error: %v", err)
		return
	}
	// kill, SIGTERM 使程序正常退出
	if err := syscall.Kill(pidInt, syscall.SIGTERM); err != nil {
		logrus.Errorf("Stop container %s error %v.", containerName, err)
		return
	}
	info, err := getContainerInfoByName(containerName)
	if err != nil {
		return
	}
	info.Status = container.STOP
	info.Pid = ""
	newBytes, err := json.Marshal(info)
	if err != nil {
		logrus.Errorf("Json marshal ContainerInfo error: ", err)
		return
	}
	dirURL := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	configPath := dirURL + container.ConfigName
	if err := ioutil.WriteFile(configPath, newBytes, 0622); err != nil {
		logrus.Errorf("Write file %s error %v.", configPath, err)
	}

}

func getContainerInfoByName(containerName string) (*container.ContainerInfo, error) {
	dirURL := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	configFilePath := dirURL + container.ConfigName
	contentBytes, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		logrus.Errorf("Read %s error: %v", configFilePath, err)
		return nil, err
	}
	var conInfo container.ContainerInfo
	if err := json.Unmarshal(contentBytes, &conInfo); err != nil {
		logrus.Errorf("GetContainerInfoByName unmarshal  error: %v", err)
		return nil, err
	}
	return &conInfo, nil
}

// 移除容器，在 stop 之后
func removeContainer(containerName string) {
	info, err := getContainerInfoByName(containerName)
	if err != nil {
		return
	}
	if info.Status != container.STOP {
		logrus.Errorf("Couldn't remove running container")
		return
	}
	dirURL := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	if err := os.RemoveAll(dirURL); err != nil {
		logrus.Errorf("Remove file %s error %v.", dirURL, err)
		return
	}
	container.DeleteWorkSpace(info.Volume,info.Name)
}
