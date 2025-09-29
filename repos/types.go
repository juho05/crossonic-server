package repos

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Date struct {
	year  int
	month int
	day   int
}

func (d Date) Year() int {
	return d.year
}

func (d Date) Month() *int {
	if d.month == 0 {
		return nil
	}
	return &d.month
}

func (d Date) Day() *int {
	if d.day == 0 {
		return nil
	}
	return &d.day
}

func (d Date) String() string {
	if d.month == 0 {
		return fmt.Sprintf("%04d", d.year)
	}
	if d.day == 0 {
		return fmt.Sprintf("%04d-%02d", d.year, d.month)
	}
	return fmt.Sprintf("%04d-%02d-%02d", d.year, d.month, d.day)
}

func (d Date) Value() (driver.Value, error) {
	return d.String(), nil
}

func (d *Date) Scan(value any) error {
	if value == nil {
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("cannot scan %T into Date; expected string value", value)
	}
	date, err := ParseDate(str)
	if err != nil {
		return err
	}
	*d = date
	return nil
}

func NewDate(year int, month *int, day *int) Date {
	var m int
	if month != nil {
		m = *month
	}
	var d int
	if day != nil {
		d = *day
	}
	return Date{
		year:  year,
		month: m,
		day:   d,
	}
}

func ParseDate(str string) (Date, error) {
	parts := strings.Split(str, "-")
	if len(parts) > 3 {
		return Date{}, ErrInvalidDate
	}

	year, err := strconv.Atoi(parts[0])
	if err != nil {
		return Date{}, ErrInvalidDate
	}

	if len(parts) == 1 {
		return Date{
			year: year,
		}, nil
	}

	month, err := strconv.Atoi(parts[1])
	if err != nil || month < 1 || month > 12 {
		return Date{}, ErrInvalidDate
	}

	if len(parts) == 2 {
		return Date{
			year:  year,
			month: month,
		}, nil
	}

	day, err := strconv.Atoi(parts[2])
	if err != nil || day < 1 || day > 31 {
		return Date{}, ErrInvalidDate
	}

	return Date{
		year:  year,
		month: month,
		day:   day,
	}, nil
}

type DurationMS time.Duration

func NewDurationMS(millis int64) DurationMS {
	return DurationMS(time.Duration(millis) * time.Millisecond)
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
	strs := strings.Split(raw, "\003")
	if len(strs) == 1 && strs[0] == "" {
		*nt = nil
		return nil
	}
	*nt = strs
	return nil
}

func (nt StringList) Value() (driver.Value, error) {
	return strings.Join(nt, "\003"), nil
}

type Map[T comparable, U any] map[T]U

func (m *Map[T, U]) Scan(value interface{}) error {
	if value == nil {
		*m = nil
		return nil
	}
	raw, ok := value.(string)
	if !ok {
		return fmt.Errorf("cannot scan %T into Map; expected string value", value)
	}

	newM := make(map[T]U)
	err := json.Unmarshal([]byte(raw), &newM)
	if err != nil {
		return fmt.Errorf("cannot scan %T into Map; expected JSON string value: %w", value, err)
	}

	*m = newM
	return nil
}

func (m Map[T, U]) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	bytes, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("marshal map: %w", err)
	}
	return string(bytes), nil
}
