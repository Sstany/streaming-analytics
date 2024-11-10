package entity

type Frame struct {
	Id        string
	Sequence  int32
	Rows      int32
	Cols      int32
	FrameType int32
	Payload   []byte
}
