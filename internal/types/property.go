package types

// PropertyValue represents a ZFS property with a value field.
type PropertyValue struct {
	Value  string      `json:"value"`
	Source string      `json:"source"`
	Parsed interface{} `json:"parsed"`
}

// PropertyRawVal represents a ZFS property with a rawvalue field.
type PropertyRawVal struct {
	Value    string      `json:"value"`
	Rawvalue string      `json:"rawvalue"`
	Source   string      `json:"source"`
	Parsed   interface{} `json:"parsed"`
}
