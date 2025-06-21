
package natsconsumer

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestNormalConsumer(t *testing.T) {
	// 构造生产者
	nc, err := nats.Connect("nats://localhost:4222")
	require.NoError(t, err)
	defer nc.Drain()

	// 创建js
	js, err := nc.JetStream()
	require.NoError(t, err)

	// 确保创建流
	_ = js.DeleteStream("TEST_STREAM")
	_, err = js.AddStream(&nats.StreamConfig{
		Name:      "TEST_STREAM",
		Subjects:  []string{"test.*"},
		Retention: nats.WorkQueuePolicy,
		Storage:   nats.FileStorage,
	})
	require.NoError(t, err)

	// 创建消费者配置
	_ = js.DeleteConsumer("TEST_STREAM", "TEST_CONSUMER")
	_, err = js.AddConsumer("TEST_STREAM", &nats.ConsumerConfig{
		Durable:   "TEST_CONSUMER",
		AckPolicy: nats.AckExplicitPolicy,
	})
	require.NoError(t, err)

	// 生产10条消息
	for i := 0; i < 10; i++ {
		msg := fmt.Sprintf("test message %d", i)
		_, err = js.Publish("test.subject", []byte(msg))
		require.NoError(t, err)
	}

	logger := zerolog.New(zerolog.NewTestWriter(t)).Level(zerolog.DebugLevel)
	consumer := NewConsumer(&Config{
		NatsURL:      "nats://localhost:4222",
		Concurrency:  2,
		Subject:      "test.subject",
		ConsumerName: "TEST_CONSUMER",
		PullMaxWait:  1 * time.Second,
	}, &logger)

	ch := make(chan string, 10)
	var wg sync.WaitGroup
	wg.Add(10)
	consumer.SetHandler(func(ctx *Context, msg *nats.Msg) {
		defer wg.Done()

		ctx.Logger.Debug().Str("nats_msg_data", string(msg.Data)).Msg("received message")
		ch <- string(msg.Data)

		// 模拟处理时间
		time.Sleep(50 * time.Millisecond)
		// 确认消息
		msg.Ack()
	})

	require.NoError(t, consumer.Start())
	defer consumer.Close()

	wg.Wait()

	require.Len(t, ch, 10)
}

func TestNakConsumer(t *testing.T) {
	// 构造生产者
	nc, err := nats.Connect("nats://localhost:4222")
	require.NoError(t, err)
	defer nc.Drain()

	// 创建js
	js, err := nc.JetStream()
	require.NoError(t, err)

	// 确保创建流
	_ = js.DeleteStream("TEST_STREAM")
	_, err = js.AddStream(&nats.StreamConfig{
		Name:      "TEST_STREAM",
		Subjects:  []string{"test.*"},
		Retention: nats.WorkQueuePolicy,
		Storage:   nats.FileStorage,
	})
	require.NoError(t, err)

	// 创建消费者配置
	_ = js.DeleteConsumer("TEST_STREAM", "TEST_CONSUMER")
	_, err = js.AddConsumer("TEST_STREAM", &nats.ConsumerConfig{
		Durable:   "TEST_CONSUMER",
		AckPolicy: nats.AckExplicitPolicy,
	})
	require.NoError(t, err)

	// 生产1条消息
	_, err = js.Publish("test.subject", []byte("test nak message"))
	require.NoError(t, err)

	logger := zerolog.New(zerolog.NewTestWriter(t)).Level(zerolog.DebugLevel)
	consumer := NewConsumer(&Config{
		NatsURL:      "nats://localhost:4222",
		Concurrency:  2,
		Subject:      "test.subject",
		ConsumerName: "TEST_CONSUMER",
		PullMaxWait:  1 * time.Second,
	}, &logger)

	var counter atomic.Int64

	var wg sync.WaitGroup
	wg.Add(1)

	consumer.SetHandler(func(ctx *Context, msg *nats.Msg) {
		counter.Add(1)

		metadata, err := msg.Metadata()
		require.NoError(t, err)

		ctx.Logger.Debug().
			Str("nats_msg_data", string(msg.Data)).
			Any("nats_msg_metadata", metadata).
			Uint64("nats_num_delivered", metadata.NumDelivered).
			Msg("received message")

		// 模拟处理时间
		time.Sleep(50 * time.Millisecond)

		if metadata.NumDelivered < 3 {
			msg.Nak() // 让第一次，第二次都重试
		} else {
			// 第三次成功
			require.Equal(t, metadata.NumDelivered, uint64(3))
			msg.Ack()
			wg.Done() // 只有在成功处理后才结束等待
		}
	})

	require.NoError(t, consumer.Start())
	defer consumer.Close()

	wg.Wait()
	require.Equal(t, int64(3), counter.Load())
}