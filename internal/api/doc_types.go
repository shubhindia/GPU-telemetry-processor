package api

type GPUDoc struct {
	ID        string `json:"id"`
	GPUID     string `json:"gpu_id"`
	Device    string `json:"device"`
	UUID      string `json:"uuid"`
	ModelName string `json:"modelName"`
	Hostname  string `json:"Hostname"`
}

type TelemetrySampleDoc struct {
	Timestamp  string  `json:"timestamp"`
	MetricName string  `json:"metric_name"`
	GPUID      string  `json:"gpu_id"`
	Device     string  `json:"device"`
	UUID       string  `json:"uuid"`
	ModelName  string  `json:"modelName"`
	Hostname   string  `json:"Hostname"`
	Container  string  `json:"container"`
	Pod        string  `json:"pod"`
	Namespace  string  `json:"namespace"`
	Value      float64 `json:"value"`
	LabelsRaw  string  `json:"labels_raw"`
}

type GPUListDocResponse struct {
	Items []GPUDoc `json:"items"`
}

type GPUTelemetryDocResponse struct {
	GPUId string               `json:"gpu_id"`
	Items []TelemetrySampleDoc `json:"items"`
}

type TelemetryQueryDocResponse struct {
	Items []TelemetrySampleDoc `json:"items"`
}

type HealthResponse struct {
	Status string `json:"status"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
