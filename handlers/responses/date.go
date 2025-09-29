package responses

import (
	"github.com/juho05/crossonic-server/repos"
	"github.com/juho05/crossonic-server/util"
)

type Date struct {
	Year  *int `xml:"year,attr,omitempty" json:"year,omitempty"`
	Month *int `xml:"month,attr,omitempty" json:"month,omitempty"`
	Day   *int `xml:"day,attr,omitempty" json:"day,omitempty"`
}

func NewDate(date *repos.Date) *Date {
	if date == nil {
		return nil
	}
	return &Date{
		Year:  util.ToPtr(date.Year()),
		Month: date.Month(),
		Day:   date.Day(),
	}
}
