package ports

// BarcodeGenerator decouple le domaine de la sequence PostgreSQL et du format Code128.
type BarcodeGenerator interface {
	GenerateBarcode() (string, error)
}
