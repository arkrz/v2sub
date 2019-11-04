package ping

import (
	"github.com/ThomasZN/v2sub/types"
	gop "github.com/sparrc/go-ping"
	"time"
)

func Ping(nodes types.Nodes, duration time.Duration) {
	ch := make(chan [2]int, len(nodes))
	//defer close(ch)  后续写入会导致 panic

	for i := range nodes {
		nodes[i].Ping = -1

		go func(ch chan<- [2]int, index int) {
			pinger, err := gop.NewPinger(nodes[index].Addr)
			if err != nil {
				panic(err)
			}

			pinger.Count = 4
			pinger.Interval = 500 * time.Millisecond
			pinger.SetPrivileged(true)
			pinger.OnFinish = func(stats *gop.Statistics) {
				ch <- [2]int{index, int(stats.AvgRtt.Nanoseconds() / 1e6)}
			}

			pinger.Run()
		}(ch, i)
	}

	for {
		select {
		case <-time.After(duration):
			return
		case res := <-ch:
			if res[1] != 0 {
				nodes[res[0]].Ping = res[1]
			}
		}
	}
}
