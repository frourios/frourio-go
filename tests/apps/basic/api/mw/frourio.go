package mw

type FrourioSpec struct {
	Middleware struct {
		All bool
		Get struct {
			Context struct {
				Role string `json:"role" validate:"required"`
			}
		}
	}
	Get struct {
		Res struct {
			Status200 struct {
				Body string `json:"body" validate:"required"`
			}
			Status403 struct {
				Body string `json:"body" validate:"required"`
			}
		}
	}
}
