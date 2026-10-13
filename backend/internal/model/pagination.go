package model

type PaginationRequest struct {
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

type PaginationResponse struct {
	Items   interface{} `json:"items"`
	Total   int         `json:"total"`
	Offset  int         `json:"offset"`
	Limit   int         `json:"limit"`
	HasMore bool        `json:"has_more"`
}
