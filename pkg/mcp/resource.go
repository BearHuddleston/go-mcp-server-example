package mcp

// Resource-related types
type Resource struct {
	URI  string `json:"uri"`
	Name string `json:"name"`
}

type ResourceContent struct {
	URI  string `json:"uri"`
	Text string `json:"text"`
}

type ResourceResponse struct {
	Contents []ResourceContent `json:"contents"`
}

type ResourceParams struct {
	URI string `json:"uri"`
}