package container

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"strings"
)

/*
 @Author: as
 @Date: Creat in 23:26 2022/3/16
 @Description: AUFS 的实现，read-only & write-only
*/

// NewWorkSpace 新的工作空间
func NewWorkSpace(volume, imageName, containerName string) {
	CreateReadOnlyLayer(imageName)
	CreateWriteLayer(containerName)
	CreateMountPoint(containerName, imageName)
	// 根据 volume 判断是否执行挂载数据卷操作
	if volume != "" {
		volumeURLs := volumeUrlExtract(volume)
		length := len(volumeURLs)
		if length == 2 && volumeURLs[0] != "" && volumeURLs[1] != "" {
			MountVolume(volumeURLs, containerName)
			logrus.Infof("%q", volumeURLs)
		} else {
			logrus.Infof("Volume parameter input is not correct .")
		}
	}
}

// MountVolume 挂载数据卷
// 1. 读取宿主机文件目录URL，创建宿主机文件目录 /root/${parent}
// 2. 读取容器挂载点URL，在容器文件系统里创建挂载点 /root/mnt/${containerUrl}
// 3. 把宿主机文件目录挂载到容器挂载点，
func MountVolume(volumeURLs []string, containerName string) {
	// 创建宿主机文件目录
	parentUrl := volumeURLs[0]
	if err := os.Mkdir(parentUrl, 0777); err != nil {
		logrus.Infof("Mkdir parent dir %s error: %v", parentUrl, err)
	}

	// 在容器文件系统里创建挂载点
	containerUrl := volumeURLs[1]
	// root/mnt/${}/containerUrl
	containerVolumeURL := fmt.Sprintf(MntURL, containerName) + "/" + containerUrl
	if err := os.Mkdir(containerVolumeURL, 0777); err != nil {
		logrus.Infof("Mkdir container dir %s error:%v", containerVolumeURL, err)
	}

	// 把宿主机文件目录挂载到容器挂载点
	dirs := "dirs=" + parentUrl
	cmd := exec.Command("mount", "-t", "aufs", "-o", dirs, "none", containerVolumeURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logrus.Errorf("Mount volume failed. %v", err)
	}
}

// CreateReadOnlyLayer 将 busybox.tar 解压到 busybox 目录下，作为容器的只读层
func CreateReadOnlyLayer(imageName string) {
	unTarFolderUrl := RootURL + "/" + imageName + "/"
	imageUrl := RootURL + "/" + imageName + ".tar"
	exist, err := PathExists(unTarFolderUrl)
	if err != nil {
		logrus.Infof("Fail to judge whether dir %s exists. %v", unTarFolderUrl, err)
	}
	// 如果不存在
	if !exist {
		if err := os.Mkdir(unTarFolderUrl, 0622); err != nil {
			logrus.Errorf("Mkdir dir %s error. %v", unTarFolderUrl, err)
		}
		_, err = exec.Command("tar", "-xvf", imageUrl, "-C", unTarFolderUrl).
			CombinedOutput()
		if err != nil {
			logrus.Errorf("unTar dir %s error %v", unTarFolderUrl, err)
		}
	}

}

// CreateWriteLayer 创建可写层 writeLayer
func CreateWriteLayer(containerName string) {
	writeURL := fmt.Sprintf(WriteLayerUrl, containerName)
	if err := os.Mkdir(writeURL, 0777); err != nil {
		logrus.Errorf("Mkdir %s error: %v", writeURL, err)
	}
}

// CreateMountPoint 创建挂载点
func CreateMountPoint(containerName, imageName string) {
	// 创建 mnt 文件夹作为挂载点
	mntUrl := fmt.Sprintf(MntURL, containerName)
	if err := os.Mkdir(mntUrl, 0777); err != nil {
		logrus.Errorf("Mkdir %s error: %v", mntUrl, err)
	}
	// /root/writeLayer/${}
	tmpWriteLayer := fmt.Sprintf(WriteLayerUrl, containerName)
	// /root/${}
	tmpImageLocation := RootURL + "/" + imageName
	// mount -t aufs -o dirs=/root/writeLayer/${}:/root/${} none ./mnt
	dirs := "dirs=" + tmpWriteLayer + ":" + tmpImageLocation
	cmd := exec.Command("mount", "-t", "aufs", "-o", dirs, "none", mntUrl)
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
func DeleteWorkSpace(volume, containerName string) {
	if volume != "" {
		volumeURLs := volumeUrlExtract(volume)
		length := len(volumeURLs)
		if length == 2 && volumeURLs[0] != "" && volumeURLs[1] != "" {
			DeleteVolume(volumeURLs, containerName)
		}
	}

	DeleteMountPoint(containerName)
	DeleteWriteLayer(containerName)
}

func DeleteVolume(volumeURLs []string, containerName string)error{
	mntUrl:=fmt.Sprintf(MntURL,containerName)
	containerUrl:=mntUrl+"/"+volumeURLs[1]
	if _,err:=exec.Command("umount",containerUrl).CombinedOutput();err!=nil{
		logrus.Errorf("Umount volume %s failed. %v", containerUrl, err)
		return err
	}
	return nil
}

// DeleteMountPointWithVolume 删除挂载点，且删除对应的数据卷
// 1. 卸载 volume 挂载点的文件系统（/root/mnt/${container}）
// 2. 卸载整个容器系统的挂载点（/root/mnt）
// 3. 删除容器文件系统挂载点
func DeleteMountPointWithVolume(volumeURLs []string, containerName string) {
	mntUrl := fmt.Sprintf(MntURL, containerName)
	// 卸载容器里的挂载点
	containerUrl := mntUrl + volumeURLs[1]
	cmd := exec.Command("umount", containerUrl)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logrus.Errorf("Umount volume failed. %v", err)
	}
	// 卸载整个容器文件系统的挂载点
	// umount /root/mnt
	cmd = exec.Command("umount", mntUrl)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logrus.Errorf("Umount mountpoint failed. %v", err)
	}

	// 删除容器文件系统挂载点
	if err := os.RemoveAll(mntUrl); err != nil {
		logrus.Infof("Remove mountpoint dir %s error: %v", mntUrl, err)
	}
}

// DeleteMountPoint umount && del
func DeleteMountPoint(containerName string) {
	mntUrl := fmt.Sprintf(MntURL, containerName)
	_, err := exec.Command("umount", mntUrl).CombinedOutput()
	if err != nil {
		logrus.Errorf("%v", err)
	}
	if err := os.RemoveAll(mntUrl); err != nil {
		logrus.Errorf("Remove mountpoint dir %s error:%v", mntUrl, err)
	}
}

// DeleteWriteLayer 删除读层
func DeleteWriteLayer(containerName string) {
	writeURL := fmt.Sprintf(WriteLayerUrl, containerName)
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

// 解析 volume 字符串
func volumeUrlExtract(volume string) []string {
	var volumeURLs []string
	volumeURLs = strings.Split(volume, ":")
	return volumeURLs
}
