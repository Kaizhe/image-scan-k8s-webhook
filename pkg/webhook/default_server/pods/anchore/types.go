package anchore

type Check struct {
	LastEvaluation string `json:"last_evaluation"`
	PolicyId       string `json:"policy_id"`
	Status         string `json:"status"`
}

type Image struct {
	ImageDigest    string        `json:"imageDigest"`
	ImageStatus    string        `json:"image_status"`
	AnalysisStatus string        `json:"analysis_status"`
	ImageDetails   []ImageDetail `json:"image_detail"`
}

type ImageDetail struct {
	Digest      string `json:"digest"`
	FullDigetst string `json:"fulldigest"`
	FullTag     string `json:"fulltag"`
	Repo        string `json:"repo"`
	Tag         string `json:"Repo"`
	Registry    string `json:"registry"`
}

type SHAResult struct {
	Status string
}

type AnchoreConfig struct {
	EndpointURL string `yaml:"ANCHORE_CLI_URL"`
	User        string `yaml:"ANCHORE_CLI_USER"`
	Password    string `yaml:"ANCHORE_CLI_PASS"`
}
