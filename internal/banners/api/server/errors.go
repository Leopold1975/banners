package server

import (
	"encoding/json"
)

type Error struct {
	Err string `json:"error"`
}

func (se Error) ToJSON() []byte {
	b, err := json.Marshal(se)
	if err != nil {
		se.Err = err.Error()

		b, err := json.Marshal(se)
		if err != nil {
			return []byte(`{
				"error": "masrhal error"
			  }`)
		}

		return b
	}

	return b
}
