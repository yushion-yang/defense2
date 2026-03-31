package weights

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
)

// Load reads a WeightFile from r using the DTLM binary format.
//
// Binary layout:
//
//	Magic(4B "DTLM") | Version(4B uint32=1) | HeaderSize(4B) | JSON Config(HeaderSize bytes)
//	TensorCount(4B uint32)
//	Per tensor: NameLen(4B) | Name(NB) | NDim(4B) | Shape(NDim×4B) | Data(∏shape × 4B float32)
func Load(r io.Reader) (*WeightFile, error) {
	// Read and verify magic
	var magicBuf [4]byte
	if _, err := io.ReadFull(r, magicBuf[:]); err != nil {
		return nil, fmt.Errorf("read magic: %w", err)
	}
	if string(magicBuf[:]) != magic {
		return nil, fmt.Errorf("bad magic: got %q, want %q", string(magicBuf[:]), magic)
	}

	// Read and verify version
	var ver uint32
	if err := binary.Read(r, binary.LittleEndian, &ver); err != nil {
		return nil, fmt.Errorf("read version: %w", err)
	}
	if ver != version {
		return nil, fmt.Errorf("unsupported version: %d", ver)
	}

	// Read JSON header
	var headerSize uint32
	if err := binary.Read(r, binary.LittleEndian, &headerSize); err != nil {
		return nil, fmt.Errorf("read header size: %w", err)
	}
	headerBuf := make([]byte, headerSize)
	if _, err := io.ReadFull(r, headerBuf); err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}

	var cfg ModelConfig
	if err := json.Unmarshal(headerBuf, &cfg); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}

	// Read tensor count
	var tensorCount uint32
	if err := binary.Read(r, binary.LittleEndian, &tensorCount); err != nil {
		return nil, fmt.Errorf("read tensor count: %w", err)
	}

	tensors := make(map[string]*TensorData, tensorCount)
	for i := uint32(0); i < tensorCount; i++ {
		td, err := readTensor(r)
		if err != nil {
			return nil, fmt.Errorf("read tensor %d: %w", i, err)
		}
		tensors[td.Name] = td
	}

	return &WeightFile{
		Config:  cfg,
		Tensors: tensors,
	}, nil
}

// readTensor reads a single tensor from r.
// Format: NameLen(4B) | Name(NB) | NDim(4B) | Shape(NDim×4B) | Data(∏shape × 4B float32)
func readTensor(r io.Reader) (*TensorData, error) {
	// Name
	var nameLen uint32
	if err := binary.Read(r, binary.LittleEndian, &nameLen); err != nil {
		return nil, fmt.Errorf("read name len: %w", err)
	}
	nameBuf := make([]byte, nameLen)
	if _, err := io.ReadFull(r, nameBuf); err != nil {
		return nil, fmt.Errorf("read name: %w", err)
	}

	// Shape
	var ndim uint32
	if err := binary.Read(r, binary.LittleEndian, &ndim); err != nil {
		return nil, fmt.Errorf("read ndim: %w", err)
	}
	shape := make([]int, ndim)
	totalElems := 1
	for i := uint32(0); i < ndim; i++ {
		var dim uint32
		if err := binary.Read(r, binary.LittleEndian, &dim); err != nil {
			return nil, fmt.Errorf("read shape[%d]: %w", i, err)
		}
		shape[i] = int(dim)
		totalElems *= int(dim)
	}

	// Data as raw bytes, then reinterpret as float32
	dataBuf := make([]byte, totalElems*4)
	if _, err := io.ReadFull(r, dataBuf); err != nil {
		return nil, fmt.Errorf("read data: %w", err)
	}
	data := make([]float32, totalElems)
	for i := 0; i < totalElems; i++ {
		bits := binary.LittleEndian.Uint32(dataBuf[i*4 : (i+1)*4])
		data[i] = math.Float32frombits(bits)
	}

	return &TensorData{
		Name:  string(nameBuf),
		Shape: shape,
		Data:  data,
	}, nil
}

// Writer writes a WeightFile in the DTLM binary format.
type Writer struct {
	w           io.Writer
	tensorCount uint32
	tensorBuf   []byte // buffered tensor data, flushed on Finish
}

// NewWriter creates a Writer that writes to w.
func NewWriter(w io.Writer) *Writer {
	return &Writer{w: w}
}

// WriteHeader writes the magic, version, and JSON config header.
// Must be called exactly once before any WriteTensor calls.
func (wr *Writer) WriteHeader(cfg ModelConfig) error {
	// Magic
	if _, err := wr.w.Write([]byte(magic)); err != nil {
		return fmt.Errorf("write magic: %w", err)
	}

	// Version
	if err := binary.Write(wr.w, binary.LittleEndian, version); err != nil {
		return fmt.Errorf("write version: %w", err)
	}

	// JSON header
	headerBytes, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	if err := binary.Write(wr.w, binary.LittleEndian, uint32(len(headerBytes))); err != nil {
		return fmt.Errorf("write header size: %w", err)
	}
	if _, err := wr.w.Write(headerBytes); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	return nil
}

// WriteTensor buffers a tensor to be written on Finish.
func (wr *Writer) WriteTensor(td *TensorData) error {
	wr.tensorCount++
	b, err := writeTensor(td)
	if err != nil {
		return err
	}
	wr.tensorBuf = append(wr.tensorBuf, b...)
	return nil
}

// Finish writes the tensor count and all buffered tensor data.
func (wr *Writer) Finish() error {
	// Tensor count
	if err := binary.Write(wr.w, binary.LittleEndian, wr.tensorCount); err != nil {
		return fmt.Errorf("write tensor count: %w", err)
	}
	// Tensor data
	if len(wr.tensorBuf) > 0 {
		if _, err := wr.w.Write(wr.tensorBuf); err != nil {
			return fmt.Errorf("write tensors: %w", err)
		}
	}
	return nil
}

// writeTensor serializes a single tensor to bytes.
func writeTensor(td *TensorData) ([]byte, error) {
	nameBytes := []byte(td.Name)
	ndim := len(td.Shape)

	totalElems := 1
	for _, d := range td.Shape {
		totalElems *= d
	}

	// Pre-allocate: nameLen(4) + name + ndim(4) + shape(ndim*4) + data(totalElems*4)
	size := 4 + len(nameBytes) + 4 + ndim*4 + totalElems*4
	buf := make([]byte, 0, size)

	// Name length + name
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(nameBytes)))
	buf = append(buf, nameBytes...)

	// NDim + shape
	buf = binary.LittleEndian.AppendUint32(buf, uint32(ndim))
	for _, d := range td.Shape {
		buf = binary.LittleEndian.AppendUint32(buf, uint32(d))
	}

	// Data
	for _, v := range td.Data {
		buf = binary.LittleEndian.AppendUint32(buf, math.Float32bits(v))
	}

	return buf, nil
}
