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
	"slices"
	"testing"
)

// The CLI used to probe every up, non-loopback interface. On the development host that is
// almost all noise: `ip -o link show` reported enp4s0, docker0, br-e967edea9dcc and
// fifteen veth* on 2026-09-09 — eighteen probes for one real NIC, and the veth count moves
// every time a container starts. Because every probe waits out a fixed collection window,
// the interesting interface competed with all of them for the one-minute context in
// main(). This census is the only place it is written down; elsewhere it is qualitative,
// so that nothing has to be re-counted when a container comes up.
//
// The filter that fixed it works on the name, and the name is the only criterion that
// can be right in both places: inside a container the probeable NIC is the far end of a
// veth pair and is called eth0. TestProbeableInterfaceNames/inside_a_container is that
// constraint, pinned.
//
// Verified against the `ip -o link show` census of the development host, Docker's
// br-<12 hex digits of the network id> naming for a user-defined bridge, and the
// kernel's own veth/dummy/sit device names.

// itf builds an interface the way net.Interfaces() would report it, so a case fits on
// one line. Only the name and the flags take part in the selection.
func itf(name string, flags net.Flags) net.Interface {
	return net.Interface{Name: name, Flags: flags}
}

// The flag sets net.Interfaces() reports for the three shapes that matter.
const (
	flagsNIC      = net.FlagUp | net.FlagBroadcast | net.FlagMulticast
	flagsLoopback = net.FlagUp | net.FlagLoopback
	flagsVPN      = net.FlagUp | net.FlagPointToPoint // no IFF_MULTICAST
)

// TestIsVirtualInterfaceName covers each family the table drops, and — the half that is
// easy to break — the names that must survive it.
func TestIsVirtualInterfaceName(t *testing.T) {
	for _, tc := range []struct {
		name string
		want bool
	}{
		// Docker: the default bridge, the swarm bridge, and a user-defined bridge named
		// after the first 12 hex digits of the network id.
		{"docker0", true},
		{"docker_gwbridge", true},
		{"br-e967edea9dcc", true},

		// The host end of a container's veth pair — docker, podman, containerd, lxc.
		{"veth1f1a98b", true},

		{"virbr0", true},     // libvirt bridge
		{"virbr0-nic", true}, // ...and the dummy device it enslaves
		{"vnet3", true},      // libvirt guest tap
		{"vmnet1", true},     // VMware
		{"vboxnet0", true},   // VirtualBox
		{"lxdbr0", true},     // LXD
		{"lxcbr0", true},     // LXC
		{"incusbr0", true},   // Incus
		{"fwbr100i0", true},  // Proxmox firewall bridge
		{"podman1", true},    // podman bridge
		{"cni-podman0", true},
		{"nerdctl0", true},
		{"cni0", true},            // Kubernetes CNI
		{"flannel.1", true},       // flannel VXLAN
		{"cali7a3f0b2c1d9", true}, // Calico
		{"cilium_host", true},     // Cilium
		{"vxlan.calico", true},
		{"weave", true},
		{"datapath", true},
		{"kube-ipvs0", true},
		{"genev_sys_6081", true}, // Geneve overlay
		{"vxlan_sys_4789", true},
		{"ovs-system", true},   // Open vSwitch
		{"dummy0", true},       // kernel pseudo devices
		{"ifb0", true},         //
		{"gre0", true},         // kernel tunnel stubs
		{"gretap0", true},      //
		{"tunl0", true},        //
		{"sit0", true},         //
		{"ip_vti0", true},      //
		{"ip6tnl0", true},      //
		{"ip6gre0", true},      //
		{"zt7a3f0b2c1d", true}, // ZeroTier

		// The one that must not be dropped: every mainstream runtime — docker, podman,
		// containerd, LXD, Kubernetes pods — names the interface inside a container eth0.
		// Filtering it would be exactly the failure --all exists to work around, in the
		// case where the operator has no reason to suspect a filter is in the way.
		{"eth0", false},

		// Real NICs, under each of the predictable naming schemes systemd uses.
		{"enp4s0", false},
		{"eno1", false},
		{"ens33", false},
		{"enx244bfedfea3e", false},
		{"wlp3s0", false},
		{"bond0", false},
		{"team0", false},
		{"vlan10", false},
		{"eth0.100", false},

		// The trap the "br-" prefix would have fallen into. Docker's bridge is br- plus
		// twelve hex digits; a bare br0 or bridge0 is usually the operator's own LAN
		// bridge with the cameras on it, and OpenWrt calls that bridge br-lan.
		{"br0", false},
		{"bridge0", false},
		{"br-lan", false},
		{"br-e967edea9dc", false},   // eleven digits, so not the docker shape
		{"br-e967edea9dccf", false}, // thirteen, likewise

		// VPN links are absent from the table on purpose, and the two do not end up alike.
		// wg0 carries no multicast, so probeableInterfaceNames drops it on its flags. A
		// tun device does carry IFF_MULTICAST — verified with `ip tuntap add mode tun`,
		// which yields <POINTOPOINT,MULTICAST,NOARP,UP> — so tun0 is probed. That is the
		// safe way round: naming either would risk dropping a tap-mode link bridged onto
		// a camera VLAN.
		{"tun0", false},
		{"wg0", false},
		{"tailscale0", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := isVirtualInterfaceName(tc.name); got != tc.want {
				t.Fatalf("isVirtualInterfaceName(%q) = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

// TestProbeableInterfaceNames pins the whole selection, in both modes: which interfaces
// are probed, which are reported as skipped, and in what order.
func TestProbeableInterfaceNames(t *testing.T) {
	// The census of the development host, abridged to three of the fourteen veth devices.
	developmentHost := []net.Interface{
		itf("lo", flagsLoopback),
		itf("enp4s0", flagsNIC),
		itf("docker0", flagsNIC),
		itf("br-e967edea9dcc", flagsNIC),
		itf("veth1f1a98b", flagsNIC),
		itf("veth1c299c1", flagsNIC),
		itf("veth6ac5881", flagsNIC),
	}

	for _, tc := range []struct {
		name          string
		interfaces    []net.Interface
		all           bool
		wantProbeable []string
		wantSkipped   []string
	}{
		// The whole point of the change: one probe instead of six, and the operator can
		// see from the skipped list why the other five went.
		{
			"development host", developmentHost, false,
			[]string{"enp4s0"},
			[]string{"docker0", "br-e967edea9dcc", "veth1f1a98b", "veth1c299c1", "veth6ac5881"},
		},
		// --all restores what the CLI did before: everything up and non-loopback.
		{
			"development host with all", developmentHost, true,
			[]string{"enp4s0", "docker0", "br-e967edea9dcc", "veth1f1a98b", "veth1c299c1", "veth6ac5881"},
			nil,
		},

		// The constraint the flag exists to protect. Probing from inside a container has
		// to work without --all, and it does because the runtime calls the interface eth0.
		{
			"inside a container",
			[]net.Interface{itf("lo", flagsLoopback), itf("eth0", flagsNIC)},
			false,
			[]string{"eth0"}, nil,
		},

		// Nothing left to probe. discover() reports the skipped names and points at --all
		// rather than falling open, so this list has to reach it.
		{
			"only virtual interfaces",
			[]net.Interface{itf("lo", flagsLoopback), itf("docker0", flagsNIC), itf("veth1f1a98b", flagsNIC)},
			false,
			nil, []string{"docker0", "veth1f1a98b"},
		},

		// WS-Discovery is multicast-only, so a link without IFF_MULTICAST cannot carry a
		// probe. --all lifts that test too, deliberately: it stays a complete escape
		// hatch, at the price of one failed group join logged as a warning.
		{
			"no multicast", []net.Interface{itf("wg0", flagsVPN)}, false,
			nil, []string{"wg0"},
		},
		{
			"no multicast with all", []net.Interface{itf("wg0", flagsVPN)}, true,
			[]string{"wg0"}, nil,
		},

		// Up and non-loopback are required either way, and a down interface is not
		// reported as skipped — it was never a candidate, and saying so would bury the
		// names that the filter actually dropped.
		{
			"down NIC", []net.Interface{itf("enp4s0", net.FlagBroadcast|net.FlagMulticast)}, false,
			nil, nil,
		},
		{
			"down NIC with all", []net.Interface{itf("enp4s0", net.FlagBroadcast|net.FlagMulticast)}, true,
			nil, nil,
		},
		{
			"loopback only", []net.Interface{itf("lo", flagsLoopback)}, true,
			nil, nil,
		},

		// discover() prints one output column per interface in the order of this slice, so
		// the order has to be the one net.Interfaces() gave, not a sorted or reversed one.
		{
			"order is preserved",
			[]net.Interface{itf("eth2", flagsNIC), itf("eth0", flagsNIC), itf("eth1", flagsNIC)},
			false,
			[]string{"eth2", "eth0", "eth1"}, nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			probeable, skipped := probeableInterfaceNames(tc.interfaces, tc.all)
			if !slices.Equal(probeable, tc.wantProbeable) {
				t.Fatalf("probeableInterfaceNames(_, %v) probed %q, want %q", tc.all, probeable, tc.wantProbeable)
			}
			if !slices.Equal(skipped, tc.wantSkipped) {
				t.Fatalf("probeableInterfaceNames(_, %v) skipped %q, want %q", tc.all, skipped, tc.wantSkipped)
			}
		})
	}
}
