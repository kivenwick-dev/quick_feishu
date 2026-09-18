package model

type DictAccountField struct {
	ID          uint   `gorm:"primaryKey"`
	FieldPath   string `gorm:"index"`
	Label       string
	FieldType   string
	Description string
	IsDefault   bool
}

type DictTokenField struct {
	ID          uint   `gorm:"primaryKey"`
	FieldPath   string `gorm:"index"`
	Label       string
	FieldType   string
	Description string
	IsDefault   bool
}

type DictUsageField struct {
	ID          uint   `gorm:"primaryKey"`
	FieldPath   string `gorm:"index"`
	Label       string
	FieldType   string
	Description string
	IsDefault   bool
}
