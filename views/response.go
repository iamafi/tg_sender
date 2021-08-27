package views

// R general structure view for HTTP json response
type R struct {
	Status    string      `json:"status"`     // {"status",""}
	ErrorCode int         `json:"error_code"` // {"error_code",""}
	ErrorNote string      `json:"error_note"` // {"error_note",""}
	Data      interface{} `json:"data"`       // {"data",""}
}

