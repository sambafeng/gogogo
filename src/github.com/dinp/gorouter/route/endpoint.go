package route

import (
	"encoding/json"
	"fmt"
)

func NewEndpoint(host string, port uint16, tags map[string]string) *Endpoint {
	return &Endpoint{
		addr:              fmt.Sprintf("%s:%d", host, port),
		Tags:              tags,
	}
}

type Endpoint struct {
	addr              string
	Tags              map[string]string
}

func (e *Endpoint) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.addr)
}

func (e *Endpoint) CanonicalAddr() string {
	return e.addr
}

func (e *Endpoint) ToLogData() interface{} {
	return struct {
		Addr          string
		Tags          map[string]string
	}{
		e.addr,
		e.Tags,
	}
}
