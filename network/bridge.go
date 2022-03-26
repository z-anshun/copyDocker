package network

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"net"
	"os/exec"
	"strings"
)

/*
 @Author: as
 @Date: Creat in 22:10 2022/3/21
 @Description: Bridge 驱动的实现
*/

type BridgeNetworkDriver struct {
}

func (b BridgeNetworkDriver) Name() string {
	panic("implement me")
}

func (b *BridgeNetworkDriver) Create(subnet string, name string) (*NetWork, error) {
	// 调取网关IP地址和网络IP段
	ip, ipRange, _ := net.ParseCIDR(subnet)
	ipRange.IP = ip
	//ipRange.IP=ip
	n := &NetWork{Name: name, IpRange: ipRange}
	// 初始化桥接
	if err := b.initBridge(n); err != nil {
		logrus.Errorf("Error init bridge %v", err)
		return nil, err
	}
	return n, nil
}

func (b *BridgeNetworkDriver) Delete(network NetWork) error {

	bridgeName := network.Name
	br, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return err
	}
	// ip link del xxx
	return netlink.LinkDel(br)
}

// Connect 实现 Veth 挂载到 bridge 的上
func (b *BridgeNetworkDriver) Connect(network *NetWork, endpoint *Endpoint) error {
	bridgeName := network.Name
	// 通过接口名获取到 Linux Bridge 接口的对象和接口属性
	br, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return err
	}

	// 创建 Veth 接口的配置
	la := netlink.NewLinkAttrs()
	// 由于 Linux 接口名的限制，名字取 endpoint ID 的前5位
	la.Name = endpoint.ID[:5]
	// 通过设置 Veth 接口的master属性，设置这个 Veth 的前一端挂载到网络对应的 Linux Bridge 上
	// ip link set xxx master bridgeName
	la.MasterIndex = br.Attrs().Index

	// 创建 Veth 对象，通过 PeerName 配置 Veth 另一端的接口名
	// 配置 Veth 另一端的名字 cif-{endpoint ID的前5位}
	endpoint.Device = netlink.Veth{
		LinkAttrs: la,
		PeerName:  "cif-" + endpoint.ID[:5],
	}

	// 调用netlink 的 LinkAdd 方法创建出这个 Veth 接口
	if err = netlink.LinkAdd(&endpoint.Device); err != nil {
		return fmt.Errorf("Error Add Endpoint Device: %v", err)
	}

	// ip l s xxx up
	if err := netlink.LinkSetUp(&endpoint.Device); err != nil {
		return err
	}
	return nil
}

func (b *BridgeNetworkDriver) Disconnect(network NetWork, endpoint *Endpoint) error {
	panic("implement me")
}

// 1. 创建 Bridge 虚拟设备
// 2. 设置 Bridge 设备地址和路由
// 3. 启动 Bridge 设备
// 4. 设置 iptables SNAT 规则, 保证 Bridge 上的容器的 Veth 能够访问外部网络
func (b *BridgeNetworkDriver) initBridge(n *NetWork) error {
	// 创建 Bridge 虚拟设备
	bridgeName := n.Name
	if err := createBridgeInterface(bridgeName); err != nil {
		return fmt.Errorf("Error add bridge %s ,Error: %v", bridgeName, err)
	}

	// 设置 Bridge 设备的地址和路由
	getewayIP := *n.IpRange
	if err := setInterfaceIP(bridgeName, getewayIP.String()); err != nil {
		return fmt.Errorf("Error assigning address:%s on brigde %s with an error of %v",
			getewayIP, bridgeName, err)
	}

	// 启动 Bridge 设备
	if err := setInterfaceUP(bridgeName); err != nil {
		return fmt.Errorf("Error set bridge up: %s,Error: %v", bridgeName, err)
	}

	// 设置 iptables 的 SNAT 规则
	if err := setupIPTables(bridgeName, n.IpRange); err != nil {
		return fmt.Errorf("Error setting iptables for %s: %v", bridgeName, err)
	}

	return nil
}

// 创建的实现
func createBridgeInterface(bridgeName string) error {
	// 先检查是否以及存在这个同名的 Bridge 设备
	_, err := net.InterfaceByName(bridgeName)
	if err == nil || !strings.Contains(err.Error(), "no such network interface") {
		return err
	}

	// 初始化一个 netlink 的 Link 基础对象，Link 的名字即 Bridge 虚拟设备的名字
	la := netlink.NewLinkAttrs()
	la.Name = bridgeName

	// 使用刚才创建的 Link 的属性创建 netlink 的 Bridge 对象
	br := &netlink.Bridge{LinkAttrs: la}

	// 使用 netlink 的 Linkadd 方法，创建 Bridge 虚拟网络设备
	// 相当于 ip link a XXX type bridge
	if err := netlink.LinkAdd(br); err != nil {
		return fmt.Errorf("Bridge creation failed for bridge %s:%v", bridgeName, err)
	}
	return nil
}

// 设置一个网络接口的IP地址
func setInterfaceIP(name string, rawIP string) error {
	iface, err := netlink.LinkByName(name)
	if err != nil {
		return err
	}

	// 返回值中既包含了网段信息，也包含了原始IP
	// 如：rawIP="192.168.0.1/24"
	// 那么：ipNet 既包含了 192.168.0.0/24,也包含了原始的ip 192.168.0.1
	ipNet, err := netlink.ParseIPNet(rawIP)
	if err != nil {
		return err
	}
	addr := &netlink.Addr{IPNet: ipNet}
	// 等同于 ip addr add ${rawIP} dev ${name},
	// 若还配置了网段信息，则会自动配置路由表 192.168.0.0/24 转发到 对应的网络接口
	return netlink.AddrAdd(iface, addr)

}

// 设置网络接口为 UP 状态
// ip link set xxx up
func setInterfaceUP(bridgeName string) error {
	iface, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return fmt.Errorf("Error retrieving a link named [ %s]: %v",
			iface.Attrs().Name, err)
	}

	if err := netlink.LinkSetUp(iface); err != nil {
		return fmt.Errorf("Error enabling interface for %s: %v", bridgeName, err)
	}
	return nil
}

// 设置 iptables 对应 bridge 的 MASQUERADE 规则
func setupIPTables(bridgeName string, subnet *net.IPNet) error {
	// 因为没有直接操作 iptables 的库
	// 创建 iptables 命令
	// iptables -t nat -A POSTROUTING -s <bridgeName> ! -o <bridgeName> -j MASQUERADE
	// -A设置POSTOUTING链 ：用于源地址转换（SNAT）。
	iptablesCmd := fmt.Sprintf("-t nat -A POSTROUTING -s %s ! -o %s -j MASQUERADE",
		subnet.String(), bridgeName)
	cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)

	output, err := cmd.Output()
	if err != nil {
		logrus.Errorf("iptables Output %v", output)
	}
	return nil
}
