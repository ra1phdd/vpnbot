package constants

const (
	ErrGetDataFromCache      = "Failed to get data from cache"
	ErrSetDataToCache        = "Failed to set data to cache"
	ErrDeleteDataFromCache   = "Failed to delete data from cache"
	ErrGetDataFromDB         = "Failed to get data from db"
	ErrExecQueryFromDB       = "Failed to execute query from db"
	ErrRowsScanFromDB        = "Failed to rows scan in struct"
	ErrUnmarshalDataFromJSON = "Failed to unmarshal data from json"
	ErrMarshalDataToJSON     = "Failed to marshal data to json"
	ErrBeginTx               = "Failed to begin transaction"
	ErrCommitTx              = "Failed to commit transaction"
)
