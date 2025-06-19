package models

type User struct {
	ID       string   `json:"id" structs:"id"`
	Username string   `json:"username" structs:"username"`
	Email    string   `json:"email" structs:"email"`
	Roles    []string `json:"roles" structs:"roles"`
	Scopes   []string `json:"scopes" structs:"scopes"`
}
