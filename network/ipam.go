package network

import (
	"encoding/json"
	"github.com/sirupsen/logrus"
	"net"
	"os"
	"path"
	"strings"
)

/*
 @Author: as
 @Date: Creat in 21:10 2022/3/21
 @Description: 实现给容器网段的分配，不会造成重复
*/

const ipamDefaultAllocatorPath = "/var/run/copyDocker/network/ipam/subnet.json"

// IPAM 存放 IP 地址分配信息
type IPAM struct {
	// 分配文件存放位置
	SubnetAllocatorPath string
	// 网段和位图算法的数组map，key为网段，value为分配的位图数组
	Subnets *map[string]string
}

// 使用默认路径作为分配信息存储位置
var ipAllocator = &IPAM{SubnetAllocatorPath: defaultNetworkPath}

// Allocate 实现地址的分配
func (i *IPAM) Allocate(subnet *net.IPNet) (ip net.IP, err error) {
	i.Subnets = &map[string]string{}

	if err = i.load(); err != nil {
		logrus.Errorf("Error load allocation into,%v", err)
		return
	}
	// 防止更改到转过来的ip
	_, subnet, _ = net.ParseCIDR(subnet.String())
	// 返回网段的子网掩码的总长度和网段前面的固定位长度
	// 如：127.0.0.0/8 其子网掩码为 255.0.0.0
	// 那么 subnet.Mask.Size() 返回的就是前面 255 对应的位数和总位数，即 8和32
	one, size := subnet.Mask.Size()

	// 如果之前并未分配过该网段，就初始化网段的分配配置
	if _, exist := (*i.Subnets)[subnet.String()]; !exist {
		// 用0填满该网段的配置，1<<uint8(size-one) 表示有多少个可用地址
		// 2^(size-one) = 1<<uint8(size-one)
		(*i.Subnets)[subnet.String()] = strings.Repeat("0", 1<<uint8(size-one))
	}

	// 现在 对应的字段就为 0 0 0 0 0... 2^n 个0 ，一个 1 代表某个被占用
	// 遍历网段的位图数组
	for c, v := range (*i.Subnets)[subnet.String()] {
		// 找到数组中为 "0" 的项和数组序号，即可以分配的IP
		if v == '0' {
			ipalloc := []byte((*i.Subnets)[subnet.String()])
			ipalloc[c] = '1'
			(*i.Subnets)[subnet.String()] = string(ipalloc)
			// 初始IP
			ip = subnet.IP
			for t := uint(4); t > 0; t-- {
				// 每一项加对应所需的值
				[]byte(ip)[4-t] += uint8(c >> ((t - 1) * 8))
			}
			ip[3] += 1 // 从1开始分配
			break
		}
	}
	// 保存至文件
	i.dump()
	return
}

// Release IP地址的释放
func (i *IPAM) Release(subnet *net.IPNet, ipaddr *net.IP) error {
	i.Subnets = &map[string]string{}
	if err := i.load(); err != nil {
		logrus.Errorf("Error dump allocation info, %v", err)
	}
	// 计算对应的IP索引在网图中的位置
	c := 0
	// IPv4
	releaseIP := ipaddr.To4()
	releaseIP[3] -= 1
	for t := uint(4); t > 0; t-- {
		// 跟上面分配相反
		c += int(releaseIP[t-1]-subnet.IP[t-1]) << ((4 - t) * 8)
	}

	ipalloc := []byte((*i.Subnets)[subnet.String()])
	ipalloc[c] = '0'
	(*i.Subnets)[subnet.String()] = string(ipalloc)

	return i.dump()
}

func (ipam *IPAM) load() error {
	// 首先，查看文件是否存在,若不存在，就表明之前未分配，并不需要加载
	if _, err := os.Stat(ipam.SubnetAllocatorPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	// 打开并读取存储文件
	subnetConfigFile, err := os.Open(ipam.SubnetAllocatorPath)
	defer subnetConfigFile.Close()
	if err != nil {
		return err
	}
	subnetJson := make([]byte, 2000)
	n, err := subnetConfigFile.Read(subnetJson)
	if err != nil {
		return err
	}
	err = json.Unmarshal(subnetJson[:n], ipam.Subnets)
	if err != nil {
		logrus.Errorf("Error dump allocation info, %v", err)
		return err
	}
	return nil
}

// 存储地址分配信息
func (ipam *IPAM) dump() error {
	// 检查存储文件所在文件夹是否存在，不存在就创建
	ipamConfigFileDir, _ := path.Split(ipam.SubnetAllocatorPath)
	if _, err := os.Stat(ipamConfigFileDir); err != nil {
		if os.IsNotExist(err) {
			// mkdir -p ${dir} 命令
			os.MkdirAll(ipamConfigFileDir, 0644)
		} else {
			return err
		}
	}

	// 打开存储文件，os.O_TRUNC 存在即清空，os.O_CREATE 如果不存在就创建
	subnetConfigFile, err := os.OpenFile(ipam.SubnetAllocatorPath,
		os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	defer subnetConfigFile.Close()
	if err != nil {
		return err
	}
	// 序列化
	bs, err := json.Marshal(ipam.Subnets)
	if err != nil {
		return err
	}
	if _, err = subnetConfigFile.Write(bs); err != nil {
		return err
	}
	return nil

}
