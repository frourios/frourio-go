package api

type FrourioSpec struct {
	Get struct {
		Res struct {
			Status200 struct {
				Body string `json:"body" validate:"required"`
			}
		}
	}
}
