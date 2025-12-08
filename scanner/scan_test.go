package scanner

import (
	"testing"

	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/crossonic-server/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_parseDate(t *testing.T) {
	type args struct {
		str string
	}
	tests := []struct {
		name    string
		args    args
		want    repos.Date
		wantErr bool
	}{
		{"empty string", args{""}, repos.Date{}, true},
		{"year", args{"2025"}, repos.NewDate(2025, nil, nil), false},
		{"year with space", args{"   2025   \t  "}, repos.NewDate(2025, nil, nil), false},
		{"year with month", args{"  2025-11   "}, repos.NewDate(2025, util.ToPtr(11), nil), false},
		{"year-month-day", args{"  2025-11-03   "}, repos.NewDate(2025, util.ToPtr(11), util.ToPtr(3)), false},
		{"year-month-day no zeros", args{"  2025-2-3   "}, repos.NewDate(2025, util.ToPtr(2), util.ToPtr(3)), false},
		{"12/11/98", args{"12/11/98"}, repos.NewDate(1998, util.ToPtr(12), util.ToPtr(11)), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDate(tt.args.str)
			require.Equal(t, tt.wantErr, err != nil)
			assert.Equal(t, tt.want, got)
		})
	}
}
