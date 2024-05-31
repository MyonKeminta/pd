// Copyright 2023 TiKV Project Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pd

import (
	"context"
	"fmt"
	"io"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pingcap/errors"
	"github.com/pingcap/kvproto/pkg/pdpb"
	"github.com/pingcap/kvproto/pkg/tsopb"
	"github.com/pingcap/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/tikv/pd/client/errs"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// TSO Stream Builder Factory

type tsoStreamBuilderFactory interface {
	makeBuilder(cc *grpc.ClientConn) tsoStreamBuilder
}

type pdTSOStreamBuilderFactory struct{}

func (*pdTSOStreamBuilderFactory) makeBuilder(cc *grpc.ClientConn) tsoStreamBuilder {
	return &pdTSOStreamBuilder{client: pdpb.NewPDClient(cc), serverURL: cc.Target()}
}

type tsoTSOStreamBuilderFactory struct{}

func (*tsoTSOStreamBuilderFactory) makeBuilder(cc *grpc.ClientConn) tsoStreamBuilder {
	return &tsoTSOStreamBuilder{client: tsopb.NewTSOClient(cc), serverURL: cc.Target()}
}

// TSO Stream Builder

type tsoStreamBuilder interface {
	build(context.Context, context.CancelFunc, time.Duration, tsoStreamOnRecvCallback) (*tsoStream, error)
}

type pdTSOStreamBuilder struct {
	serverURL string
	client    pdpb.PDClient
}

func (b *pdTSOStreamBuilder) build(ctx context.Context, cancel context.CancelFunc, timeout time.Duration, onRecvCallback tsoStreamOnRecvCallback) (*tsoStream, error) {
	done := make(chan struct{})
	// TODO: we need to handle a conner case that this goroutine is timeout while the stream is successfully created.
	go checkStreamTimeout(ctx, cancel, done, timeout)
	stream, err := b.client.Tso(ctx)
	done <- struct{}{}
	if err == nil {
		return newTSOStream(b.serverURL, pdTSOStreamAdapter{stream}, onRecvCallback), nil
	}
	return nil, err
}

type tsoTSOStreamBuilder struct {
	serverURL string
	client    tsopb.TSOClient
}

func (b *tsoTSOStreamBuilder) build(
	ctx context.Context, cancel context.CancelFunc, timeout time.Duration, onRecvCallback tsoStreamOnRecvCallback,
) (*tsoStream, error) {
	done := make(chan struct{})
	// TODO: we need to handle a conner case that this goroutine is timeout while the stream is successfully created.
	go checkStreamTimeout(ctx, cancel, done, timeout)
	stream, err := b.client.Tso(ctx)
	done <- struct{}{}
	if err == nil {
		return newTSOStream(b.serverURL, tsoTSOStreamAdapter{stream}, onRecvCallback), nil
	}
	return nil, err
}

func checkStreamTimeout(ctx context.Context, cancel context.CancelFunc, done chan struct{}, timeout time.Duration) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-done:
		return
	case <-timer.C:
		cancel()
	case <-ctx.Done():
	}
	<-done
}

type tsoRequestResult struct {
	physical, logical   int64
	count               uint32
	suffixBits          uint32
	respKeyspaceGroupID uint32
}

type grpcTSOStreamAdapter interface {
	Send(clusterID uint64, keyspaceID, keyspaceGroupID uint32, dcLocation string,
		count int64) error
	Recv() (tsoRequestResult, error)
	CloseSend() error
}

type pdTSOStreamAdapter struct {
	stream pdpb.PD_TsoClient
}

func (s pdTSOStreamAdapter) Send(clusterID uint64, _, _ uint32, dcLocation string, count int64) error {
	req := &pdpb.TsoRequest{
		Header: &pdpb.RequestHeader{
			ClusterId: clusterID,
		},
		Count:      uint32(count),
		DcLocation: dcLocation,
	}
	return s.stream.Send(req)
}

func (s pdTSOStreamAdapter) Recv() (tsoRequestResult, error) {
	resp, err := s.stream.Recv()
	if err != nil {
		return tsoRequestResult{}, err
	}
	return tsoRequestResult{
		physical:            resp.GetTimestamp().GetPhysical(),
		logical:             resp.GetTimestamp().GetLogical(),
		count:               resp.GetCount(),
		suffixBits:          resp.GetTimestamp().GetSuffixBits(),
		respKeyspaceGroupID: defaultKeySpaceGroupID,
	}, nil
}

func (s pdTSOStreamAdapter) CloseSend() error {
	return s.stream.CloseSend()
}

type tsoTSOStreamAdapter struct {
	stream tsopb.TSO_TsoClient
}

func (s tsoTSOStreamAdapter) Send(clusterID uint64, keyspaceID, keyspaceGroupID uint32, dcLocation string, count int64) error {
	req := &tsopb.TsoRequest{
		Header: &tsopb.RequestHeader{
			ClusterId:       clusterID,
			KeyspaceId:      keyspaceID,
			KeyspaceGroupId: keyspaceGroupID,
		},
		Count:      uint32(count),
		DcLocation: dcLocation,
	}
	return s.stream.Send(req)
}

func (s tsoTSOStreamAdapter) Recv() (tsoRequestResult, error) {
	resp, err := s.stream.Recv()
	if err != nil {
		return tsoRequestResult{}, err
	}
	return tsoRequestResult{
		physical:            resp.GetTimestamp().GetPhysical(),
		logical:             resp.GetTimestamp().GetLogical(),
		count:               resp.GetCount(),
		suffixBits:          resp.GetTimestamp().GetSuffixBits(),
		respKeyspaceGroupID: resp.GetHeader().GetKeyspaceGroupId(),
	}, nil
}

func (s tsoTSOStreamAdapter) CloseSend() error {
	return s.stream.CloseSend()
}

//// TSO Stream
//
//type tsoStream interface {
//	getServerURL() string
//	//// processRequests processes TSO requests in streaming mode to get timestamps
//	//processRequests(
//	//	clusterID uint64, keyspaceID, keyspaceGroupID uint32, dcLocation string,
//	//	count int64, batchStartTime time.Time,
//	//) (respKeyspaceGroupID uint32, physical, logical int64, suffixBits uint32, err error)
//	processRequestsAsync(
//		clusterID uint64, keyspaceID, keyspaceGroupID uint32, dcLocation string,
//		count int64, batchStartTime time.Time, resultCh chan<- tsoRequestResult,
//	)
//
//	Close()
//}

type batchedReq struct {
	dispatcherID string
	reqID        uint64
	startTime    time.Time
}

type tsoStreamOnRecvCallback = func(reqID uint64, res tsoRequestResult, err error)

var streamIDAlloc atomic.Int32

type tsoStream struct {
	serverURL string
	stream    grpcTSOStreamAdapter

	// Not thread-safe. Assuming that `processRequests` will never be called concurrently.
	reqSeq         int64
	pendingReqIDs  NonblockingSPSC[batchedReq]
	onRecvCallback tsoStreamOnRecvCallback

	estimateLatencyMicros atomic.Uint64

	cancel context.CancelFunc
	wg     sync.WaitGroup

	onTheFlyRequestCountGauge prometheus.Gauge

	onTheFlyRequests atomic.Int32
}

func newTSOStream(serverURL string, stream grpcTSOStreamAdapter, onRecvCallback tsoStreamOnRecvCallback) *tsoStream {
	streamID := fmt.Sprintf("%d", streamIDAlloc.Add(1))

	ctx, cancel := context.WithCancel(context.Background())
	s := &tsoStream{
		serverURL: serverURL,
		stream:    stream,

		pendingReqIDs:  NewNonblockingSPSC[batchedReq](64),
		onRecvCallback: onRecvCallback,

		cancel: cancel,

		onTheFlyRequestCountGauge: onTheFlyRequestCountGauge.WithLabelValues(streamID),
	}
	s.wg.Add(1)
	go s.recvLoop(ctx)
	return s
}

func (s *tsoStream) getServerURL() string {
	return s.serverURL
}

func (s *tsoStream) processRequests(reqID uint64,
	clusterID uint64, keyspaceID, keyspaceGroupID uint32, dcLocation string, count int64, batchStartTime time.Time,
) error {
	s.pendingReqIDs.Push(batchedReq{
		reqID:     reqID,
		startTime: time.Now(),
	})

	if err := s.stream.Send(clusterID, keyspaceID, keyspaceGroupID, dcLocation, count); err != nil {
		if err == io.EOF {
			err = errs.ErrClientTSOStreamClosed
		} else {
			err = errors.WithStack(err)
		}
		return err
	}
	tsoBatchSendLatency.Observe(float64(time.Since(batchStartTime)))

	s.onTheFlyRequestCountGauge.Set(float64(s.onTheFlyRequests.Add(1)))

	// TODO: handle broken stream: ensure all request are responded with error when the broken is broken with error.

	return nil

	//res, err := s.stream.Recv()
	//if err != nil {
	//	if err == io.EOF {
	//		err = errs.ErrClientTSOStreamClosed
	//	} else {
	//		err = errors.WithStack(err)
	//	}
	//	return
	//}
	//requestDurationTSO.Observe(time.Since(start).Seconds())
	//tsoBatchSize.Observe(float64(count))

	//if res.count != uint32(count) {
	//	err = errors.WithStack(errTSOLength)
	//	return
	//}

	//return
}

func (s *tsoStream) recvLoop(ctx context.Context) {
	defer func() {
		s.cancel()
		s.wg.Done()
		s.onTheFlyRequests.Store(0)
		s.onTheFlyRequestCountGauge.Set(0)
	}()

	var finishWithErr error

	const (
		initialEstimateTSOLatencyMicros float64 = 2000

		// Constants used in the simple RC low-pass filter
		filterCutoffFreq float64 = 1.0
		filterRC                 = 1.0 / (2.0 * math.Pi * filterCutoffFreq)
	)

	// A fake very-faraway value
	lastReceiveTime := time.Now().Add(-time.Hour * 10)
	logEstimatedLatency := math.Log(initialEstimateTSOLatencyMicros)

	updateEstimateLatency := func(now time.Time, latency time.Duration) {
		if latency < 0 {
			// Unreachable
			return
		}
		// Delta time
		dt := now.Sub(lastReceiveTime).Seconds()
		// Current sample represented and calculated in log(microseconds)
		currentSample := math.Log(float64(latency.Microseconds()))
		alpha := dt / (filterRC + dt)
		logEstimatedLatency = (1-alpha)*logEstimatedLatency + alpha*currentSample
		s.estimateLatencyMicros.Store(uint64(math.Exp(logEstimatedLatency)))
	}

recvLoop:
	for {
		select {
		case <-ctx.Done():
			finishWithErr = context.Canceled
			break recvLoop
		default:
		}

		res, err := s.stream.Recv()

		if err != nil {
			if err == io.EOF {
				finishWithErr = errs.ErrClientTSOStreamClosed
			} else {
				finishWithErr = errors.WithStack(err)
			}
			break
		}
		req, ok := s.pendingReqIDs.Pop()
		if !ok {
			finishWithErr = errors.New("tsoStream timing order broken")
			break
		}

		now := time.Now()
		latency := now.Sub(req.startTime)

		requestDurationTSO.Observe(latency.Seconds())
		tsoBatchSize.Observe(float64(res.count))

		updateEstimateLatency(now, latency)

		// TODO: Check request and result have matching count.

		s.onRecvCallback(req.reqID, res, nil)
		s.onTheFlyRequestCountGauge.Set(float64(s.onTheFlyRequests.Add(-1)))
	}

	if finishWithErr == nil {
		panic("unreachable")
	}

	log.Info("tsoStream.recvLoop ended", zap.Error(finishWithErr))

	// TODO: Consider concurrent pushing causing some requests left in the queue.
	for {
		req, ok := s.pendingReqIDs.Pop()
		if !ok {
			break
		}
		s.onRecvCallback(req.reqID, tsoRequestResult{}, finishWithErr)
	}
}

func (s *tsoStream) EstimatedRoundTripLatency() time.Duration {
	latencyUs := s.estimateLatencyMicros.Load()
	// Limit it at least 100us
	if latencyUs < 100 {
		latencyUs = 100
	}
	return time.Microsecond * time.Duration(latencyUs)
}

func (s *tsoStream) Close() {
	s.cancel()
	s.wg.Wait()
}

type NonblockingSPSC[T any] struct {
	buffer       []T
	capacity     int64
	capacityMask int64
	head         atomic.Int64
	tail         atomic.Int64
}

func NewNonblockingSPSC[T any](capacity int) NonblockingSPSC[T] {
	capacity = int(roundToPowerOf2(int64(capacity)))
	if capacity <= 0 {
		panic("invalid capacity for NonblockingSPSC")
	}
	return NonblockingSPSC[T]{
		buffer:       make([]T, capacity),
		capacity:     int64(capacity),
		capacityMask: int64(capacity - 1),
	}
}

func roundToPowerOf2(v int64) int64 {
	if v&(v-1) == 0 {
		return v
	}
	// Duplicate bits
	v |= v >> 1
	v |= v >> 2
	v |= v >> 4
	v |= v >> 8
	v |= v >> 16
	v |= v >> 32
	// Plus 1 to carry to next bit
	v += 1
	return v
}

func (q *NonblockingSPSC[T]) Push(v T) bool {
	tail := q.tail.Load()
	head := q.head.Load()
	if tail-head >= q.capacity {
		return false
	}

	q.buffer[tail&q.capacityMask] = v
	if !q.tail.CompareAndSwap(tail, tail+1) {
		panic("race on NonblockingSPSC.Push. did you push concurrently?")
	}
	return true
}

func (q *NonblockingSPSC[T]) Pop() (T, bool) {
	head := q.head.Load()
	tail := q.tail.Load()
	if head >= tail {
		var empty T
		return empty, false
	}
	index := head & q.capacityMask
	res := q.buffer[index]
	// Clear to avoid leak in case the type T is referencing some other data.
	var empty T
	q.buffer[index] = empty
	if !q.head.CompareAndSwap(head, head+1) {
		panic("race on NonblockingSPSC.Push. did you pop concurrently?")
	}
	return res, true
}

//
//type pdTSOStream struct {
//	serverURL string
//	stream    pdpb.PD_TsoClient
//}
//
//func (s *pdTSOStream) getServerURL() string {
//	return s.serverURL
//}
//
//func (s *pdTSOStream) processRequests(
//	clusterID uint64, _, _ uint32, dcLocation string, count int64, batchStartTime time.Time,
//) (respKeyspaceGroupID uint32, physical, logical int64, suffixBits uint32, err error) {
//	start := time.Now()
//	req := &pdpb.TsoRequest{
//		Header: &pdpb.RequestHeader{
//			ClusterId: clusterID,
//		},
//		Count:      uint32(count),
//		DcLocation: dcLocation,
//	}
//
//	if err = s.stream.Send(req); err != nil {
//		if err == io.EOF {
//			err = errs.ErrClientTSOStreamClosed
//		} else {
//			err = errors.WithStack(err)
//		}
//		return
//	}
//	tsoBatchSendLatency.Observe(float64(time.Since(batchStartTime)))
//	resp, err := s.stream.Recv()
//	if err != nil {
//		if err == io.EOF {
//			err = errs.ErrClientTSOStreamClosed
//		} else {
//			err = errors.WithStack(err)
//		}
//		return
//	}
//	requestDurationTSO.Observe(time.Since(start).Seconds())
//	tsoBatchSize.Observe(float64(count))
//
//	if resp.GetCount() != uint32(count) {
//		err = errors.WithStack(errTSOLength)
//		return
//	}
//
//	ts := resp.GetTimestamp()
//	respKeyspaceGroupID = defaultKeySpaceGroupID
//	physical, logical, suffixBits = ts.GetPhysical(), ts.GetLogical(), ts.GetSuffixBits()
//	return
//}
//
//type tsoTSOStream struct {
//	serverURL string
//	stream    tsopb.TSO_TsoClient
//}
//
//func (s *tsoTSOStream) getServerURL() string {
//	return s.serverURL
//}
//
//func (s *tsoTSOStream) processRequests(
//	clusterID uint64, keyspaceID, keyspaceGroupID uint32, dcLocation string,
//	count int64, batchStartTime time.Time,
//) (respKeyspaceGroupID uint32, physical, logical int64, suffixBits uint32, err error) {
//	start := time.Now()
//	req := &tsopb.TsoRequest{
//		Header: &tsopb.RequestHeader{
//			ClusterId:       clusterID,
//			KeyspaceId:      keyspaceID,
//			KeyspaceGroupId: keyspaceGroupID,
//		},
//		Count:      uint32(count),
//		DcLocation: dcLocation,
//	}
//
//	if err = s.stream.Send(req); err != nil {
//		if err == io.EOF {
//			err = errs.ErrClientTSOStreamClosed
//		} else {
//			err = errors.WithStack(err)
//		}
//		return
//	}
//	tsoBatchSendLatency.Observe(float64(time.Since(batchStartTime)))
//	resp, err := s.stream.Recv()
//	if err != nil {
//		if err == io.EOF {
//			err = errs.ErrClientTSOStreamClosed
//		} else {
//			err = errors.WithStack(err)
//		}
//		return
//	}
//	requestDurationTSO.Observe(time.Since(start).Seconds())
//	tsoBatchSize.Observe(float64(count))
//
//	if resp.GetCount() != uint32(count) {
//		err = errors.WithStack(errTSOLength)
//		return
//	}
//
//	ts := resp.GetTimestamp()
//	respKeyspaceGroupID = resp.GetHeader().GetKeyspaceGroupId()
//	physical, logical, suffixBits = ts.GetPhysical(), ts.GetLogical(), ts.GetSuffixBits()
//	return
//}
