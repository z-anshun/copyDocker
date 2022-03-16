package container

import (
	"github.com/sirupsen/logrus"
	"os"
	"os/exec"
)

/*
 @Author: as
 @Date: Creat in 23:26 2022/3/16
 @Description: AUFS 的实现，read-only & write-only
*/

// NewWorkSpace 新的工作空间
func NewWorkSpace(rootURL string, mntURL string) {
	CreateReadOnlyLayer(rootURL)
	CreateWriteLayer(rootURL)
	CreateMountPoint(rootURL, mntURL)
}

// CreateReadOnlyLayer 将 busybox.tar 解压到 busybox 目录下，作为容器的只读层
func CreateReadOnlyLayer(rootURL string) {
	busyboxURL := rootURL + "busybox/"
	busyboxTarURL := rootURL + "busybox.tar"
	exist, err := PathExists(busyboxURL)
	if err != nil {
		logrus.Infof("Fail to judge whether dir %s exists. %v", busyboxURL, err)
	}
	// 如果不存在
	if exist == false {
		if err := os.Mkdir(busyboxURL, 0777); err != nil {
			logrus.Errorf("Mkdir dir %s error. %v", busyboxURL, err)
		}
		_, err = exec.Command("tar", "-xvf", busyboxTarURL, "-C", busyboxURL).CombinedOutput()
		if err != nil {
			logrus.Errorf("unTar dir %s error %v", busyboxURL, err)
		}
	}

}

// CreateWriteLayer 创建可写层 writeLayer
func CreateWriteLayer(rootURL string) {
	writeURL := rootURL + "writeLayer/"
	if err := os.Mkdir(writeURL, 0777); err != nil {
		logrus.Errorf("Mkdir %s error: %v", writeURL, err)
	}
}

// CreateMountPoint 创建挂载点
func CreateMountPoint(rootURL string, mntURL string) {
	// 创建 mnt 文件夹作为挂载点
	if err := os.Mkdir(mntURL, 0777); err != nil {
		logrus.Errorf("Mkdir %s error: %v", mntURL, err)
	}
	// mount -t aufs -o dirs=./writeLayer:./busybox none ./mnt
	dirs := "dirs=" + rootURL + "writeLayer:" + rootURL + "busybox"
	cmd := exec.Command("mount", "-t", "aufs", "-o", dirs, "none", mntURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logrus.Errorf("exec mount error:%v", err)
	}
}

// DeleteWorkSpace
// 1. umount mnt 目录
// 2. 删除 mnt 目录
// 3. 在 DeleteWriteLayer 函数中删除 writeLayer 文件夹
func DeleteWorkSpace(rootURL string, mntURL string) {
	DeleteMountPoint(rootURL, mntURL)
	DeleteWriteLayer(rootURL)
}

// DeleteMountPoint umount && del
func DeleteMountPoint(rootURL string, mntURL string) {
	cmd := exec.Command("umount", mntURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logrus.Errorf("%v", err)
	}
	if err := os.RemoveAll(mntURL); err != nil {
		logrus.Errorf("Remove dir %s error:%v", mntURL, err)
	}
}

// DeleteWriteLayer 删除读层
func DeleteWriteLayer(rootURL string) {
	writeURL := rootURL + "/writeLayer"
	if err := os.RemoveAll(writeURL); err != nil {
		logrus.Errorf("Remove dir %s error: %v", writeURL, err)
	}
}

// PathExists 判断某路径是否存在
func PathExists(path string) (bool, error) {
	// 指向文件名
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	// 是否存在
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
