package entity

import "image"

type Result struct {
	ID       string    `json:"id"`
	Predicts []Predict `json:"predicts"`
}

type Predict struct {
	Class     string          `json:"class"`
	Rectangle image.Rectangle `json:"rectangle"`
}
