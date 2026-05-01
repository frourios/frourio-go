package nest

type FrourioSpec struct {
	Middleware struct {
		All struct {
			Context struct {
				TraceID string `json:"traceId" validate:"required"`
			}
		}
	}
}
