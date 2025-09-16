package repos

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type DurationMS time.Duration

func NewDurationMS(millis int64) DurationMS {
	return DurationMS(time.Duration(millis) * time.Millisecond)
}

//goland:noinspection GoMixedReceiverTypes
func (nt DurationMS) Millis() int64 {
	return nt.ToStd().Milliseconds()
}

//goland:noinspection GoMixedReceiverTypes
func (nt DurationMS) Seconds() int {
	return int(nt.ToStd().Seconds())
}

//goland:noinspection GoMixedReceiverTypes
func (nt DurationMS) ToStd() time.Duration {
	return time.Duration(nt)
}

//goland:noinspection GoMixedReceiverTypes
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

//goland:noinspection GoMixedReceiverTypes
func (nt DurationMS) Value() (driver.Value, error) {
	return time.Duration(nt).Milliseconds(), nil
}

type NullDurationMS struct {
	Duration DurationMS
	Valid    bool
}

//goland:noinspection GoMixedReceiverTypes
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

//goland:noinspection GoMixedReceiverTypes
func (nt NullDurationMS) Value() (driver.Value, error) {
	if !nt.Valid {
		return nil, nil
	}
	return time.Duration(nt.Duration).Milliseconds(), nil
}

type StringList []string

//goland:noinspection GoMixedReceiverTypes
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

//goland:noinspection GoMixedReceiverTypes
func (nt StringList) Value() (driver.Value, error) {
	return strings.Join(nt, "\003"), nil
}

type Map[T comparable, U any] map[T]U

//goland:noinspection GoMixedReceiverTypes
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

//goland:noinspection GoMixedReceiverTypes
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
