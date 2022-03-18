package container

import (
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

// NewParentProcess 父进程
/*
这里的/proc/self/exe 调用中，/proc/self/ 指当前运行进程自己的环境，那么后面跟个exe，
就是自己调用了自己
*/
func NewParentProcess(tty bool,volume string) (*exec.Cmd, *os.File) {
	readPipe, writePipe, err := NewPipe()
	if err != nil {
		logrus.Errorf("New pipe error %v", err)
		return nil, nil
	}

	// 这里相当于自己调用自己,即fork，并且跟上参数 init $command，也就进入了 initCommand
	cmd := exec.Command("/proc/self/exe", "init")
	// 设置隔离
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID |syscall.CLONE_NEWNS | syscall.CLONE_NEWNET | syscall.CLONE_NEWIPC,
	}

	// 设置了 -it 参数，则需要把当前进程的输入输出导入到标准输入输出上
	if tty {
		cmd.Stdin = os.Stdin
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
	}

	// 传入管道读入端，即带着这个文件句柄去创建子进程
	// 进程默认会有三个文件描述，标准输入、输出、错误。所以这里要绑定一个额外的文件描述符
	cmd.ExtraFiles = []*os.File{readPipe}

	mntURL:="/root/mnt/"
	rootURL:="/root/"
	NewWorkSpace(rootURL,mntURL,volume)
	cmd.Dir=mntURL
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
