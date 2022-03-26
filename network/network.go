package network

/*
 @Author: as
 @Date: Creat in 18:42 2022/3/21
 @Description: docker run -p 80:80 --net testbridgenet
*/
import (
	"copyDocker/container"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"text/tabwriter"
)

var (
	defaultNetworkPath = "/var/run/copyDocker/network/network"
	drivers            = map[string]NetworkDriver{}
	networks           = map[string]*NetWork{}
)

// NetWork 网络
// 一个集合，这个网络上的容器可以互相通信
// 可以直接通过 Bridge 设备实现网络互连
type NetWork struct {
	Name    string     // 网络名
	IpRange *net.IPNet // 地址段
	Driver  string     // 网络驱动名
}

// Endpoint 网络端点
// 用于连接容器与网络，保证容器内部与网络的通信
// 比如 Veth Bridge的使用
// 包含地址、Veth设备、端口映射、连接的容器和网络
type Endpoint struct {
	ID          string           `json:"id"`
	Device      netlink.Veth     `json:"device"`
	IPAddress   net.IP           `json:"ip_address"`
	MacAddress  net.HardwareAddr `json:"mac_address"`
	PortMapping []string         `json:"port_mapping"`
	Network     *NetWork         `json:"network"`
}

// NetworkDriver 网络驱动
// 不同的驱动对网络的创建、连接和销毁策略不同。即创建不同的网络需要指定不同的网络驱动
type NetworkDriver interface {
	Name() string                                         // 	驱动名
	Create(subnet string, name string) (*NetWork, error)  // 创建网络
	Delete(network NetWork) error                         // 删除网络
	Connect(network *NetWork, endpoint *Endpoint) error   // 连接容器网络端点到网络
	Disconnect(network NetWork, endpoint *Endpoint) error // 从网络上移除容器网络端点
}

// Init 初始化
func Init() error {
	bridgeDrive := &BridgeNetworkDriver{}
	drivers[bridgeDrive.Name()] = bridgeDrive
	// 判断网络的配置目录是否存在
	if _, err := os.Stat(defaultNetworkPath); err != nil {
		if os.IsNotExist(err) {
			os.Create(defaultNetworkPath)
		} else {
			return err
		}
	}

	// 检查网络配置目录中的所有文件
	filepath.Walk(defaultNetworkPath, func(nwPath string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		// 文件名即为网络名
		_, nwName := path.Split(nwPath)
		nw := &NetWork{Name: nwName}
		if err := nw.load(nwPath); err != nil {
			logrus.Errorf("path %s load network error", nwPath, err)
		}
		networks[nwName] = nw
		return nil
	})
	return nil
}

// ListNetWork 遍历 networks 获取已创建的网络
func ListNetWork() {
	w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
	fmt.Fprint(w, "NAME\tIpRange\tDriver\n")

	// 遍历
	for _, nw := range networks {
		fmt.Fprintf(w, "%s\t%s\t%s\n",
			nw.Name, nw.IpRange, nw.Driver,
		)
	}
	// 输出至标准输出
	if err := w.Flush(); err != nil {
		logrus.Errorf("Flush error %v", err)
	}
}

// CreateNetwork 创建网络
func CreateNetwork(driver, subnet, name string) error {
	// 将网段的字符串转换为 net.IPNet 对象
	// 返回 IP、IPNet、error
	_, ipNet, _ := net.ParseCIDR(subnet)
	// IPAM 分配网关IP，
	getwayIp, err := ipAllocator.Allocate(ipNet)
	if err != nil {
		return err
	}

	ipNet.IP = getwayIp

	// 调用指定的驱动去创建网络
	nw, err := drivers[driver].Create(ipNet.String(), name)
	if err != nil {
		return err
	}
	return nw.dump(defaultNetworkPath)
}

func DeleteNetwork(networkName string) error {
	// 查找网络是否存在
	nw, ok := networks[networkName]
	if !ok {
		return fmt.Errorf("No Such NetWork:%s", networkName)
	}
	// 调用 IPAM 释放分配的IP
	if err := ipAllocator.Release(nw.IpRange, &nw.IpRange.IP); err != nil {
		return err
	}

	// 删除网络创建的设备与配置
	if err := drivers[nw.Driver].Delete(*nw); err != nil {
		return fmt.Errorf("Error Remove Network DriverError: %s", err)
	}
	return nw.remove(defaultNetworkPath)
}

// 将网络配置信息存储在文件系统中，以便于网络查询及在这个网络上连接网络端点
func (nw *NetWork) dump(dumpPath string) error {
	// 首先检查目录是否存在，不在就创建
	if _, err := os.Stat(dumpPath); err != nil {
		if os.IsNotExist(err) {
			// 不存在，创建
			os.MkdirAll(dumpPath, 0644)
		} else {
			return err
		}
	}
	// ${dumpPath}/${netWork}
	nwPath := path.Join(dumpPath, nw.Name)
	// 打开保证为空，只写，不存在就创建
	nwFile, err := os.OpenFile(nwPath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		logrus.Errorf("error:", err)
		return err
	}
	defer nwFile.Close()

	// 跟前面一样，json序列化存储
	nwJson, err := json.Marshal(nw)
	if err != nil {
		logrus.Errorf("Json nw error:", err)
		return err
	}

	_, err = nwFile.Write(nwJson)
	if err != nil {
		logrus.Errorf("error:", err)
		return err
	}
	return nil
}

// 读取网络的配置
func (nw *NetWork) load(dumpPath string) error {
	config, err := os.Open(dumpPath)
	if err != nil {
		return err
	}
	defer config.Close()
	nwJson := make([]byte, 2000)
	n, err := config.Read(nwJson)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(nwJson[:n], nw); err != nil {
		logrus.Errorf("Error load nw : %v", err)
		return err
	}
	return nil
}

// 从网络配置目录删除对应的文件
func (nw *NetWork) remove(dumpPath string) error {
	if _, err := os.Stat(path.Join(dumpPath, nw.Name)); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return os.Remove(path.Join(dumpPath, nw.Name))
}

// Connect 实现容器内到宿主机端口的连接
func Connect(networkName string, cinfo *container.ContainerInfo) error {
	// 从map中找到对应的netWork
	network, ok := networks[networkName]
	if !ok {
		return fmt.Errorf("No Such NetWork: %s", networkName)
	}
	// 获取可用IP，作为容器IP
	ip, err := ipAllocator.Allocate(network.IpRange)
	if err != nil {
		return err
	}

	// 创建网络端点
	ep := &Endpoint{
		ID:          fmt.Sprintf("%s-%s", cinfo.ID, networkName),
		IPAddress:   ip,
		Network:     network,
		PortMapping: cinfo.PortMapping,
	}

	// 调用驱动，去连接和配置网络端点
	if err = drivers[network.Driver].Connect(network, ep); err != nil {
		return err
	}

	// 配置容器NS的IP和路由
	if err = configEndpointIpAddressAndRoute(ep, cinfo); err != nil {
		return err
	}
	// 配置容器到宿主机的端口映射 , 如 -p 80:80
	return configPortMapping(ep, cinfo)
}

// 配置容器网络端点的地址和路由
func configEndpointIpAddressAndRoute(ep *Endpoint, cinfo *container.ContainerInfo) error {
	// 通过网络端点中的 Veth 的另一端
	peerLink, err := netlink.LinkByName(ep.Device.PeerName)
	if err != nil {
		return fmt.Errorf("fail config endpoint: %v", err)
	}

	// 将容器的网络端点加入到容器的网络空间
	// 使这个函数下面的操作都在这个网络空间中进行
	// 执行完函数后，恢复默认的网络空间
	defer enterContainerNetns(&peerLink, cinfo)()

	/*
		--------- 注意 -----------
		下面都是在容器环境执行的，即，配置容器路由，ip，启动等
		ip netns exec ns1 ip l s veth0 up
		--------------------------
	*/

	// 获取到容器的IP地址及网段，用于配置容器内部接口地址
	// 如：容器ip为 192.168.1.2/24，网络的网段为 192.168.1.0/24
	// 那么这里的 ip 字符串为 192.168.1.2/24，用于容器内 Veth 端点配置
	interfaceIP := *ep.Network.IpRange
	interfaceIP.IP = ep.IPAddress
	// ip addr add ${ip} dev ${name}
	if err = setInterfaceIP(ep.Device.PeerName, interfaceIP.String()); err != nil {
		return fmt.Errorf("%v,%s", ep.Network, err)
	}

	// 启动容器内的 Veth 端点
	if err = setInterfaceUP(ep.Device.PeerName); err != nil {
		return err
	}

	// NS 中默认本地地址 127.0.0.1 的 "lo" 网卡是关闭状态
	// 启动它以保证容器能自我访问
	// ip link set xx up
	if err := setInterfaceUP("lo"); err != nil {
		return err
	}

	// 设置容器内的外部请求都通过容器内的 Veth 端点访问
	// 0.0.0.0/0 的网段，表示所有的 IP 地址段
	_, cidr, _ := net.ParseCIDR("0.0.0.0/0")

	// 构建要添加的路由数据，包括网络设备、网关IP及目的网段
	// route add -net 0.0.0.0/0 gw {Bridge 网桥地址} dev {容器内的 Veth 端口设备}
	defaultRoute := &netlink.Route{
		LinkIndex: peerLink.Attrs().Index,
		Gw:        ep.Network.IpRange.IP,
		Dst:       cidr,
	}

	// 添加路由到容器的网络空间
	// route add
	if err = netlink.RouteAdd(defaultRoute); err != nil {
		return err
	}

	return nil
}

// 配置端口映射，使容器能成功访问到外部
func configPortMapping(ep *Endpoint, cinfo *container.ContainerInfo) error {
	// 遍历容器的映射
	for _, pm := range ep.PortMapping {
		// 分割成宿主机的端口和容器的端口
		portMapping := strings.Split(pm, ":")
		if len(portMapping) != 2 {
			logrus.Errorf("port mapping format error. %v", pm)
			continue
		}

		iptablesCmd := fmt.Sprintf("-t nat -A PREROUTING -p tcp -m tcp --dport %s -j DNAT --to-destination %s:%s",
			portMapping[0], ep.IPAddress.String(), portMapping[1])
		// 执行 iptables 命令，并添加端口映射转发规则
		cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
		output, err := cmd.Output()
		if err != nil {
			logrus.Errorf("iptables Output, %v", output)
		}
	}
	return nil
}

// 1. 将容器的网络端点加入到容器的网络空间中
// 2. 锁定当前程序所执行的线程，使当前线程进入到容器的网络空间
// 3. 返回一个函数指针，并执行这个函数，退出容器的网络空间
func enterContainerNetns(enLink *netlink.Link, cinfo *container.ContainerInfo) func() {
	// 找到容器的NS
	// /proc/${pid}/ns/net 打开该文件的文件描述符就可以用来操作 Net Namespace
	// ContainerInfo 中的PID，就是容器映射在主机上的进程ID
	f, err := os.OpenFile(fmt.Sprintf("/proc/%s/ns/net", cinfo.Pid), os.O_RDONLY, 0)
	if err != nil {
		logrus.Errorf("error get container net namespace, %v", err)
	}

	// 对应的文件描述符
	nsFD := f.Fd()

	// 锁定当前的程序所执行的线程，防止 g 被调度到别的线程上
	// 以保证一直在所需的线程中
	runtime.LockOSThread()

	// 修改网络端点 Veth 的另一端，将其移动到容器的 Net Namespace 中
	if err = netlink.LinkSetNsFd(*enLink, int(nsFD)); err != nil {
		logrus.Errorf("error set link netns, %v", err)
		return nil
	}

	// 通过 netns.Get 方法获得当前网络的 Net Namespace
	// 以便后续退出
	origns, err := netns.Get()
	if err != nil {
		logrus.Errorf("error get current netns, %v", err)
	}

	// 调用 netns.Set 方法，将当前进程加入容器的 Net Namespace
	if err = netns.Set(netns.NsHandle(nsFD)); err != nil {
		logrus.Errorf("error set netns,%v", err)
	}

	return func() {
		// 恢复到上面获取到的之前的 Net Namespace
		netns.Set(origns)
		// 关闭文件
		origns.Close()
		runtime.UnlockOSThread() // 解锁
		f.Close()
	}

}
