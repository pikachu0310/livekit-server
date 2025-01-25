package util

func BoolPtr(b bool) *bool {
	return &b
}

// Metadataに収容されるJSONの構造体
type Metadata struct {
	// ルームのメタデータ
	Status string `json:"status"`

	// webinarかどうか
	IsWebinar bool `json:"isWebinar"`
}
