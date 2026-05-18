package contracts

type Event struct {
	Name      string `json:"name"`
	Key       string `json:"key"`
	Body      []byte `json:"body"`
	Version   int64  `json:"version"`
	TraceID   string `json:"traceId"`
	CreatedAt string `json:"createdAt"`
}
