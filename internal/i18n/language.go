package i18n

import (
	"encoding/json"
	"fmt"
)

type Language string

const (
	Japanese Language = ""
	English  Language = "en"
)

func (l *Language) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	switch value {
	case "", "ja":
		*l = Japanese
	case "en":
		*l = English
	default:
		return fmt.Errorf("unsupported language: %q", value)
	}
	return nil
}

func (l Language) Text(english, japanese string) string {
	if l == English {
		return english
	}
	return japanese
}
