// Copyright (C) 2022-2026 Jean-Francois SMIGIELSKI
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"net"
	"regexp"
	"strings"
)

// allInterfacesHelp is shared by discover and streams so the two cannot drift. Cobra
// does not wrap flag usage, so it stays inside 80 columns; the device families it stands
// for are enumerated in README.md.
const allInterfacesHelp = "probe the container and VM interfaces too"

// Name prefixes of the devices a container runtime, a hypervisor or a CNI plugin
// creates. A prefix suffices for all of them, and is unambiguous: a NIC a camera can be
// reached through is named en*, eth*, wl*, ww*, bond*, team* or vlan*, and none of those
// collide with an entry below. "tunl" is the kernel's IPIP stub and is deliberately not
// "tun", which would take an OpenVPN tun0 with it.
//
// These are prefixes and not globs because net.Interfaces() reports "veth1f1a98b": the
// "@if2" suffix ip-link displays is its rendering of IFLA_LINK, not part of the name.
//
// This is a denylist on purpose, and it will age: a new CNI plugin ships a new prefix,
// its interface gets probed, and the result is noise rather than a wrong answer. The
// inverse design, an allowlist of en*/eth*/wl*, would be silently wrong on br0, bond0
// and vendor-renamed NICs — and that is the failure that actually loses a camera.
var virtualInterfacePrefixes = []string{
	"docker",        // docker0, docker_gwbridge
	"veth",          // the host end of a container's pair, and weave's vethwe-*
	"virbr", "vnet", // libvirt bridges (incl. virbr0-nic) and guest taps
	"vmnet", "vboxnet", // VMware, VirtualBox
	"lxdbr", "lxcbr", "incusbr", // LXD, LXC, Incus
	"fwbr", "fwln", "fwpr", // Proxmox firewall bridges
	"podman", "nerdctl", "cni", // podman1, nerdctl0, cni0, cni-podman0
	"flannel.", "cali", "cilium_", // Kubernetes CNIs
	"weave", "datapath", "kube-", // weave, kube-ipvs0, kube-bridge
	"vxlan.", "genev_sys_", "vxlan_sys_", "ovs-", // overlays and Open vSwitch
	"dummy", "ifb", "nlmon", "teql", // kernel pseudo devices
	"gre", "tunl", "sit", "erspan", "ip_vti", "ip6tnl", "ip6gre", // kernel tunnel stubs
	"zt", // ZeroTier
}

// dockerBridge matches the name Docker gives a user-defined bridge: "br-" followed by
// the first 12 hex digits of the network id. The shape is the whole point: br0, bridge0
// and OpenWrt's br-lan are legitimate LAN bridges that a camera sits on, and a bare
// "br-" prefix would take br-lan with it.
var dockerBridge = regexp.MustCompile(`^br-[0-9a-f]{12}$`)

// isVirtualInterfaceName reports whether name designates a container, VM or overlay
// device rather than a link a camera can answer on.
//
// The criterion is the name and nothing else, on purpose. Inside a container the
// probeable NIC is the far end of a veth pair and every mainstream runtime names it
// eth0, so a structural criterion that works on a host — no backing device under
// /sys/class/net/<name>/device, no permanent MAC, a driver of "veth" — refuses to probe
// from within a container, which is the one case that has to keep working. A name is
// relative to the network namespace, and eth0 is in neither table.
func isVirtualInterfaceName(name string) bool {
	if dockerBridge.MatchString(name) {
		return true
	}
	for _, prefix := range virtualInterfacePrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// probeableInterfaceNames splits the interfaces into those worth a WS-Discovery probe
// and those dropped. It is pure over its argument so that the whole policy is testable
// from a synthetic list rather than from whatever the machine running the test happens
// to be plugged into. It yields names because that is what wsd.Discover takes, and
// yields the dropped ones so the caller can report what it dropped. Input order is
// preserved: the output of discover() is ordered by interface.
//
// Up and non-loopback are required either way. On top of that the default mode wants an
// interface that can carry the probe at all — WS-Discovery is multicast-only, so a link
// without IFF_MULTICAST can only make wsd.Discover fail its group join — and a name that
// is not tooling's. all lifts both, so that it stays a complete escape hatch.
func probeableInterfaceNames(interfaces []net.Interface, all bool) (probeable, skipped []string) {
	for _, itf := range interfaces {
		if itf.Flags&net.FlagUp == 0 || itf.Flags&net.FlagLoopback != 0 {
			continue
		}
		if !all && (itf.Flags&net.FlagMulticast == 0 || isVirtualInterfaceName(itf.Name)) {
			skipped = append(skipped, itf.Name)
			continue
		}
		probeable = append(probeable, itf.Name)
	}
	return probeable, skipped
}
