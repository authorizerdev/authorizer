package types

import "encoding/json"

// Type for interface slice. Used for redis store.
type InterfaceSlice []interface{}

// MarshalBinary for interface slice.
func (s InterfaceSlice) MarshalBinary() ([]byte, error) {
	return json.Marshal(s)
}

// UnmarshalBinary for interface slice.
func (s *InterfaceSlice) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, s)
}
