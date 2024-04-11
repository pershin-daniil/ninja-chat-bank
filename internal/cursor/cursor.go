package cursor

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

func Encode(data any) (string, error) {
	bytes, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshall data: %v", err)
	}

	return base64.URLEncoding.EncodeToString(bytes), nil
}

func Decode(in string, to any) error {
	bytes, err := base64.URLEncoding.DecodeString(in)
	if err != nil {
		return fmt.Errorf("failed to decode %s: %v", in, err)
	}

	if err = json.Unmarshal(bytes, to); err != nil {
		return fmt.Errorf("failed to unmarshal json: %v", err)
	}

	return nil
}
