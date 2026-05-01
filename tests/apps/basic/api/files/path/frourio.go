package path

type FrourioSpec struct {
	Get struct {
		Param *[]string
		Res   struct {
			Status200 struct {
				Body string `json:"body" validate:"required"`
			}
		}
	}
}
