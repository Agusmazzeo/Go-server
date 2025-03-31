package utils

//nolint:depguard
import (
	"sort"

	"github.com/go-gota/gota/dataframe"
	"github.com/go-gota/gota/series"
)

// UnionDataFramesByIndex merges df1 and df2 by indexCol, coalescing fields and sorting by indexCol.
func UnionDataFramesByIndex(df1, df2 dataframe.DataFrame, indexCol string) dataframe.DataFrame {
	// Gather all unique columns
	allColsMap := map[string]bool{}
	for _, col := range df1.Names() {
		allColsMap[col] = true
	}
	for _, col := range df2.Names() {
		allColsMap[col] = true
	}
	var allCols []string
	for col := range allColsMap {
		allCols = append(allCols, col)
	}

	df1 = ensureColumns(df1, allCols)
	df2 = ensureColumns(df2, allCols)

	// Index rows from both frames by indexCol
	df1Rows := indexRowsByColumn(df1, indexCol)
	df2Rows := indexRowsByColumn(df2, indexCol)

	// Merge rows
	mergedRows := map[string]map[string]interface{}{}
	indexKeys := map[string]bool{}

	// Combine df1 and df2
	for k, row := range df1Rows {
		kStr, _ := k.(string)
		merged := make(map[string]interface{})
		for _, col := range allCols {
			merged[col] = row[col]
		}
		mergedRows[kStr] = merged
		indexKeys[kStr] = true
	}
	for k, row := range df2Rows {
		kStr, _ := k.(string)
		if existing, ok := mergedRows[kStr]; ok {
			for col, val := range row {
				if val != nil && val != "" {
					existing[col] = val // fill missing values from df2
				}
			}
		} else {
			mergedRows[kStr] = row
		}
		indexKeys[kStr] = true
	}

	// Sort index keys
	var sortedKeys []string
	for k := range indexKeys {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)

	// Reorder columns: index first
	finalCols := []string{indexCol}
	for _, col := range allCols {
		if col != indexCol {
			finalCols = append(finalCols, col)
		}
	}

	// Build final DataFrame
	colSeries := make([]series.Series, len(finalCols))
	for i, col := range finalCols {
		colData := make([]interface{}, len(sortedKeys))
		for j, key := range sortedKeys {
			colData[j] = mergedRows[key][col]
		}
		colSeries[i] = series.New(colData, series.String, col)
	}

	return dataframe.New(colSeries...)
}

// ensureColumns adds missing columns to the DataFrame with nil values
func ensureColumns(df dataframe.DataFrame, columns []string) dataframe.DataFrame {
	for _, col := range columns {
		if !hasCol(df, col) {
			empty := make([]interface{}, df.Nrow())
			df = df.Mutate(series.New(empty, series.String, col))
		}
	}
	return df
}

// hasCol checks whether a DataFrame contains a given column
func hasCol(df dataframe.DataFrame, colName string) bool {
	for _, name := range df.Names() {
		if name == colName {
			return true
		}
	}
	return false
}

// indexRowsByColumn builds a map from indexCol -> row (as map)
func indexRowsByColumn(df dataframe.DataFrame, indexCol string) map[interface{}]map[string]interface{} {
	result := make(map[interface{}]map[string]interface{})
	for i := 0; i < df.Nrow(); i++ {
		row := map[string]interface{}{}
		for _, col := range df.Names() {
			row[col] = df.Col(col).Elem(i).Val()
		}
		idx := df.Col(indexCol).Elem(i).Val()
		result[idx] = row
	}
	return result
}
