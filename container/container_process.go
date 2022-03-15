package container

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
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
func NewParentProcess(tty bool) (*exec.Cmd, *os.File) {
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
	return cmd, writePipe
}

// RunContainerInitProcess 执行到这里了，也就证明容器所在的进程已经创建出来了，那么，这就是容器的第一个进程
// 使用mount 挂载proc文件系统，以便后续使用 ps 等系统命令查看当前进程资源的情况
func RunContainerInitProcess() error {
	cmdArray:=readUserCommand()
	if cmdArray==nil||len(cmdArray)==0{
		return fmt.Errorf("Run container get user command error, cmdArray is nil")
	}

	defaultMountFlags:=syscall.MS_NOEXEC|syscall.MS_NOSUID|syscall.MS_NODEV

	/*
		MS_NOEXEC：本文件系统不允许运行其它程序
		MS_NOSUID：本系统中运行程序时，不允许 set-user-ID 或者 set-group-ID
		MS_NODEV：所有 mount的系统都会默认设定的参数
	*/
	// 等价于 mount -t proc -o noexec,nosuid,nodev proc /proc
	syscall.Mount("proc","/proc","proc",uintptr(defaultMountFlags),"")

	// 查找对应文件名的绝对路径
	// 即 /bin/sh
	path,err:=exec.LookPath(cmdArray[0])
	if err!=nil{
		logrus.Errorf("Exec loop path error %v",err)
		return err
	}
	logrus.Infof("Find path %s",path)
	// 注意这里的系统调用，能使在容器中，进行 ps 查看进程时，PID=1为前台进程，而不是init
	// 该调用会覆盖当前的进程，即覆盖init进程
	if err := syscall.Exec(path, cmdArray[0:], os.Environ()); err != nil {
		logrus.Errorf(err.Error())
	}
	return nil
}

// NewPipe 创建管道，对用户参数的缓存，具有 4K 的缓冲区
func NewPipe() (*os.File, *os.File, error) {
	read, write, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}
	return read, write, nil
}

func readUserCommand() []string {
	// index 为 3 的文件描述符，也就是传递进来管道的一端
	pipe := os.NewFile(uintptr(3), "pipe")

	msg, err := ioutil.ReadAll(pipe)
	if err != nil {
		logrus.Errorf("init read pipe error %v", err)
		return nil
	}
	msgStr := string(msg)
	return strings.Split(msgStr, " ")
}
