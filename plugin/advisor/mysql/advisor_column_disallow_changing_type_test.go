package mysql

import (
	"testing"

	"github.com/bytebase/bytebase/plugin/advisor"
)

func TestColumnDisallowChangingType(t *testing.T) {
	tests := []advisor.TestCase{
		{
			Statement: ``,
			Want: []advisor.Advice{
				{
					Status:  advisor.Success,
					Code:    advisor.Ok,
					Title:   "OK",
					Content: "",
				},
			},
		},
	}

	advisor.RunSQLReviewRuleTests(t, tests, &ColumnDisallowChangingTypeAdvisor{}, &advisor.SQLReviewRule{
		Level:   advisor.SchemaRuleLevelWarning,
		Payload: "",
	}, advisor.MockMySQLDatabase)
}
