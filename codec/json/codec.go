package json

import (
	"encoding/json"

	"github.com/pwnedgod/wracha/codec"
)

type jsonCodec struct {
}

func NewCodec() codec.Codec {
	return &jsonCodec{}
}

func (c jsonCodec) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (c jsonCodec) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
