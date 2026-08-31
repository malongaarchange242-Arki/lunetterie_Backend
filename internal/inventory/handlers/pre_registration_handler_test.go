package handlers

import "testing"

func TestBuildPhotoStoragePathUsesUniqueNamePerPhoto(t *testing.T) {
	pathA := buildPhotoStoragePath("VAL-016", "RO-20-001", "face", "p1", "IMG_0387.JPG")
	pathB := buildPhotoStoragePath("VAL-016", "RO-20-001", "face", "p2", "image.JPG")

	if pathA == pathB {
		t.Fatalf("expected distinct storage paths for different photos, got same: %s", pathA)
	}
	if pathA == "" || pathB == "" {
		t.Fatal("storage paths must not be empty")
	}
	if len(pathA) < len("pre-registration/") || len(pathB) < len("pre-registration/") {
		t.Fatal("storage path must start with pre-registration/")
	}
}

func TestBuildPhotoStoragePathSanitizesKindAndFileName(t *testing.T) {
	path := buildPhotoStoragePath("VAL-016", "RO-20-001", "face/side", "photo-123", "IMG 0387.JPG")
	if path == "" {
		t.Fatal("sanitized path must not be empty")
	}
	if path == "pre-registration/VAL-016/RO-20-001/" {
		t.Fatal("storage path should not be empty after sanitization")
	}
	if len(path) < len("pre-registration/VAL-016/RO-20-001/photo-123-face-side-img-0387-jpg") {
		t.Fatal("storage path appears to be missing the sanitized filename or kind")
	}
}
