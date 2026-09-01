package services

import (
	"bytes"
	"image/png"
	"testing"
)

func TestGenerateBarcodeImageProducesPNG(t *testing.T) {
	svc := &BarcodeService{}

	img, err := svc.GenerateBarcodeImage("LUN-CNG-000001")
	if err != nil {
		t.Fatalf("GenerateBarcodeImage returned error: %v", err)
	}
	if len(img) == 0 {
		t.Fatal("GenerateBarcodeImage returned empty image payload")
	}
	if !bytes.Equal(img[:8], []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}) {
		t.Fatal("GenerateBarcodeImage did not return a valid PNG header")
	}
	if _, err := png.Decode(bytes.NewReader(img)); err != nil {
		t.Fatalf("GenerateBarcodeImage returned invalid PNG: %v", err)
	}
}
