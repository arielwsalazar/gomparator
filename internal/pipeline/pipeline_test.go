package pipeline

import (
	"context"
	"fmt"
	"github.com/ecampolo/gomparator/internal/platform/http"
	"github.com/ecampolo/gomparator/internal/stages"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
	"time"
)

type readerStub struct{}

func (*readerStub) Read() <-chan *stages.URLPair {
	stream := make(chan *stages.URLPair)
	go func() {
		defer close(stream)

		stream <- makeURLPair("hostA1", "hostA2")
		stream <- makeURLPair("hostB1", "hostB2")
		stream <- makeURLPair("hostC1", "hostC2")
		stream <- makeURLPair("hostD1", "hostD2")
		stream <- makeURLPair("hostE1", "hostE2")
		stream <- makeURLPair("hostF1", "hostF2")
	}()

	return stream
}

func makeURLPair(leftHost, rightHost string) *stages.URLPair {
	leftUrl := &stages.URL{}
	leftUrl.URL, leftUrl.Error = http.JoinPath(fmt.Sprintf("http://%s.com", leftHost), "")

	rightUrl := &stages.URL{}
	rightUrl.URL, rightUrl.Error = http.JoinPath(fmt.Sprintf("http://%s.com", rightHost), "")

	sleepRandom(200)
	return &stages.URLPair{Left: leftUrl, Right: rightUrl}
}

type producerStub struct {
	cancel        context.CancelFunc
	toBeProcessed int
}

func (p *producerStub) Produce(in <-chan *stages.URLPair) <-chan *stages.HostsPair {
	stream := make(chan *stages.HostsPair)
	go func() {
		defer close(stream)

		var processed int
		for val := range in {
			if p.toBeProcessed > 0 && p.toBeProcessed == processed {
				p.cancel()
				sleepRandom(200)
			}
			response := &stages.HostsPair{}
			response.Left = &stages.Host{
				URL: val.Left.URL,
			}
			response.Right = &stages.Host{
				URL: val.Right.URL,
			}
			stream <- response
			processed++
			sleepRandom(50)
		}
	}()

	return stream
}

type consumerSpy struct {
	responses []*stages.HostsPair
	times     int
}

func (c *consumerSpy) Consume(val *stages.HostsPair) {
	c.responses = append(c.responses, val)
	c.times++
}

func TestRun(t *testing.T) {
	reader := new(readerStub)
	producer := new(producerStub)
	consumer := new(consumerSpy)

	p := New(reader, producer, context.Background(), consumer)
	p.Run()

	assert.Equal(t, 6, consumer.times)

	assert.Equal(t, "http://hostA1.com", consumer.responses[0].Left.URL.String())
	assert.Equal(t, "http://hostA2.com", consumer.responses[0].Right.URL.String())

	assert.Equal(t, "http://hostB1.com", consumer.responses[1].Left.URL.String())
	assert.Equal(t, "http://hostB2.com", consumer.responses[1].Right.URL.String())

	assert.Equal(t, "http://hostC1.com", consumer.responses[2].Left.URL.String())
	assert.Equal(t, "http://hostC2.com", consumer.responses[2].Right.URL.String())

	assert.Equal(t, "http://hostD1.com", consumer.responses[3].Left.URL.String())
	assert.Equal(t, "http://hostD2.com", consumer.responses[3].Right.URL.String())

	assert.Equal(t, "http://hostE1.com", consumer.responses[4].Left.URL.String())
	assert.Equal(t, "http://hostE2.com", consumer.responses[4].Right.URL.String())

	assert.Equal(t, "http://hostF1.com", consumer.responses[5].Left.URL.String())
	assert.Equal(t, "http://hostF2.com", consumer.responses[5].Right.URL.String())
}

func TestRunWithCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	reader := new(readerStub)
	producer := &producerStub{
		toBeProcessed: 3,
		cancel:        cancel,
	}
	consumer := new(consumerSpy)

	p := New(reader, producer, ctx, consumer)

	p.Run()
	assert.Equal(t, 3, consumer.times)
}

func sleepRandom(max int) {
	r := rand.Intn(max)
	time.Sleep(time.Duration(r) * time.Millisecond)
}
