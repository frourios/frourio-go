package users

import "github.com/frourios/frourio-go/tests/apps/middleware/api/mw/admin"

type UsersResponseBody struct {
	Context admin.AdminContextResponse `json:"context"`
	Users   []string                   `json:"users"`
}

type FrourioSpec struct {
	Get struct {
		Query struct {
			Role *string `json:"role"`
		}
		Res struct {
			Status200 struct {
				Body UsersResponseBody
			}
		}
	}
}
