package util

import (
	"errors"
	"fmt"
	"slices"
)

const (
	BeDict BeType = 'd'
	BeList BeType = 'l'
	BeInt  BeType = 'i'
	BeStr  BeType = 's'
)

type BeType = byte

type Be struct {
	Tag  BeType
	Dict *map[string]Be
	List []Be
	Int  int64
	Str  []byte
}

type decoder struct {
	data []byte
	pos  uint64
}

func (d *decoder) decode() (*Be, error) {
	if err := d.checkPos(); err != nil {
		return nil, err
	}

	tag := d.data[d.pos]
	d.pos++

	switch {
	case tag == BeDict:
		be := Be{Tag: tag}
		beDict := make(map[string]Be)
		be.Dict = &beDict
		if err := d.checkPos(); err != nil {
			return nil, err
		}
		for d.data[d.pos] != 'e' {
			beKey, err := d.decodeBeStr()
			if err != nil {
				return nil, err
			}
			beValue, err := d.decode()
			if err != nil {
				return nil, err
			}
			beDict[string(beKey.Str)] = *beValue
		}
		d.pos++
		return &be, nil

	case tag == BeList:
		be := Be{Tag: tag}
		be.List = make([]Be, 0)
		if err := d.checkPos(); err != nil {
			return nil, err
		}
		for d.data[d.pos] != 'e' {
			innerBe, err := d.decode()
			if err != nil {
				return nil, err
			}
			be.List = append(be.List, *innerBe)
		}
		d.pos++
		return &be, nil

	case tag >= '0' && tag <= '9':
		d.pos--
		return d.decodeBeStr()

	case tag == BeInt:
		return d.decodebeInt()

	default:
		return nil, errors.New(fmt.Sprint("wrong tag, got:", tag))
	}
}

func (d *decoder) decodebeInt() (*Be, error) {
	var be Be = Be{Tag: BeInt}
	sign := false
	switch {
	case d.data[d.pos] >= '0' && d.data[d.pos] <= '9':
		be.Int = int64(d.data[d.pos] - '0')
	case d.data[d.pos] == '-':
		sign = true
	}
	d.pos++

	for d.data[d.pos] >= '0' && d.data[d.pos] <= '9' {
		if err := d.checkPos(); err != nil {
			return nil, err
		}
		be.Int = be.Int*10 + int64(d.data[d.pos]-'0')
		d.pos++
	}

	if sign {
		be.Int *= -1
	}

	if err := d.checkIfE(); err != nil {
		return nil, err
	}
	return &be, nil
}

func (d *decoder) decodeBeStr() (*Be, error) {
	tag := d.data[d.pos]
	var length uint64 = uint64(tag - '0')
	d.pos++
	for d.data[d.pos] >= '0' && d.data[d.pos] <= '9' {
		if err := d.checkPos(); err != nil {
			return nil, err
		}
		length = length*10 + uint64(d.data[d.pos]-'0')
		d.pos++
	}

	if err := d.checkPos(); err != nil {
		return nil, err
	}

	if d.data[d.pos] != ':' {
		return nil, errors.New(fmt.Sprintf("expected: %v got: %v pos: %v", ':', d.data[d.pos], d.pos))
	}

	if d.pos+length > uint64(len(d.data)) {
		return nil, errors.New(fmt.Sprintf("len(d.data) %v <= d.pos %v + length %v", len(d.data), d.pos, length))
	}

	var be Be = Be{Tag: BeStr}
	d.pos++
	strByte := slices.Clone(d.data[d.pos : d.pos+length])
	be.Str = strByte
	d.pos = d.pos + length
	return &be, nil
}

func (d *decoder) checkPos() error {
	if len(d.data) == 0 {
		return errors.New("empty []byte")
	}

	if uint64(len(d.data)) <= d.pos {
		return errors.New("len(d.data) <= d.pos")
	}

	return nil
}

func (d *decoder) checkIfE() error {
	if err := d.checkPos(); err != nil {
		return err
	}

	tag := d.data[d.pos]

	if tag != 'e' {
		return errors.New(fmt.Sprintf("wrong tag, got: %v, expected: %v, pos: %v", tag, 'e', d.pos))
	}

	d.pos++
	return nil
}

func Decode(b []byte) (*Be, error) {
	d := decoder{b, 0}
	return d.decode()
}

func (be *Be) String() string {
	switch {
	case be.Tag == BeDict:
		str := "{"
		first := true
		for k, v := range *be.Dict {
			if first {
				str += fmt.Sprintf("\"%v\": %v", k, v.String())
				first = false
			} else {
				str += fmt.Sprintf(",\"%v\": %v", k, v.String())
			}
		}
		str += "}"
		return str

	case be.Tag == BeList:
		str := "["
		first := true
		for _, v := range be.List {
			if first {
				str += fmt.Sprintf("%v", v.String())
				first = false
			} else {
				str += fmt.Sprintf(",%v", v.String())
			}
		}
		str += "]"
		return str

	case be.Tag == BeStr:
		printable := true
		for _, b := range be.Str {
			if b < 32 || b > 126 {
				printable = false
				break
			}
		}
		if printable {
			return fmt.Sprintf("\"%s\"", string(be.Str))
		}
		return fmt.Sprintf("\"<binary %d bytes>\"", len(be.Str))

	case be.Tag == BeInt:
		return fmt.Sprintf("%v", be.Int)

	default:
		return "+++bad Be+++"
	}
}

func GetIndeces(key string, b []byte) (int, int, error) {
	d := decoder{b, 0}
	return d.getIndeces(key)
}

func (d *decoder) getIndeces(key string) (int, int, error) {
	if err := d.checkPos(); err != nil {
		return -1, -1, err
	}

	tag := d.data[d.pos]
	d.pos++

	switch {
	case tag == BeDict:
		if err := d.checkPos(); err != nil {
			return -1, -1, err
		}
		for d.data[d.pos] != 'e' {
			beKey, err := d.decodeBeStr()
			if err != nil {
				return -1, -1, err
			}

			begin := int(d.pos)
			beg, end, err := d.getIndeces(key)
			if string(beKey.Str) == key && (err == nil || err.Error() == "not found") {
				return begin, int(d.pos), nil
			}
			if err != nil && err.Error() != "not found" {
				return -1, -1, err
			}
			if err == nil {
				return beg, end, nil
			}

		}
		d.pos++
		return -1, -1, errors.New("not found")

	case tag == BeList:
		if err := d.checkPos(); err != nil {
			return -1, -1, err
		}
		for d.data[d.pos] != 'e' {
			beg, end, err := d.getIndeces(key)
			if err != nil {
				if err.Error() == "not found" {
					continue
				}
				return -1, -1, err
			}
			return beg, end, nil
		}
		d.pos++
		return -1, -1, errors.New("not found")

	case tag >= '0' && tag <= '9':
		d.pos--
		d.decodeBeStr()
		return -1, -1, errors.New("not found")

	case tag == BeInt:
		d.decodebeInt()
		return -1, -1, errors.New("not found")

	default:
		return -1, -1, errors.New(fmt.Sprint("wrong tag, got:", tag))
	}
}
