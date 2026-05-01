package users

type FrourioSpec struct {
	Get struct {
		Query struct {
			Role *string `json:"role"`
		}
		Res struct {
			Status200 struct {
				Body string `json:"body" validate:"required"`
			}
		}
	}
}
