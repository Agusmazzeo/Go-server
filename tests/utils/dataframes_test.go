//nolint:depguard
package utils_test

import (
	"server/src/utils"
	"strings"
	"testing"

	"github.com/go-gota/gota/dataframe"
)

func TestUnionDataFramesByIndex(t *testing.T) {
	tests := []struct {
		name      string
		csv1      string
		csv2      string
		indexCol  string
		wantRows  int
		wantCols  []string
		wantIndex map[string]bool
	}{
		{
			name: "basic union with unique ids",
			csv1: `id,name
1,Alice
2,Bob`,
			csv2: `id,name
3,Charlie
4,Diana`,
			indexCol:  "id",
			wantRows:  4,
			wantCols:  []string{"id", "name"},
			wantIndex: map[string]bool{"1": true, "2": true, "3": true, "4": true},
		},
		{
			name: "overlapping ids, keep first",
			csv1: `id,name
1,Alice
2,Bob`,
			csv2: `id,name
2,Robert
3,Charlie`,
			indexCol:  "id",
			wantRows:  3,
			wantCols:  []string{"id", "name"},
			wantIndex: map[string]bool{"1": true, "2": true, "3": true}, // 2 from df1
		},
		{
			name: "different columns, auto-filled",
			csv1: `id,name
1,Alice`,
			csv2: `id,score
2,99`,
			indexCol:  "id",
			wantRows:  2,
			wantCols:  []string{"id", "name", "score"},
			wantIndex: map[string]bool{"1": true, "2": true},
		},
		{
			name: "completely overlapping ids, deduplicated",
			csv1: `id,name
1,Alice`,
			csv2: `id,name
1,Duplicate`,
			indexCol:  "id",
			wantRows:  1,
			wantCols:  []string{"id", "name"},
			wantIndex: map[string]bool{"1": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			df1 := dataframe.ReadCSV(strings.NewReader(tt.csv1))
			df2 := dataframe.ReadCSV(strings.NewReader(tt.csv2))

			result := utils.UnionDataFramesByIndex(df1, df2, tt.indexCol)

			if result.Nrow() != tt.wantRows {
				t.Errorf("expected %d rows, got %d", tt.wantRows, result.Nrow())
			}

			cols := result.Names()
			for _, wantCol := range tt.wantCols {
				found := false
				for _, c := range cols {
					if c == wantCol {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected column %q in result, but not found", wantCol)
				}
			}

			// Check for presence of all expected index values
			idCol := result.Col(tt.indexCol)
			for i := 0; i < idCol.Len(); i++ {
				val := idCol.Elem(i).String()
				if _, ok := tt.wantIndex[val]; !ok {
					t.Errorf("unexpected id %v found in result", val)
				}
			}
		})
	}
}
