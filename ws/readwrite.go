package ws

import (
	"encoding/json"
	"errors"
	"github.com/ezshare/server/ws/outgoing"
	"github.com/rs/zerolog/log"
	"io"
)

var provider = map[string]func() Event{}

// Typed contains a JSON message and specifies the type of the message.
type Typed struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// ToTypedOutgoing parses an outgoing.Message into a Typed struct which can be
// written to a JSON stream.
func ToTypedOutgoing(outgoing outgoing.Message) (Typed, error) {
	// 将outgoing转换为json格式
	payload, err := json.Marshal(outgoing)
	if err != nil {
		log.Error().Err(err).Msg("marshal outgoing")
		return Typed{}, err
	}
	return Typed{
		Type:    outgoing.Type(),
		Payload: payload,
	}, nil
}

// ReadTypedIncoming reads a JSON message from the reader and parses it. It returns
// the parsed Event and an error if any.
func ReadTypedIncoming(r io.Reader) (Event, error) {
	typed := Typed{}
	if err := json.NewDecoder(r).Decode(&typed); err != nil {
		log.Error().Err(err).Msg("Failed decode incoming")
		return nil, err
	}

	// According to the Type field in typed, create the corresponding type of Event.
	creator, ok := provider[typed.Type]
	if !ok {
		log.Error().Msg(typed.Type + "handler not found")
		return nil, errors.New("Cannot handle " + typed.Type)
	}
	event := creator()

	// Parse the payload to the event
	if err := json.Unmarshal(typed.Payload, event); err != nil {
		log.Error().Err(err).RawJSON("payload", typed.Payload).Msg("Can not parse incoming payload")
		return nil, err
	}
	return event, nil
}

// register registers a function to create an Event type based on the type string.
func register(t string, incoming func() Event) {
	provider[t] = incoming
}
