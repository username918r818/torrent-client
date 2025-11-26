package util_test

import (
	"testing"

	"github.com/username918r818/torrent-client/util"
)

func TestEncoding(t *testing.T) {
	data := []byte{
		0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf1,
		0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x12,
		0x34, 0x56, 0x78, 0x9a,
	}

	encodedUrl := util.EncodeUrl(data)
	expected := "%124Vx%9A%BC%DE%F1%23Eg%89%AB%CD%EF%124Vx%9A"

	if encodedUrl != expected {
		t.Errorf("expected:\n%v\ngot:\n%v", expected, encodedUrl)
	}
}
