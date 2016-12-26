// Copyright © 2012, 2013 Lawrence E. Bakst. All rights reserved.
// THIS SOURCE CODE IS THE PROPRIETARY INTELLECTUAL PROPERTY AND CONFIDENTIAL
// INFORMATION OF LAWRENCE E. BAKST AND IS PROTECTED UNDER U.S. AND
// INTERNATIONAL LAW. ANY USE OF THIS SOURCE CODE WITHOUT THE PRIOR WRITTEN
// AUTHORIZATION OF LAWRENCE E. BAKST IS STRICTLY PROHIBITED.

package stats

import "fmt"
import "math"
import "time"
import "sort"

// Generate statistics for a slice of points. Calc min, max, avg, med, sdv, and decials
type Stats struct {
	Med float64
	Sdv float64
	Min float64
	Max float64
	Avg float64
	Tot float64
	Nel float64
	Plp float64   // crock for packet loss percentage
	Tim time.Time // also a CROC, will change to 64 bit JS time, like Unix but ms since Epoch
	Ils int64
	Dec []float64
}

const cmax = math.MaxFloat64
const cmin = -math.MaxFloat64

// 2 pass stats
func New(e []float64, ils int64) *Stats {
	if ils <= 0 || ils > 100 {
		ils = 100
	}

	//	fmt.Printf("NewStats: e=%#v\n", e)
	n := len(e)
	s := new(Stats)
	if n <= 0 {
		fmt.Printf("NewStats: no elements to process\n")
		return s
	}
	s.Dec = make([]float64, ils)
	//	fmt.Printf("NewStats: len(cvals)=%d\n", len(e))
	c := make(sort.Float64Slice, n, n)
	n2 := copy(c, e)
	if n != n2 {
		panic("stats")
	}
	sort.Sort(c)
	s.Med = c[len(c)/2]

	min := c[0]
	max := c[n-1]
	rng := max - min
	inc := rng / float64(ils)

	// 200-1000, range == 800, inc = 10; 250 = 50, / 10 == 5, 300 = 100/10 = 10

	//	fmt.Printf("NewStats: med=%f, cvals=%#v\n", s.med, cvals)

	s.Min = cmax
	s.Max = cmin
	s.Sdv = 0
	s.Avg = 0
	s.Nel = float64(n)
	s.Ils = ils
	for _, vv := range c {
		v := vv
		s.Tot += v
		s.Sdv += v * v
		if v > s.Max {
			s.Max = v
		}
		if v > 0 && v < s.Min {
			s.Min = v
		}
		v2 := 0.0
		if inc == 0 {
			v2 = 0.0
		} else {
			v2 = (v - min) / inc
		}
		//		fmt.Printf("NewStats: v=%f, min=%f, inc=%f, v2=%f\n", v, min, inc, v2)
		// INVESTIGATE off by 1
		if v2 >= float64(ils) {
			//			fmt.Printf("NewStats: v2=%f\n", v2)
			v2 = float64(ils) - 1
		}
		if v2 < 0 {
			v2 = 0
		}
		i := int(v2)
		//		fmt.Printf("NewStats: v2=%f, i=%d\n", v2, i)
		s.Dec[i]++
	}
	s.Avg = s.Tot / s.Nel
	s.Sdv = 0.0
	for _, v := range c {
		//		fmt.Printf("NewStats: i=%d v=%v avg=%v, d=%v d^2=%v, s.sdv=%v\n", i, v, s.avg, (v-s.avg), (v-s.avg)*(v-s.avg), s.sdv)
		s.Sdv += (v - s.Avg) * (v - s.Avg)
	}
	//	fmt.Printf("NewStats: s.sdv=%v, s.n=%v\n", s.sdv, s.n)
	s.Sdv = math.Sqrt(s.Sdv / s.Nel)
	//	fmt.Printf("NewStats: s=%#v\n", s)
	return s
}

func (s *Stats) String() string {
	r := fmt.Sprintf("stats{Med:%.2f, Sdv:%.2f, Min:%.2f, Max:%.2f, Avg:%.2f, Tot:%.2f, Nel:%.2f, Plp:%.2f, Tim:%v, Dec:[%d]{",
		s.Med, s.Sdv, s.Min, s.Max, s.Avg, s.Tot, s.Nel, s.Plp, s.Tim, len(s.Dec))
	for k, v := range s.Dec {
		if k != 0 {
			r = r + fmt.Sprintf(", ")
		}
		r = r + fmt.Sprintf("%.f", v)
	}
	r = r + fmt.Sprintf("}}")
	return r
}

type cs struct {
	schan chan *Stats
	e     []float64
	ils   int64
}

var stat_chan chan *cs = make(chan *cs, 10)

// read a stat request, calc the stats, send it down the cstats channel when done. Calc the time it takes to do this.
func StatProcessor() {
	fmt.Printf("StatProcessor: started\n")
	for {
		cs := <-stat_chan
		start := time.Now()
		s := New(cs.e, cs.ils)
		stop := time.Now()
		secs := float64(start.Sub(stop)) / 1000e6
		if secs > 0.1 {
			fmt.Printf("StatProcessor: calc_stats took %f seconds\n", secs)
		}
		cs.schan <- s
		secs++
	}
}

// calc some stats for samples (vals) on a separate thread. probably didn't need to do this, but calc_stats now has to copy the data and sort it
func Stater(cstats chan *Stats, e []float64) {
	//	fmt.Printf("calc_stater: cnt=%f, secs=%f\n", cnt, secs)
	stat_chan <- &cs{schan: cstats, e: e}
}
