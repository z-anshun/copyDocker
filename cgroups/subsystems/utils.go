package subsystems

import (
	"bufio"
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"path"
	"strings"
)

/*
 @Author: as
 @Date: Creat in 17:40 2022/3/14
 @Description: 设置 subsystem 的工具包
*/

// FindCgroupMountpoint 找到挂载了某个 subsystem 的 hierarchy cgroup 根节点所在的目录
func FindCgroupMountpoint(subsystem string) string {
	f, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		txt := scanner.Text()
		fileds := strings.Split(txt, " ")
		// $ cat /proc/self/mountinfo | grep memory
		// 39 34 0:33 / /sys/fs/cgroup/memory rw,nosuid,nodev,noexec,relatime shared:15 - cgroup cgroup rw,memory
		for _, opt := range strings.Split(fileds[len(fileds)-1], ",") {
			if opt == subsystem {
				return fileds[4]
			}
		}
	}
	if err:=scanner.Err();err!=nil{
		return ""
	}
	return ""
}

// GetCgroupPath 获取在 cgroup 在文件系统中的绝对路径
func GetCgroupPath(subsystem string, cgroupPath string, autoCreate bool) (string, error) {
	cgroupRoot:=FindCgroupMountpoint(subsystem)
	// 查看路径是否正确 或者 自动创建的，则不允许存在
	if _,err:=os.Stat(path.Join(cgroupRoot,cgroupPath));err==nil||
		(autoCreate&&os.IsNotExist(err)){
		// 如果文件不存在，就证明自动创建
		if os.IsNotExist(err){
			if err:=os.Mkdir(path.Join(cgroupRoot,cgroupPath),0755);err!=nil{
				return "",fmt.Errorf("error create cgroup:%v",err)
			}
			logrus.Infof("Create Cgroup file succeess: %s",path.Join(cgroupRoot,cgroupPath))
		}
		return path.Join(cgroupRoot,cgroupPath), nil
	}else{
		return "", fmt.Errorf("cgroup path error:%v",err)
	}
}
