package container

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

/*
 @Author: as
 @Date: Creat in 23:47 2022/3/13
 @Description: 初始化容器，会做的事情 -> 隔离，挂载当前进程 root
*/

// RunContainerInitProcess 执行到这里了，也就证明容器所在的进程已经创建出来了，那么，这就是容器的第一个进程
// 使用mount 挂载proc文件系统，以便后续使用 ps 等系统命令查看当前进程资源的情况
func RunContainerInitProcess() error {
	cmdArray := readUserCommand()
	if cmdArray == nil || len(cmdArray) == 0 {
		return fmt.Errorf("Run container get user command error, cmdArray is nil")
	}

	defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV

	/*
		MS_NOEXEC：本文件系统不允许运行其它程序
		MS_NOSUID：本系统中运行程序时，不允许 set-user-ID 或者 set-group-ID
		MS_NODEV：所有 mount的系统都会默认设定的参数
	*/
	// 等价于 mount -t proc -o noexec,nosuid,nodev proc /proc
	syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), "")

	// 查找对应文件名的绝对路径
	// 即 /bin/sh
	path, err := exec.LookPath(cmdArray[0])
	if err != nil {
		logrus.Errorf("Exec loop path error %v", err)
		return err
	}
	logrus.Infof("Find path %s", path)
	// 注意这里的系统调用，能使在容器中，进行 ps 查看进程时，PID=1为前台进程，而不是init
	// 该调用会覆盖当前的进程，即覆盖init进程
	if err := syscall.Exec(path, cmdArray[0:], os.Environ()); err != nil {
		logrus.Errorf(err.Error())
	}
	return nil
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

func pivotRoot(root string) error {
	// 使当前 root 的老root和新root在同一个文件系统下，即同一个 mount namespace下
	// 将 root 重新 mount 了一次
	// 这里的 bind mount 是将相同的内容换一个挂载点的意思
	if err := syscall.Mount(root, root, "bind", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("Mount rootfs to itself error:%v", err)
	}

	// 创建 rootfs/.pivot_root 存储 old_root
	pivotDir := filepath.Join(root, ".pivot_root")
	if err := os.Mkdir(pivotDir, 0777); err != nil {
		return err
	}

	// pivot_root 到新的 rootfs，老的 old_root 现在挂载在 rootfs/.pivot_root 上
	// 挂载点目前依然能够在 mount 命令中看到
	// new_root put_old
	if err := syscall.PivotRoot(root, pivotDir); err != nil {
		return fmt.Errorf("pivot_root %v", err)
	}

	// 修改当前的工作目录到根目录
	if err := syscall.Chdir("/"); err != nil {
		return fmt.Errorf("chdir / %v", err)
	}

	pivotDir = filepath.Join("/", ".pivot_root")
	// umount rootfs/.pivot_root
	if err := syscall.Unmount(pivotDir, syscall.MNT_DETACH); err != nil {
		return fmt.Errorf("unmount pivot_root")
	}

	// 删除临时文件夹
	return os.Remove(pivotDir)
}

// init 容器时，进行一些了 mount 操作
func setUpMount() {
	// 获取当前的文件路径
	pwd, err := os.Getwd()
	if err != nil {
		logrus.Error("get current location error: %v", err)
		return
	}
	logrus.Infof("Current location is %s", pwd)
	// 将当前进程的 root 切换到当前路径
	pivotRoot(pwd)

	defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV
	// 当前进程信息挂载
	syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), "")
	// 将 tmpfs 文件系统挂载到 dev下
	syscall.Mount("tmpfs", "/dev", "tmpfs",
		syscall.MS_NOSUID|syscall.MS_STRICTATIME, "mode=755")
}
