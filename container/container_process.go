package container

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"syscall"
)

/*
 @Author: as
 @Date: Creat in 23:47 2022/3/13
 @Description: copyDocker
*/

var (
	RUNNING             string = "running"
	STOP                string = "stop"
	Exit                string = "exited"
	DefaultInfoLocation string = "/var/run/copyDocker/%s/"
	ConfigName          string = "config.json"
	ContainerLogFile    string = "container.log"
	MntURL              string = "/root/mnt/%s"
	RootURL             string = "/root"
	WriteLayerUrl       string = "/root/writeLayer/%s"
)

// ContainerInfo 存储容器的信息
type ContainerInfo struct {
	Pid         string   `json:"pid"`          // 容器的init进程在宿主机上对应的PID
	ID          string   `json:"id"`           // 容器ID
	Name        string   `json:"name"`         // 容器名
	Command     string   `json:"command"`      // 容器内 init 进程的运行命令
	CreatedTime string   `json:"created_time"` // 创建时间
	Status      string   `json:"status"`       // 容器状态
	Volume      string   `json:"volume"`       //容器的数据卷
	PortMapping []string `json:"port_mapping"`
}

// NewParentProcess 父进程
/*
这里的/proc/self/exe 调用中，/proc/self/ 指当前运行进程自己的环境，那么后面跟个exe，
就是自己调用了自己
*/
func NewParentProcess(tty bool, volume, containerName,
	imageName string, envSlice []string) (*exec.Cmd, *os.File) {

	readPipe, writePipe, err := NewPipe()
	if err != nil {
		logrus.Errorf("New pipe error %v", err)
		return nil, nil
	}

	// 这里相当于自己调用自己,即fork，并且跟上参数 init $command，也就进入了 initCommand
	cmd := exec.Command("/proc/self/exe", "init")
	// 设置隔离
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWNET | syscall.CLONE_NEWIPC,
	}

	// 设置了 -it 参数，则需要把当前进程的输入输出导入到标准输入输出上
	if tty {
		cmd.Stdin = os.Stdin
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
	} else {
		// 输出至对应的 log 文件
		dirURL := fmt.Sprintf(DefaultInfoLocation, containerName)
		if err := os.MkdirAll(dirURL, 0622); err != nil {
			logrus.Errorf("NewParentProcess mkdir %s error %v.", dirURL, err)
			return nil, nil
		}
		stdLogFilePath := dirURL + ContainerLogFile
		stdLogFile, err := os.Create(stdLogFilePath)
		if err != nil {
			logrus.Errorf("NewParentProcess create file %s error %v", stdLogFilePath, err)
			return nil, nil
		}
		// 重定向输出
		cmd.Stdout = stdLogFile

	}

	// 传入管道读入端，即带着这个文件句柄去创建子进程
	// 进程默认会有三个文件描述，标准输入、输出、错误。所以这里要绑定一个额外的文件描述符
	cmd.ExtraFiles = []*os.File{readPipe}

	// 添加环境
	cmd.Env = append(os.Environ(), envSlice...)

	NewWorkSpace(volume, imageName, containerName)
	cmd.Dir = fmt.Sprintf(MntURL, containerName)
	return cmd, writePipe
}

// NewPipe 创建管道，对用户参数的缓存，具有 4K 的缓冲区
func NewPipe() (*os.File, *os.File, error) {
	read, write, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}
	return read, write, nil
}
