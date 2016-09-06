package disk

import (
	"encoding/binary"
	"fmt"
	"io"
)

type block struct {
	magic byte
	key   []byte
	value []byte

	rbuf, wbuf [4]byte
}

func (b *block) size() int64 {
	return int64(len(b.key) + len(b.value) + 8)
}

func (b *block) keyLen() uint32 {
	return uint32(len(b.key))
}

func (b *block) valueLen() uint32 {
	return uint32(len(b.value))
}

func (b *block) writeTo(w io.Writer) (err error) {
	if err = b.writeUint32(w, b.keyLen()); err != nil {
		return
	}

	// what if fails here? corrupted
	if err = writeBytes(w, b.key); err != nil {
		return err
	}

	if err = b.writeUint32(w, b.valueLen()); err != nil {
		return
	}

	if err = writeBytes(w, b.value); err != nil {
		return
	}

	return
}

func (b *block) readFrom(r io.Reader, buf []byte) error {
	keyLen, err := b.readUint32(r)
	if err != nil {
		return err
	}

	if keyLen > maxBlockSize {
		return ErrSegmentCorrupt
	}

	if keyLen > 0 {
		if err = readBytes(r, buf[:int(keyLen)]); err != nil {
			return err
		}
		if len(b.key) < int(keyLen) {
			b.key = make([]byte, int(keyLen))
		}
		copy(b.key, buf[:int(keyLen)])
	}

	valueLen, err := b.readUint32(r)
	if err != nil {
		return err
	}

	if valueLen > maxBlockSize {
		return ErrSegmentCorrupt
	}

	if err = readBytes(r, buf[:int(valueLen)]); err != nil {
		return err
	}
	if len(b.value) < int(valueLen) {
		b.value = make([]byte, int(valueLen))
	}
	copy(b.value, buf[:int(valueLen)])

	return nil
}

func (b *block) readUint32(r io.Reader) (uint32, error) {
	if err := readBytes(r, b.rbuf[:]); err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint32(b.rbuf[:]), nil
}

func (b *block) writeUint32(w io.Writer, v uint32) error {
	binary.BigEndian.PutUint32(b.wbuf[:], v)
	return writeBytes(w, b.wbuf[:])
}

func writeBytes(w io.Writer, b []byte) error {
	n, err := w.Write(b)
	if err != nil {
		return err
	}

	if n != len(b) {
		return fmt.Errorf("short write. exp %d, got %d", len(b), n)
	}
	return nil
}

func readBytes(r io.Reader, b []byte) error {
	n, err := io.ReadAtLeast(r, b, len(b))
	if err != nil {
		return err
	}

	if n != len(b) {
		return fmt.Errorf("bad read. exp %v, got %v", len(b), n)
	}
	return nil
}