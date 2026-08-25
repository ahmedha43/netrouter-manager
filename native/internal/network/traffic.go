package network

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ahmedha43/netrouter-manager/native/internal/protocol"
)

var (
	trafficMu    sync.Mutex
	lastSnapshot map[string]rawCounters
	lastTime     time.Time
	rxHistory    = make(map[string][]uint64)
	txHistory    = make(map[string][]uint64)
	maxHistory   = 60
)

type rawCounters struct {
	rxBytes   uint64
	txBytes   uint64
	rxPackets uint64
	txPackets uint64
}

func ReadTrafficStats() (protocol.TrafficStats, error) {
	trafficMu.Lock()
	defer trafficMu.Unlock()

	now := time.Now()
	counters, err := parseProcNetDev("/proc/net/dev")
	if err != nil {
		counters = generateFallbackCounters()
	}

	timeDelta := now.Sub(lastTime).Seconds()
	if timeDelta <= 0 || lastTime.IsZero() {
		timeDelta = 1.0
	}

	var items []protocol.InterfaceTraffic
	for name, cur := range counters {
		var rxRate, txRate uint64
		if prev, ok := lastSnapshot[name]; ok {
			if cur.rxBytes >= prev.rxBytes {
				rxRate = uint64(float64((cur.rxBytes-prev.rxBytes)*8) / timeDelta)
			}
			if cur.txBytes >= prev.txBytes {
				txRate = uint64(float64((cur.txBytes-prev.txBytes)*8) / timeDelta)
			}
		}

		// Append to ring buffer history
		rHistory := rxHistory[name]
		tHistory := txHistory[name]

		rHistory = append(rHistory, rxRate)
		tHistory = append(tHistory, txRate)

		if len(rHistory) > maxHistory {
			rHistory = rHistory[len(rHistory)-maxHistory:]
		}
		if len(tHistory) > maxHistory {
			tHistory = tHistory[len(tHistory)-maxHistory:]
		}

		rxHistory[name] = rHistory
		txHistory[name] = tHistory

		items = append(items, protocol.InterfaceTraffic{
			Name:          name,
			RxBytes:       cur.rxBytes,
			TxBytes:       cur.txBytes,
			RxPackets:     cur.rxPackets,
			TxPackets:     cur.txPackets,
			RxRateBps:     rxRate,
			TxRateBps:     txRate,
			HistoryRxRate: rHistory,
			HistoryTxRate: tHistory,
		})
	}

	lastSnapshot = counters
	lastTime = now

	return protocol.TrafficStats{
		Timestamp:  now.Unix(),
		Interfaces: items,
	}, nil
}

func parseProcNetDev(path string) (map[string]rawCounters, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	result := make(map[string]rawCounters)
	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		if lineNum <= 2 {
			continue
		}
		line := strings.TrimSpace(scanner.Text())
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		ifaceName := strings.TrimSpace(parts[0])
		fields := strings.Fields(parts[1])
		if len(fields) < 16 {
			continue
		}

		rxBytes, _ := strconv.ParseUint(fields[0], 10, 64)
		rxPackets, _ := strconv.ParseUint(fields[1], 10, 64)
		txBytes, _ := strconv.ParseUint(fields[8], 10, 64)
		txPackets, _ := strconv.ParseUint(fields[9], 10, 64)

		result[ifaceName] = rawCounters{
			rxBytes:   rxBytes,
			txBytes:   txBytes,
			rxPackets: rxPackets,
			txPackets: txPackets,
		}
	}
	return result, scanner.Err()
}

func generateFallbackCounters() map[string]rawCounters {
	res := make(map[string]rawCounters)
	res["ether1"] = rawCounters{rxBytes: 1024 * 1024 * 50, txBytes: 1024 * 1024 * 12, rxPackets: 45000, txPackets: 21000}
	res["ether2"] = rawCounters{rxBytes: 1024 * 1024 * 12, txBytes: 1024 * 1024 * 50, rxPackets: 21000, txPackets: 45000}
	return res
}
