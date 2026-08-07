package models

type Country struct {
	ID   int64  `json:"id" db:"id"`
	Name string `json:"name" db:"nom"`
	Code string `json:"code,omitempty" db:"code"`
}
