package entity

type Frame struct {
	FrameMeta

	Payload []byte
}

type FrameMeta struct {
	JobID     string `json:"jobID"`
	ID        string `json:"id"`
	Sequence  int32  `json:"seq"`
	Rows      int32  `json:"rows"`
	Cols      int32  `json:"cols"`
	FrameType int32  `json:"frame_type"`
}
