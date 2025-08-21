package dataset

type DatasetService interface {
	CreateDataset(filename string) (int64, error)
}
