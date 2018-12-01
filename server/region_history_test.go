
package server

import (
	. "github.com/pingcap/check"
	"github.com/pingcap/kvproto/pkg/metapb"
	"time"
)

var _ = Suite(&testUtilSuite{})

type testRegionHistory struct{}

func GetRegionId(nodes []*Node) []int {
	ans := make([]int, 0)
	for _, n := range nodes {
		ans = append(ans, int(n.meta.Id))
	}
	return ans
}

func (s *testRegionHistory) TestSplit(c *C) {
	history := newRegionHistory(nil)
	history.onRegionBootstrap(&metapb.Region{
	Id: 0,
	// Region key range [start_key, end_key).
	StartKey: []byte(""),
	EndKey:   []byte("zzz"),
	})
	time.Sleep(20 * time.Millisecond)
	history.onRegionSplit(0, []*metapb.Region{
		{Id:0, StartKey:[]byte(""), EndKey:[]byte("abc")},
		{Id:1, StartKey:[]byte("abc"), EndKey:[]byte("zzz")},
	})
	//ed := time.Now().UnixNano()
	time.Sleep(20 * time.Millisecond)
	history.onRegionSplit(1, []*metapb.Region{
		{Id:1, StartKey:[]byte("abc"), EndKey:[]byte("bbb")},
		{Id:2, StartKey:[]byte("bbb"), EndKey:[]byte("zzz")},
	})

	//c.Assert(nt.Equal(t), IsTrue)
	//c.Assert(err, IsNil)
	//c.Assert(err, NotNil)
	//c.Assert(nt.Equal(zeroTime), IsTrue)
}

