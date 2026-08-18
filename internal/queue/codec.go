package queue

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

func encodeMessage(message Message) ([]byte, error) {
	var buf bytes.Buffer

	if err := writeString(&buf, message.ID); err != nil {
		return nil, fmt.Errorf("encode message ID: %w", err)
	}

	if err := writeString(&buf, message.Topic); err != nil {
		return nil, fmt.Errorf("encode topic: %w", err)
	}

	if err := writeString(&buf, message.RoutingKey); err != nil {
		return nil, fmt.Errorf("encode routing key: %w", err)
	}

	if err := writeBytes(&buf, message.Payload); err != nil {
		return nil, fmt.Errorf("encode payload: %w", err)
	}

	return buf.Bytes(), nil
}

func decodeMessage(data []byte) (Message, error) {
	reader := bytes.NewReader(data)

	id, err := readString(reader)
	if err != nil {
		return Message{}, fmt.Errorf("decode message ID: %w", err)
	}

	topic, err := readString(reader)
	if err != nil {
		return Message{}, fmt.Errorf("decode topic: %w", err)
	}

	routingKey, err := readString(reader)
	if err != nil {
		return Message{}, fmt.Errorf("decode routing key: %w", err)
	}

	payload, err := readBytes(reader)
	if err != nil {
		return Message{}, fmt.Errorf("decode payload: %w", err)
	}

	return Message{
		Topic:      topic,
		ID:         id,
		RoutingKey: routingKey,
		Payload:    payload,
	}, nil
}

func writeString(buf *bytes.Buffer, value string) error {
	return writeBytes(buf, []byte(value))
}

func writeBytes(buf *bytes.Buffer, value []byte) error {
	if uint64(len(value)) > uint64(^uint32(0)) {
		return fmt.Errorf("value exceeds maximum supported size")
	}

	if err := binary.Write(buf, binary.BigEndian, uint32(len(value))); err != nil {
		return err
	}

	_, err := buf.Write(value)
	return err
}

func readString(reader *bytes.Reader) (string, error) {
	value, err := readBytes(reader)
	if err != nil {
		return "", err
	}

	return string(value), nil
}

func readBytes(reader *bytes.Reader) ([]byte, error) {
	var size uint32

	if err := binary.Read(reader, binary.BigEndian, &size); err != nil {
		return nil, err
	}

	value := make([]byte, size)

	if _, err := reader.Read(value); err != nil {
		return nil, err
	}

	return value, nil
}
