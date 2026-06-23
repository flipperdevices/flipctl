package parsers

import (
	"regexp"
	"strconv"
	"strings"
)

func Parse(kind, out string) any {
	switch kind {
	case "ping":
		return Ping(out)
	case "nmap":
		return Nmap(out)
	default:
		return map[string]string{"raw": out}
	}
}

func Progress(kind, line string) any {
	switch kind {
	case "ping":
		if r := pingReply(line); r != nil {
			return map[string]any{"kind": "ping_reply", "reply": r}
		}
	case "nmap":
		if p := nmapProgress(line); p != nil {
			return p
		}
	}
	return nil
}

func Ping(out string) map[string]any {
	replies := []map[string]any{}
	for _, l := range strings.Split(out, "\n") {
		if r := pingReply(l); r != nil {
			replies = append(replies, r)
		}
	}
	m := map[string]any{"transmitted": 0, "received": 0, "replies": replies}
	stats := map[string]any{"transmitted": 0, "received": 0}
	re := regexp.MustCompile(`(\d+) packets transmitted, (\d+) (?:packets )?received(?:, (?:\+\d+ errors, )?(\d+(?:\.\d+)?)% packet loss)?`)
	if x := re.FindStringSubmatch(out); len(x) == 4 {
		a, _ := strconv.Atoi(x[1])
		b, _ := strconv.Atoi(x[2])
		m["transmitted"], stats["transmitted"] = a, a
		m["received"], stats["received"] = b, b
		if x[3] != "" {
			loss, _ := strconv.ParseFloat(x[3], 64)
			m["loss_percent"], stats["loss_percent"] = loss, loss
		}
	}
	rttRe := regexp.MustCompile(`(?:rtt|round-trip) min/avg/max/(?:mdev|stddev) = ([0-9.]+)/([0-9.]+)/([0-9.]+)/([0-9.]+) ms`)
	if x := rttRe.FindStringSubmatch(out); len(x) == 5 {
		keys := []string{"rtt_min_ms", "rtt_avg_ms", "rtt_max_ms", "rtt_mdev_ms"}
		for i, k := range keys {
			v, _ := strconv.ParseFloat(x[i+1], 64)
			m[k], stats[k] = v, v
		}
	}
	m["stats"] = stats
	return m
}

func pingReply(line string) map[string]any {
	re := regexp.MustCompile(`(\d+) bytes from ([^:]+): icmp_seq=(\d+) ttl=(\d+) time[=<]([0-9.]+) ms`)
	x := re.FindStringSubmatch(line)
	if len(x) != 6 {
		return nil
	}
	bytes, _ := strconv.Atoi(x[1])
	seq, _ := strconv.Atoi(x[3])
	ttl, _ := strconv.Atoi(x[4])
	tm, _ := strconv.ParseFloat(x[5], 64)
	return map[string]any{"bytes": bytes, "from": x[2], "icmp_seq": seq, "ttl": ttl, "time_ms": tm}
}

func Nmap(out string) map[string]any {
	hosts := []map[string]any{}
	var cur map[string]any
	portRe := regexp.MustCompile(`^(\d+)/(tcp|udp)\s+(\S+)\s+(\S+)`)
	for _, l := range strings.Split(out, "\n") {
		if strings.HasPrefix(l, "Nmap scan report for ") {
			cur = map[string]any{"host": strings.TrimPrefix(l, "Nmap scan report for "), "ports": []map[string]any{}}
			hosts = append(hosts, cur)
			continue
		}
		if cur != nil && strings.Contains(l, "Host is up") {
			cur["state"] = "up"
			continue
		}
		if cur != nil {
			if x := portRe.FindStringSubmatch(strings.TrimSpace(l)); len(x) == 5 {
				port, _ := strconv.Atoi(x[1])
				cur["ports"] = append(cur["ports"].([]map[string]any), map[string]any{"port": port, "protocol": x[2], "state": x[3], "service": x[4]})
			}
		}
	}
	return map[string]any{"hosts": hosts}
}

func nmapProgress(line string) map[string]any {
	if strings.Contains(line, "Stats:") || strings.Contains(line, "About ") && strings.Contains(line, "% done") {
		return map[string]any{"kind": "nmap_progress", "message": strings.TrimSpace(line)}
	}
	return nil
}
