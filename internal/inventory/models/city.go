package models

type City struct {
	ID      int64  `json:"id" db:"id"`
	Name    string `json:"name" db:"nom"`
	CountryID int64 `json:"country_id" db:"pays_id"`
}
