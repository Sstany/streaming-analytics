package framer

import (
	"corgiAnalytics/runner/internal/entity"

	"github.com/google/uuid"
	"gocv.io/x/gocv"
)

type VideoFramer struct {
	jobID    string
	stream   *gocv.VideoCapture
	img      *gocv.Mat
	sequence int
}
type Framer interface {
	Start(entity.Job) error
	Next() (*entity.Frame, error)
	Stop() error
}

func NewVideoFramer() Framer {
	return &VideoFramer{}
}

func (r *VideoFramer) Start(job entity.Job) error {
	video, err := gocv.OpenVideoCapture(job.URL)
	if err != nil {
		return nil
	}

	r.stream = video
	r.jobID = job.ID
	img := gocv.NewMat()
	r.img = &img

	return nil
}

func (r *VideoFramer) Stop() error {
	if err := r.stream.Close(); err != nil {
		return err
	}

	if err := r.img.Close(); err != nil {
		return err
	}

	return nil
}

func (r *VideoFramer) Next() (*entity.Frame, error) {
	if ok := r.stream.Read(r.img); !ok {
		r.Stop()
		return nil, nil
	}

	r.sequence++

	return &entity.Frame{
		FrameMeta: entity.FrameMeta{
			JobID:     r.jobID,
			ID:        uuid.NewString(),
			Sequence:  int32(r.sequence),
			Rows:      int32(r.img.Rows()),
			Cols:      int32(r.img.Cols()),
			FrameType: int32(r.img.Type()),
		},
		Payload: r.img.ToBytes(),
	}, nil
}
