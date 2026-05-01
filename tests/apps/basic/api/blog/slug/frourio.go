package slug

type FrourioSpec struct {
	Get struct {
		Param []string `validate:"required"`
		Res   struct {
			Status200 struct {
				Body string `json:"body" validate:"required"`
			}
		}
	}
}
