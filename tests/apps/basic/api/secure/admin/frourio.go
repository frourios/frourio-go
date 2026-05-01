package admin

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
				Body string `json:"body" validate:"required"`
			}
		}
	}
	Post struct {
		Body struct {
			Data string `json:"data" validate:"required"`
		}
		Res struct {
			Status201 struct {
				Body string `json:"body" validate:"required"`
			}
			Status403 struct {
				Body string `json:"body" validate:"required"`
			}
		}
	}
}
