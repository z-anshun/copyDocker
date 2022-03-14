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
func NewParentProcess(tty bool) *exec.Cmd {
	args := []string{"init"} // 设置参数
	// 这里相当于自己调用自己,即fork，并且跟上参数 init $command，也就进入了 initCommand
	cmd := exec.Command("/proc/self/exe", args...)
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
	return cmd
}

// RunContainerInitProcess 执行到这里了，也就证明容器所在的进程已经创建出来了，那么，这就是容器的第一个进程
// 使用mount 挂载proc文件系统，以便后续使用 ps 等系统命令查看当前进程资源的情况
func RunContainerInitProcess(command string, args []string) error {
	logrus.Infof("command %s", command)

	defaultMountFlags:=syscall.MS_NOEXEC|syscall.MS_NOSUID|syscall.MS_NODEV

	/*
		MS_NOEXEC：本文件系统不允许运行其它程序
		MS_NOSUID：本系统中运行程序时，不允许 set-user-ID 或者 set-group-ID
		MS_NODEV：所有 mount的系统都会默认设定的参数
	*/
	// 等价于 mount -t proc -o noexec,nosuid,nodev proc /proc
	syscall.Mount("proc","/proc","proc",uintptr(defaultMountFlags),"")
	agrv:=[]string{command}

	// 注意这里的系统调用，能使在容器中，进行 ps 查看进程时，PID=1为前台进程，而不是init
	// 该调用会覆盖当前的进程，即覆盖init进程
	if err:=syscall.Exec(command,agrv,os.Environ());err!=nil{
		logrus.Errorf(err.Error())
	}
	return nil
}