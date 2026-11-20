package ip

import "testing"

func TestIsPublic(t *testing.T) {
	tests := []struct {
		addr string
		want bool
	}{
		// 0.0.0.0/8
		{"0.0.0.1", false},
		{"0.255.255.255", false},
		{"1.0.0.0", true},

		// 100.64.0.0/10
		{"100.63.255.255", true},
		{"100.64.0.0", false},
		{"100.127.255.255", false},
		{"100.128.0.0", true},
		{"::ffff:100.64.1.2", false},
		{"::ffff:100.128.0.1", true},

		// 192.0.0.0/24
		{"191.255.255.255", true},
		{"192.0.0.0", false},
		{"192.0.0.8", false},
		{"192.0.0.255", false},
		{"192.0.1.0", true},

		// 198.18.0.0/15
		{"198.17.255.255", true},
		{"198.18.0.0", false},
		{"198.18.0.1", false},
		{"198.19.255.255", false},
		{"198.20.0.0", true},

		// 240.0.0.0/4
		{"239.255.255.255", false}, // multicast
		{"240.0.0.0", false},
		{"240.0.0.1", false},
		{"255.255.255.254", false},
		{"255.255.255.255", false},

		// 100::/64
		{"100::", false},
		{"100::1", false},
		{"100::ffff:ffff:ffff:ffff", false},
		{"100:0:0:1::", true},
		{"ff::1", true},

		// documentation ranges stay public (netsim and tests rely on them)
		{"192.0.2.1", true},
		{"198.51.100.1", true},
		{"203.0.113.1", true},

		// public addresses
		{"31.61.247.94", true},
		{"8.8.8.8", true},
		{"2001:4860:4860::8888", true},

		// private, loopback, link-local, unspecified
		{"10.0.0.1", false},
		{"172.16.0.1", false},
		{"192.168.0.1", false},
		{"fc00::1", false},
		{"127.0.0.1", false},
		{"::1", false},
		{"169.254.1.1", false},
		{"fe80::1", false},
		{"0.0.0.0", false},
		{"::", false},
	}

	for _, tc := range tests {
		t.Run(tc.addr, func(t *testing.T) {
			ip, err := ParseIP(tc.addr)
			if err != nil || ip == nil {
				t.Fatalf("ParseIP(%q) failed: %v", tc.addr, err)
			}
			if got := ip.IsPublic(); got != tc.want {
				t.Errorf("IsPublic(%s) = %v, want %v", tc.addr, got, tc.want)
			}
		})
	}
}

func TestIsPublicInvalid(t *testing.T) {
	for _, ip := range []IP{nil, {}, {1, 2, 3}} {
		if ip.IsPublic() {
			t.Errorf("IsPublic(%v) = true, want false", []byte(ip))
		}
	}
}
