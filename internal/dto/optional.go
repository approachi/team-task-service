package dto

import "encoding/json"

// OptionalString distinguishes "field absent" (Set=false) from "field
// explicitly null" (Set=true, Value=nil) from "field has a value" — needed
// for partial updates (PUT /tasks/{id}) where only supplied fields should
// be changed.
type OptionalString struct {
	Value *string
	Set   bool
}

func (o *OptionalString) UnmarshalJSON(data []byte) error {
	o.Set = true
	if string(data) == "null" {
		o.Value = nil
		return nil
	}
	var v string
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	o.Value = &v
	return nil
}

// OptionalInt64 is the int64 counterpart of OptionalString, used for
// assignee_to (nullable — null means "unassign").
type OptionalInt64 struct {
	Value *int64
	Set   bool
}

func (o *OptionalInt64) UnmarshalJSON(data []byte) error {
	o.Set = true
	if string(data) == "null" {
		o.Value = nil
		return nil
	}
	var v int64
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	o.Value = &v
	return nil
}
