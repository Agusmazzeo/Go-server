package utils

const ShortSlashDateLayout = "2006/01/02"
const ShortDashDateLayout = "2006-01-02"

const (
	A3500ID            = "5"
	MonthlyInflationID = "27"
)

const (
	AssetCurrencyPesos   = "Pesos"
	AssetCurrencyDolares = "USD"
)

// ChartColors defines a palette of distinct colors for chart visualization
// These colors are designed to be easily distinguishable from each other
// Using a professional report color palette for business and financial reports
var ChartColors = []string{
	"#ffa366", // Light Orange
	"#ff8080", // Light Red
	"#80b3ff", // Light Blue
	"#a3d977", // Light Green
	"#c285ff", // Light Purple
	"#80e6d4", // Light Teal
	"#ffb366", // Medium Orange
	"#ff6666", // Medium Red
	"#80b366", // Medium Green
	"#e680ff", // Light Magenta
	"#808080", // Medium Gray
	"#b3a3ff", // Light Slate Blue
	"#80d4cc", // Light Sea Green
}

// GetChartColor returns a color from the chart color palette
// If the index exceeds the palette size, it cycles back to the beginning
func GetChartColor(index int) string {
	return ChartColors[index%len(ChartColors)]
}
