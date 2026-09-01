package main
import (
  "fmt"
  "os"
  "github.com/joho/godotenv"
  "github.com/lunetterie/backend/internal/inventory/services"
)
func main(){
  _ = godotenv.Load()
  fmt.Printf("URL=%q\nKEY=%q\n", os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_SERVICE_ROLE_KEY"))
  s := services.NewStorageService(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_SERVICE_ROLE_KEY"), "glasses-photos")
  url, err := s.Upload("debug-check/test.jpg", []byte("dummy-jpg-content"), "image/jpeg")
  fmt.Printf("UPLOAD_URL=%s\nERR=%v\n", url, err)
}
