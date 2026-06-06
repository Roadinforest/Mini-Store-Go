package valueobject

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type JSON[T any] struct {
	Data  T
	Valid bool
}

func NewJSON[T any](value T) JSON[T] {
	return JSON[T]{
		Data:  value,
		Valid: true,
	}
}

func (j JSON[T]) MarshalJSON() ([]byte, error) {
	if !j.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(j.Value)
}

func (j *JSON[T]) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		var zero T
		j.Data = zero
		j.Valid = false
		return nil
	}

	if err := json.Unmarshal(data, &j.Data); err != nil {
		return err
	}
	j.Valid = true
	return nil
}

func (j JSON[T]) ValueDriver() (driver.Value, error) {
	if !j.Valid {
		return nil, nil
	}
	return json.Marshal(j.Data)
}

func (j JSON[T]) Value() (driver.Value, error) {
	return j.ValueDriver()
}

func (j *JSON[T]) Scan(src interface{}) error {
	if src == nil {
		var zero T
		j.Data = zero
		j.Valid = false
		return nil
	}

	var data []byte
	switch typed := src.(type) {
	case []byte:
		data = typed
	case string:
		data = []byte(typed)
	default:
		return fmt.Errorf("unsupported json scan type %T", src)
	}

	if len(data) == 0 {
		var zero T
		j.Data = zero
		j.Valid = false
		return nil
	}

	if err := json.Unmarshal(data, &j.Data); err != nil {
		return err
	}
	j.Valid = true
	return nil
}
