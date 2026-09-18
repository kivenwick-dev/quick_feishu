package model

type DictAccountField struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	FieldPath   string `gorm:"index" json:"field_path"`
	Label       string `json:"label"`
	FieldType   string `json:"field_type"`
	Description string `json:"description"`
	IsDefault   bool   `json:"is_default"`
}

type DictTokenField struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	FieldPath   string `gorm:"index" json:"field_path"`
	Label       string `json:"label"`
	FieldType   string `json:"field_type"`
	Description string `json:"description"`
	IsDefault   bool   `json:"is_default"`
}

type DictUsageField struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	FieldPath   string `gorm:"index" json:"field_path"`
	Label       string `json:"label"`
	FieldType   string `json:"field_type"`
	Description string `json:"description"`
	IsDefault   bool   `json:"is_default"`
}
