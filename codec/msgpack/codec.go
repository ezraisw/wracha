package msgpack

import (
	"github.com/pwnedgod/wracha/codec"
	"github.com/vmihailenco/msgpack/v5"
)

type jsonCodec struct {
}

func NewCodec() codec.Codec {
	return &jsonCodec{}
}

func (c jsonCodec) Marshal(v interface{}) ([]byte, error) {
	return msgpack.Marshal(v)
}

func (c jsonCodec) Unmarshal(data []byte, v interface{}) error {
	return msgpack.Unmarshal(data, v)
}
