package schemas

type CurrenciesResponse struct {
	Currencies []Currency
}

type Currency struct {
	ID          string
	Description string
}

type CurrencyWithValuationResponse struct {
	ID          string
	Description string
	Valuations  []CurrencyValuation
}

type CurrencyValuation struct {
	Date                string
	ArgCurrencyRelation float32
	UsdCurrencyRelation float32
}

type VariablesResponse struct {
	Variables []Variable
}

type Variable struct {
	ID          string
	Description string
}

type VariableWithValuationResponse struct {
	ID          string
	Description string
	Valuations  []VariableValuation
}

type VariableValuation struct {
	Date  string
	Value float64
}
