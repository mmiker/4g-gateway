package mqtt

import (
	"strings"

	log "github.com/zsy-cn/4g-gateway/pkg/logger"
	"github.com/zsy-cn/4g-gateway/pkg/mqtt"
	"github.com/zsy-cn/4g-gateway/pkg/mqtt/packets"
)

// Store MQTT 存储
type Store struct {
	store mqtt.Store
}

// NewStore 文件存储
func NewStore() *Store {
	return &Store{
		store: mqtt.NewFileStore("./mqttStore"),
	}
}

// Open 打开
func (s *Store) Open() {
	s.store.Open()
}

// Put put
func (s *Store) Put(key string, message packets.ControlPacket) {
	switch m := message.(type) {
	case *packets.ConnackPacket:
		log.Info("Put connect ack packet")

	case *packets.ConnectPacket:
		log.Info("Put connect packet")

	case *packets.DisconnectPacket:
		log.Info("Put disconnect packet")

	case *packets.PingreqPacket:
		log.Info("Put ping request packet")

	case *packets.PingrespPacket:
		log.Info("Put ping response packet")

	case *packets.PubackPacket:
		log.Info("Put publish ack packet",
			"message-id", m.MessageID,
		)

	case *packets.PubcompPacket:
		log.Info("Put pubcomp packet",
			"message-id", m.MessageID,
		)

	case *packets.PublishPacket:
		log.Info("Put publish packet",
			"message-id", m.MessageID,
			"topic", m.TopicName,
			"qos", m.Qos,
			"retained", m.Retain,
			"payload", string(m.Payload),
		)

	case *packets.PubrecPacket:
		log.Info("Put pubrec packet",
			"message-id", m.MessageID,
		)

	case *packets.PubrelPacket:
		log.Info("Put pubrel packet",
			"message-id", m.MessageID,
		)

	case *packets.SubackPacket:
		log.Info("Put subscribe ack packet",
			"message-id", m.MessageID,
		)

	case *packets.SubscribePacket:
		log.Info("Put subscribe packet",
			"message-id", m.MessageID,
			"topics", strings.Join(m.Topics, ";"),
		)

	case *packets.UnsubackPacket:
		log.Info("Put unsubscribe ack packet",
			"message-id", m.MessageID,
		)

	case *packets.UnsubscribePacket:
		log.Info("Put unsubscribe packet",
			"message-id", m.MessageID,
			"topic", strings.Join(m.Topics, ";"),
		)

	default:
		log.Debug("Put unknown packet", "key", key, "message", message.String())
	}

	s.store.Put(key, message)
}

// Get Get
func (s *Store) Get(key string) packets.ControlPacket {
	return s.store.Get(key)
}

// All All
func (s *Store) All() []string {
	return s.store.All()
}

// Del Del
func (s *Store) Del(key string) {
	s.store.Del(key)
}

// Close Close
func (s *Store) Close() {
	s.store.Close()
}

// Reset Reset
func (s *Store) Reset() {
	s.store.Reset()
}
