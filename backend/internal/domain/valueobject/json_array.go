package valueobject

import (
	"database/sql/driver"
	"encoding/json"

	"github.com/lib/pq"
)

type JSONArray[T any] struct {
	Data  []T
	Valid bool
}

func NewJSONArray[T any](value []T) JSONArray[T] {
	if value == nil {
		value = []T{}
	}
	return JSONArray[T]{
		Data:  value,
		Valid: true,
	}
}

func (j JSONArray[T]) MarshalJSON() ([]byte, error) {
	if !j.Valid {
		return []byte("[]"), nil
	}
	return json.Marshal(j.Data)
}

func (j *JSONArray[T]) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		j.Data = []T{}
		j.Valid = false
		return nil
	}

	if err := json.Unmarshal(data, &j.Data); err != nil {
		return err
	}
	j.Valid = true
	return nil
}

func (j JSONArray[T]) Value() (driver.Value, error) {
	if !j.Valid {
		return pq.StringArray{}.Value()
	}

	encoded := make([]string, 0, len(j.Data))
	for _, item := range j.Data {
		raw, err := json.Marshal(item)
		if err != nil {
			return nil, err
		}
		encoded = append(encoded, string(raw))
	}

	return pq.StringArray(encoded).Value()
}

func (j *JSONArray[T]) Scan(src interface{}) error {
	if src == nil {
		j.Data = []T{}
		j.Valid = false
		return nil
	}

	var encoded pq.StringArray
	if err := encoded.Scan(src); err != nil {
		return err
	}

	items := make([]T, 0, len(encoded))
	for _, value := range encoded {
		var item T
		if err := json.Unmarshal([]byte(value), &item); err != nil {
			return err
		}
		items = append(items, item)
	}

	j.Data = items
	j.Valid = true
	return nil
}
