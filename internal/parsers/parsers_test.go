package parsers

import "testing"

func TestPingParsesIputilsRepliesAndStats(t *testing.T) {
	out := "64 bytes from target-http (172.20.0.3): icmp_seq=1 ttl=64 time=0.123 ms\n" +
		"64 bytes from 172.20.0.3: icmp_seq=2 ttl=64 time=1.5 ms\n" +
		"--- target-http ping statistics ---\n" +
		"3 packets transmitted, 2 received, 33.3333% packet loss, time 2002ms\n" +
		"rtt min/avg/max/mdev = 0.123/0.811/1.500/0.688 ms"
	m := Ping(out)
	if m["received"].(int) != 2 || m["loss_percent"].(float64) == 0 {
		t.Fatal(m)
	}
	replies := m["replies"].([]map[string]any)
	if len(replies) != 2 || replies[0]["bytes"].(int) != 64 || replies[0]["icmp_seq"].(int) != 1 || replies[0]["ttl"].(int) != 64 || replies[0]["time_ms"].(float64) != 0.123 {
		t.Fatalf("bad replies %#v", replies)
	}
	stats := m["stats"].(map[string]any)
	if stats["rtt_avg_ms"].(float64) != 0.811 {
		t.Fatalf("bad stats %#v", stats)
	}
}

func TestPingParsesPacketsReceivedVariant(t *testing.T) {
	m := Ping("3 packets transmitted, 2 packets received, 0% packet loss\nround-trip min/avg/max/stddev = 0.1/0.2/0.3/0.4 ms")
	if m["received"].(int) != 2 || m["stats"].(map[string]any)["rtt_mdev_ms"].(float64) != 0.4 {
		t.Fatal(m)
	}
}

func TestNmapParsesProgressAndPorts(t *testing.T) {
	if p := Progress("nmap", "Stats: 0:00:01 elapsed; 0 hosts completed (1 up), 1 undergoing Connect Scan"); p == nil {
		t.Fatal("missing progress")
	}
	m := Nmap("Nmap scan report for target-http (172.20.0.2)\nHost is up (0.00012s latency).\nPORT   STATE SERVICE\n80/tcp open  http\n443/tcp closed https")
	hosts := m["hosts"].([]map[string]any)
	ports := hosts[0]["ports"].([]map[string]any)
	if len(hosts) != 1 || hosts[0]["state"] != "up" || ports[0]["port"].(int) != 80 || ports[0]["state"] != "open" {
		t.Fatalf("bad nmap parse %#v", m)
	}
}
