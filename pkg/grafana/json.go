package grafana

import (
	"bytes"
	"encoding/json"
	"errors"
)

type jsonObj struct {
	data interface{}
}

func newJsonObj(body []byte) (*jsonObj, error) {
	j := &jsonObj{
		data: make(map[string]interface{}),
	}

	err := j.unmarshalJSON(body)
	if err != nil {
		return nil, err
	}
	return j, nil
}

func (j *jsonObj) unmarshalJSON(p []byte) error {
	dec := json.NewDecoder(bytes.NewBuffer(p))
	dec.UseNumber()
	return dec.Decode(&j.data)
}

func (j *jsonObj) marshalJSON() ([]byte, error) {
	return json.Marshal(&j.data)
}

func (j *jsonObj) get(key string) *jsonObj {
	m, err := j.asMap()
	if err == nil {
		if val, ok := m[key]; ok {
			return &jsonObj{val}
		}
	}
	return &jsonObj{nil}
}

func (j *jsonObj) set(key string, val interface{}) {
	m, err := j.asMap()
	if err != nil {
		return
	}
	m[key] = val
}

func (j *jsonObj) asMap() (map[string]interface{}, error) {
	if m, ok := (j.data).(map[string]interface{}); ok {
		return m, nil
	}
	return nil, errors.New("type assertion to map[string]interface{} failed")
}

func (j *jsonObj) asArray() ([]interface{}, error) {
	if a, ok := (j.data).([]interface{}); ok {
		return a, nil
	}
	return nil, errors.New("type assertion to []interface{} failed")
}

func (j *jsonObj) String() (string, error) {
	if s, ok := (j.data).(string); ok {
		return s, nil
	}
	return "", errors.New("type assertion to string failed")
}
