package schemas

type TokenRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type TokenResponse struct {
	AccessToken      string `json:"access_token"`
	TokenType        string `json:"token_type"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshToken     string `json:"refresh_token"`
	ClientID         string `json:"as:client_id"`
	SessionTimeLife  string `json:"sessionTimeLife"`
	UserName         string `json:"userName"`
	UserID           string `json:"userID"`
	UserType         string `json:"userType"`
	AccountLength    string `json:"accountLength"`
	AUSER            string `json:"AUSER"`
	FechaUltimoLogon string `json:"FechaUltimoLogon"`
	Issued           string `json:".issued"`
	Expires          string `json:".expires"`
}
