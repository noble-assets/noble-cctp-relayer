package types

type AccountResp struct {
	Account `json:"account"`
}
type Account struct {
	Type          string `json:"@type"`
	Address       string `json:"address"`
	AccountNumber string `json:"account_number"`
	Sequence      string `json:"sequence"`
}
