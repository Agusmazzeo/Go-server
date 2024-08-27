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
