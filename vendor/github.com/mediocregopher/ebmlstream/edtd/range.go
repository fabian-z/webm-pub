package edtd

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

type rangeParam struct {
	loweri, upperi   int64
	lowerui, upperui uint64
	lowerf, upperf   float64
	exLower, exUpper bool // exclusive, only applies for float checking

	more *rangeParam
}

func parseRangeParams(typ Type, r []*token) (*rangeParam, error) {
	var f func(Type, *token) (*rangeParam, error)
	switch typ {
	case Int, String, Binary:
		f = parseIntRange
	case Uint:
		f = parseUintRange
	case Float:
		f = parseFloatRange
	default:
		return nil, fmt.Errorf("range on unsupported type")
	}

	var root, prev *rangeParam
	for i := range r {
		rp, err := f(typ, r[i])
		if err != nil {
			return nil, err
		}
		if root == nil {
			root = rp
		}
		if prev != nil {
			prev.more = rp
		}
		prev = rp
	}
	return root, nil
}

func parseIntRange(typ Type, r *token) (*rangeParam, error) {
	lower, upper := int64(math.MinInt64), int64(math.MaxInt64)
	i := strings.Index(r.val, "..")

	var lowers, uppers string
	var err error
	if i < 0 {
		lowers, uppers = r.val, r.val
	} else {
		lowers, uppers = r.val[:i], r.val[i+2:]
	}

	if lowers != "" {
		if lower, err = strconv.ParseInt(lowers, 10, 64); err != nil {
			return nil, err
		}
	}
	if uppers != "" {
		if upper, err = strconv.ParseInt(uppers, 10, 64); err != nil {
			return nil, err
		}
	}

	return &rangeParam{loweri: lower, upperi: upper}, nil
}

func parseUintRange(typ Type, r *token) (*rangeParam, error) {
	lower, upper := uint64(0), uint64(math.MaxUint64)
	i := strings.Index(r.val, "..")

	var lowers, uppers string
	var err error
	if i < 0 {
		lowers, uppers = r.val, r.val
	} else if i == 0 {
		return nil, fmt.Errorf("Invalid range '%s' for uint", r.val)
	} else {
		lowers, uppers = r.val[:i], r.val[i+2:]
	}

	if lowers != "" {
		if lower, err = strconv.ParseUint(lowers, 10, 64); err != nil {
			return nil, err
		}
	}
	if uppers != "" {
		if upper, err = strconv.ParseUint(uppers, 10, 64); err != nil {
			return nil, err
		}
	}

	return &rangeParam{lowerui: lower, upperui: upper}, nil
}

func parseFloatRange(typ Type, r *token) (*rangeParam, error) {
	lower, upper := -1*math.MaxFloat64, math.MaxFloat64
	var greaterThan, lessThan, equalTo bool
	var i int

	// We first check if the range is of the form: >=0.0

	if r.val[0] == '>' {
		greaterThan = true
		i++
	} else if r.val[0] == '<' {
		lessThan = true
		i++
	}
	if r.val[1] == '=' {
		equalTo = true
		i++
	}

	if greaterThan || lessThan {
		f, err := strconv.ParseFloat(r.val[i:], 64)
		if err != nil {
			return nil, err
		}
		if greaterThan {
			lower = f
		} else {
			upper = f
		}
		return &rangeParam{
			lowerf:  lower,
			upperf:  upper,
			exLower: !equalTo,
			exUpper: !equalTo,
		}, nil
	}

	// At this point the range ought to be of the form 0.0<=..<5.0

	i = strings.Index(r.val, "..")
	if i < 0 {
		return nil, fmt.Errorf("Invalid float range '%s'", r.val)
	}

	exLower, exUpper := true, true
	lefti, righti := i-1, i+3
	if r.val[lefti] == '=' {
		exLower = false
		lefti--
	}
	if r.val[righti] == '=' {
		exUpper = false
		righti++
	}

	var err error
	if lower, err = strconv.ParseFloat(r.val[:lefti], 64); err != nil {
		return nil, err
	}
	if upper, err = strconv.ParseFloat(r.val[righti:], 64); err != nil {
		return nil, err
	}

	return &rangeParam{
		lowerf:  lower,
		upperf:  upper,
		exLower: exLower,
		exUpper: exUpper,
	}, nil
}
