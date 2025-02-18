package esco

type CuentaSchema struct {
	SC []interface{} `json:"SC"`
	FI string        `json:"FI"`
	BY bool          `json:"BY"`
	RF bool          `json:"RF"`
	AT bool          `json:"AT"`
	CH bool          `json:"CH"`
	ID string        `json:"ID"`
	D  string        `json:"D"`
	N  int           `json:"N"`
	F  string        `json:"F"`
	EF bool          `json:"EF"`
}

type DetalleSchema struct {
	TIT  string      `json:"TIT"`
	DESC interface{} `json:"DESC"`
}

type MercadoSchema struct {
	MERC string `json:"MERC"`
	NUM  string `json:"NUM"`
}

type CuentaDetalleSchema struct {
	DET  []DetalleSchema `json:"DET"`
	MERC []MercadoSchema `json:"MERC"`
}

type EstadoCuentaSchema struct {
	CA             string      `json:"CA"`
	CI             string      `json:"CI"`
	A              string      `json:"A"`
	D              string      `json:"D"`
	TI             string      `json:"TI"`
	C              float64     `json:"C"`
	CD             float64     `json:"CD"`
	CND            interface{} `json:"CND"`
	F              string      `json:"F"`
	PR             float64     `json:"PR"`
	PRS            string      `json:"PR_S"`
	N              float64     `json:"N"`
	NS             string      `json:"N_S"`
	PC             float64     `json:"PC"`
	M              string      `json:"M"`
	MS             string      `json:"MS"`
	G              string      `json:"G"`
	CTA            float64     `json:"CTA"`
	CGAR           interface{} `json:"C_GAR"`
	CVENC          interface{} `json:"C_VENC"`
	C24            interface{} `json:"C_24"`
	C48            interface{} `json:"C_48"`
	VALGAR         interface{} `json:"VAL_GAR"`
	VALVENC        interface{} `json:"VAL_VENC"`
	VAL24          interface{} `json:"VAL_24"`
	VAL48          interface{} `json:"VAL_48"`
	VALFUT         interface{} `json:"VAL_FUT"`
	VALACUM        interface{} `json:"VAL_ACUM"`
	VALFUTT        interface{} `json:"VAL_FUT_T"`
	PRE            float64     `json:"PR_E"`
	PRERS          float64     `json:"PR_E_RS"`
	NE             float64     `json:"N_E"`
	NERS           float64     `json:"N_E_RS"`
	T              string      `json:"T"`
	PO             string      `json:"PO"`
	TB             interface{} `json:"TB"`
	TO             string      `json:"TO"`
	PPP            interface{} `json:"PPP"`
	MSI            string      `json:"MSI"`
	FDOD           interface{} `json:"FDO_D"`
	FDOA           interface{} `json:"FDO_A"`
	FDODC          interface{} `json:"FDO_CD"`
	FDOC           interface{} `json:"FDO_CA"`
	PPPRendimiento interface{} `json:"PPPRendimiento"`
	PPPPorcentaje  interface{} `json:"PPPPorcentaje"`
}

type Liquidacion struct {
	CFM  float64  `json:"CFM"`
	C    float64  `json:"C"`
	N    float64  `json:"N"`
	NFM  float64  `json:"NFM"`
	S    float64  `json:"S"`
	R    *float64 `json:"R"`
	FC   string   `json:"FC"`
	FL   string   `json:"FL"`
	F    string   `json:"F"`
	Q    float64  `json:"Q"`
	TO   string   `json:"TO"`
	VC   float64  `json:"VC"`
	I    float64  `json:"I"`
	MS   string   `json:"MS"`
	MSF  string   `json:"MSF"`
	CF   string   `json:"CF"`
	IDL  string   `json:"IDL"`
	CODF int      `json:"CODF"`
	CA   string   `json:"CA"`
	CD   string   `json:"CD"`
	FA   string   `json:"FA"`
	FD   string   `json:"FD"`
}

type Boleto struct {
	T   string   `json:"T"`
	I   string   `json:"I"`
	NRO int      `json:"NRO"`
	F   string   `json:"F"`
	FL  string   `json:"FL"`
	O   string   `json:"O"`
	C   float64  `json:"C"`
	PR  float64  `json:"PR"`
	B   float64  `json:"B"`
	A   *float64 `json:"A"`
	DB  float64  `json:"DB"`
	DM  float64  `json:"DM"`
	MS  string   `json:"MS"`
	N   float64  `json:"N"`
	AS  string   `json:"A_S"`
	PRS string   `json:"PR_S"`
	BS  string   `json:"B_S"`
	DBS string   `json:"DB_S"`
	DMS string   `json:"DM_S"`
	NS  string   `json:"N_S"`
	ID  string   `json:"ID"`
	TM  string   `json:"TM"`
	ORI string   `json:"ORI"`
}

type Instrumentos struct {
	ID     int      `json:"ID"`
	SA     bool     `json:"SA"`
	F      *string  `json:"F"`
	FL     *string  `json:"FL"`
	FE     *string  `json:"FE"`
	I      string   `json:"I"`
	D      string   `json:"D"`
	C      *float64 `json:"C"`
	PR     *float64 `json:"PR"`
	PR_S   *string  `json:"PR_S"`
	N      *float64 `json:"N"`
	S      float64  `json:"S"`
	TC     string   `json:"TC"`
	N_S    *string  `json:"N_S"`
	S_S    *string  `json:"S_S"`
	M      *string  `json:"M"`
	TA     *string  `json:"TA"`
	TPCP   *string  `json:"TPCP"`
	CTCNCD *string  `json:"CTCNCD"`
}
