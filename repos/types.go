package repos

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"time"
)

type DurationMS time.Duration

func NewDurationMS(millis int64) DurationMS {
	return DurationMS(time.Duration(millis) * time.Millisecond)
}

func NewDurationMSFromStd(d time.Duration) DurationMS {
	return DurationMS(d)
}

func (nt DurationMS) Millis() int64 {
	return nt.ToStd().Milliseconds()
}

func (nt DurationMS) Seconds() int {
	return int(nt.ToStd().Seconds())
}

func (nt DurationMS) ToStd() time.Duration {
	return time.Duration(nt)
}

func (nt *DurationMS) Scan(value any) error {
	if value == nil {
		return nil
	}
	var milliseconds int64
	switch value := value.(type) {
	case int:
		milliseconds = int64(value)
	case int32:
		milliseconds = int64(value)
	case int64:
		milliseconds = value
	default:
		return fmt.Errorf("cannot scan %T into DurationMS; expected integer value", value)
	}
	*nt = DurationMS(milliseconds) * DurationMS(time.Millisecond)
	return nil
}

func (nt DurationMS) Value() (driver.Value, error) {
	return time.Duration(nt).Milliseconds(), nil
}

type NullDurationMS struct {
	Duration DurationMS
	Valid    bool
}

func (nt *NullDurationMS) Scan(value any) error {
	if value == nil {
		return nil
	}
	var milliseconds int64
	switch value := value.(type) {
	case int:
		milliseconds = int64(value)
	case int32:
		milliseconds = int64(value)
	case int64:
		milliseconds = value
	default:
		return fmt.Errorf("cannot scan %T into DurationMS; expected integer value", value)
	}
	*nt = NullDurationMS{
		Duration: DurationMS(milliseconds) * DurationMS(time.Millisecond),
		Valid:    true,
	}
	return nil
}

func (nt NullDurationMS) Value() (driver.Value, error) {
	if !nt.Valid {
		return nil, nil
	}
	return time.Duration(nt.Duration).Milliseconds(), nil
}

type StringList []string

func (nt *StringList) Scan(value interface{}) error {
	if value == nil {
		*nt = nil
		return nil
	}
	raw, ok := value.(string)
	if !ok {
		return fmt.Errorf("cannot scan %T into StringList; expected string value", value)
	}
	*nt = strings.Split(raw, "\003")
	return nil
}

func (nt StringList) Value() (driver.Value, error) {
	return strings.Join(nt, "\003"), nil
}
