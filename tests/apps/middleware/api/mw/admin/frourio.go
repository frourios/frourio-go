package admin

type ForbiddenBody struct {
	Message string `json:"message" validate:"required"`
}

type AdminContextResponse struct {
	UserID      *string  `json:"userId,omitempty"`
	TraceID     string   `json:"traceId"`
	IsAdmin     bool     `json:"isAdmin"`
	Permissions []string `json:"permissions"`
}

type AdminPostBody struct {
	Data string `json:"data" validate:"required"`
}

type AdminPostResponseBody struct {
	Received string               `json:"received"`
	Context  AdminContextResponse `json:"context"`
}

type FrourioSpec struct {
	Middleware struct {
		All struct {
			Context struct {
				IsAdmin     bool     `json:"isAdmin"`
				Permissions []string `json:"permissions"`
			}
		}
		Post bool
	}
	Get struct {
		Res struct {
			Status200 struct {
				Body AdminContextResponse
			}
		}
	}
	Post struct {
		Body AdminPostBody
		Res  struct {
			Status201 struct {
				Body AdminPostResponseBody
			}
			Status403 struct {
				Body ForbiddenBody
			}
		}
	}
}
