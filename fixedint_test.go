package fixedintpref

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"testing"
)

var msgs = [][]byte{
	[]byte("foo"),
	[]byte("badkfja akfjkadjf  akdfja ka kadjfa"),
	[]byte(`{"FileSize":22897614,"xxxxxxMIMEType":"image/gif","FileType":"GIF","PixelHeight":480,"FileExtension":"gif","Count":130,"PixelWidth":852}`),
}

func TestLengthPref(t *testing.T) {

	for _, b := range msgs {
		pref := make([]byte, 16)
		binary.LittleEndian.PutUint16(pref, uint16(len(b)))
		buf := bytes.NewBuffer(pref)
		buf.Write(b)

		fr := NewFixedintReader(buf, 16)
		if i, err := fr.NextMsgLen(); err != nil {
			t.Fatal(err)
		} else if i != len(b) {
			t.Fatal(fmt.Errorf("NextMsgLen returned [%d] for buffer size: [%d]", i, len(b)))
		}

	}

}

func TestMsgStream(t *testing.T) {

	var buf = &bytes.Buffer{}
	for _, b := range msgs {
		pref := make([]byte, 16)
		binary.LittleEndian.PutUint16(pref, uint16(len(b)))
		buf.Write(pref)
		buf.Write(b)
	}
	fr := NewFixedintReader(buf, 16)
	for _, b := range msgs {
		if bstream, err := fr.ReadMsg(); err != nil {
			t.Fatal(err)
		} else if !bytes.Equal(bstream, b) {
			t.Error(
				fmt.Errorf(
					"bytes not equal, source:[%s], from stream:[%s]",
					b,
					bstream,
				),
			)
		}
	}
}
