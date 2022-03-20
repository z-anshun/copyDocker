package main

import (
	"copyDocker/container"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

/*
 @Author: as
 @Date: Creat in 21:24 2022/3/19
 @Description: exec 命令的具体实现
*/

const ENV_EXEC_PID = "copyDocker_pid"
const ENV_EXEC_CMD = "copyDocker_cmd"

func ExecContainer(containerName string, commandArray []string) {

	pid, err := getContainerPidByName(containerName)
	if err != nil {
		logrus.Errorf("Exec container getContainerPidByName %s error %v",
			containerName, err)
		return
	}

	cmdStr := strings.Join(commandArray, "")
	logrus.Infof("container pid %s", pid)
	logrus.Infof("command %s", cmdStr)

	// 子程序，相当于 ./copyDocker exec
	cmd := exec.Command("/proc/self/exe", "exec")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	os.Setenv(ENV_EXEC_PID, pid)
	os.Setenv(ENV_EXEC_CMD, cmdStr)
	// 获取对应的变量
	containerEnvs := getEnvsByPid(pid)
	cmd.Env = append(os.Environ(), containerEnvs...)

	if err := cmd.Run(); err != nil {
		logrus.Errorf("Exec container %s error: %v", containerName, err)
	}
}

func getContainerPidByName(containerName string) (string, error) {
	// 老规矩，先拼接容器存储的路径
	// /var/run/copyDocker/${}/
	dirURL := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	configFilePath := dirURL + container.ConfigName
	// 读取config
	contentBytes, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return "", err
	}
	var containerInfo container.ContainerInfo
	if err := json.Unmarshal(contentBytes, &containerInfo); err != nil {
		return "", err
	}
	return containerInfo.Pid, err
}

// 根据 Pid 来获取 Envs
func getEnvsByPid(pid string) []string {
	// 进程存放环境变量的位置为 /proc/PID/environ
	path := fmt.Sprintf("/proc/%s/environ", pid)
	contentBytes, err := ioutil.ReadFile(path)
	if err != nil {
		logrus.Errorf("Read file %s error %v", path, err)
		return nil
	}
	// 多个环境变量的分隔符为 \u0000
	envs := strings.Split(string(contentBytes), "\u0000")
	return envs
}
