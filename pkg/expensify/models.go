package expensify

type User struct {
	Role      string `json:"role"`
	Email     string `json:"email"`
	SubmitsTo string `json:"submitsTo"`
}

type Policy struct {
	OutputCurrency string `json:"outputCurrency"`
	Owner          string `json:"owner"`
	Role           string `json:"role"`
	Name           string `json:"name"`
	ID             string `json:"id"`
	Type           string `json:"type"`
}
